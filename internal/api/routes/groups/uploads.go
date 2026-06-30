/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/uploads
** File description:
** Revoking a file shared with a group: deletes the database link and the stored
** code (files on disk + cache), so the transfer is truly gone.
 */

package groups

import (
	"errors"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/auth"
	codepkg "github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// DeleteGroupUploadHandler: Revokes the transfer identified by the uploadId path
// value from the group identified by the id path value. Accessible to a global
// admin or a maintainer/owner of the group. The stored code (files on disk and
// cache) is removed too, so the share is fully revoked.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func DeleteGroupUploadHandler(queries *database.Queries) http.HandlerFunc {
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

		if _, status, err := auth.RequireGroupMaintainer(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID); err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var uploadID pgtype.UUID
		if err := uploadID.Scan(r.PathValue("uploadId")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid upload id")
			return
		}

		upload, err := queries.GetGroupUploadByID(r.Context(), uploadID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusNotFound, "Upload not found")
				return
			}
			routes.WriteError(w, http.StatusInternalServerError, "Cannot revoke the upload")
			return
		}
		if upload.GroupID != groupID {
			routes.WriteError(w, http.StatusNotFound, "Upload not found")
			return
		}

		// Revoke the stored code (files on disk + cache). Best-effort: an already
		// expired code may be gone, which must not block removing the DB link.
		_ = codepkg.DeleteCode(upload.Code)

		if err := queries.DeleteGroupUpload(r.Context(), uploadID); err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot revoke the upload")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
