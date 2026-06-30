/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/folders
** File description:
** Folder hierarchy inside a group: members explore the tree (folders + shared
** transfers) level by level, maintainers/owners create and delete folders.
** Uploading/downloading the transfers themselves reuses the native endpoints.
 */

package groups

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/auth"
	codepkg "github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// FolderResponse: a sub-folder at the explored level.
type FolderResponse struct {
	ID   pgtype.UUID `json:"id"`
	Name string      `json:"name"`
}

// UploadResponse: a file transfer sitting at the explored level. The code is
// included so members fetch contents and download through the native endpoints,
// which enforce membership for group-bound codes.
type UploadResponse struct {
	ID        pgtype.UUID        `json:"id"`
	Code      string             `json:"code"`
	Uploader  string             `json:"uploader"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

// ExploreResponse: the contents of one folder level (sub-folders and transfers).
type ExploreResponse struct {
	Folders []FolderResponse `json:"folders"`
	Uploads []UploadResponse `json:"uploads"`
}

// CreateFolderRequest: the POST payload to create a folder. An empty parent_id
// creates the folder at the group root.
type CreateFolderRequest struct {
	Name     string `json:"name" validate:"required,min=1"`
	ParentID string `json:"parent_id" validate:"omitempty,uuid"`
}

// folderInGroup: Resolves a folder id and ensures it belongs to the group.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - groupID (pgtype.UUID): The group the folder must belong to.
// - folderID (pgtype.UUID): The folder to resolve.
//
// Returns:
// - result1 (bool): True when the folder exists and belongs to the group.
// - result2 (error): A non-nil error only on an unexpected database failure.
func folderInGroup(ctx context.Context, queries *database.Queries, groupID, folderID pgtype.UUID) (bool, error) {
	folder, err := queries.GetGroupFolderByID(ctx, folderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return folder.GroupID == groupID, nil
}

// ExploreGroupHandler: Lists the sub-folders and transfers at a given level of
// the group's folder tree (the root when no folder query is given). Accessible
// to a global admin or any member of the group.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func ExploreGroupHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var groupID pgtype.UUID
		if err := groupID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		if _, status, err := auth.RequireGroupMember(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID); err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var folderID pgtype.UUID
		if folderParam := r.URL.Query().Get("folder"); folderParam != "" {
			if err := folderID.Scan(folderParam); err != nil {
				routes.WriteError(w, http.StatusBadRequest, "Invalid folder id")
				return
			}
			ok, err := folderInGroup(r.Context(), queries, groupID, folderID)
			if err != nil {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot explore the group")
				return
			}
			if !ok {
				routes.WriteError(w, http.StatusNotFound, "Folder not found")
				return
			}
		}

		folders, err := queries.ListGroupFoldersByParent(r.Context(), database.ListGroupFoldersByParentParams{
			GroupID:  groupID,
			ParentID: folderID,
		})
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot explore the group")
			return
		}

		uploads, err := queries.ListGroupUploadsByFolder(r.Context(), database.ListGroupUploadsByFolderParams{
			GroupID:  groupID,
			FolderID: folderID,
		})
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot explore the group")
			return
		}

		out := ExploreResponse{
			Folders: make([]FolderResponse, 0, len(folders)),
			Uploads: make([]UploadResponse, 0, len(uploads)),
		}
		for _, folder := range folders {
			out.Folders = append(out.Folders, FolderResponse{ID: folder.ID, Name: folder.Name})
		}
		for _, upload := range uploads {
			if !codepkg.IsCodeAlreadyExistInList(upload.Code) {
				_ = queries.DeleteGroupUpload(r.Context(), upload.ID)
				continue
			}
			out.Uploads = append(out.Uploads, UploadResponse{
				ID:        upload.ID,
				Code:      upload.Code,
				Uploader:  upload.UploaderUsername,
				CreatedAt: upload.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
	}
}

// CreateGroupFolderHandler: Creates a folder in the group (admin or
// maintainer/owner). An empty parent_id creates it at the group root.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func CreateGroupFolderHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var groupID pgtype.UUID
		if err := groupID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		caller, status, err := auth.RequireGroupMaintainer(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID)
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var req CreateFolderRequest
		validate := validator.New()

		if err := decoder.Decode(&req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		var parentID pgtype.UUID
		if req.ParentID != "" {
			if err := parentID.Scan(req.ParentID); err != nil {
				routes.WriteError(w, http.StatusBadRequest, "Invalid parent id")
				return
			}
			ok, err := folderInGroup(r.Context(), queries, groupID, parentID)
			if err != nil {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot create folder")
				return
			}
			if !ok {
				routes.WriteError(w, http.StatusBadRequest, "Parent folder not found")
				return
			}
		}

		folder, err := queries.CreateGroupFolder(r.Context(), database.CreateGroupFolderParams{
			GroupID:  groupID,
			ParentID: parentID,
			Name:     req.Name,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				routes.WriteError(w, http.StatusConflict, "A folder with this name already exists")
				return
			}
			logging.LogInfoError("Cannot create folder %q in group %q: %v", req.Name, groupID.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create folder")
			return
		}

		logging.LogInfoSuccess("%q created folder %q (%s) in group %q", caller.Username, folder.Name, folder.ID.String(), groupID.String())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(FolderResponse{ID: folder.ID, Name: folder.Name})
	}
}

// DeleteGroupFolderHandler: Deletes a folder (admin or maintainer/owner),
// cascading its sub-folders and the transfers it contains.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func DeleteGroupFolderHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var groupID pgtype.UUID
		if err := groupID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		caller, status, err := auth.RequireGroupMaintainer(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID)
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var folderID pgtype.UUID
		if err := folderID.Scan(r.PathValue("folderId")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid folder id")
			return
		}

		ok, err := folderInGroup(r.Context(), queries, groupID, folderID)
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot delete folder")
			return
		}
		if !ok {
			routes.WriteError(w, http.StatusNotFound, "Folder not found")
			return
		}

		// Revoke the stored codes of every transfer in the folder subtree (files
		// on disk + cache) before the DB rows are cascaded away. Best-effort: a
		// missing/expired code must not block the deletion.
		if codes, err := queries.ListGroupUploadCodesInFolderTree(r.Context(), folderID); err == nil {
			for _, code := range codes {
				_ = codepkg.DeleteCode(code)
			}
		}

		if err := queries.DeleteGroupFolder(r.Context(), folderID); err != nil {
			logging.LogInfoError("Cannot delete folder %q in group %q: %v", folderID.String(), groupID.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot delete folder")
			return
		}

		logging.LogInfoSuccess("%q deleted folder %q in group %q", caller.Username, folderID.String(), groupID.String())

		w.WriteHeader(http.StatusNoContent)
	}
}
