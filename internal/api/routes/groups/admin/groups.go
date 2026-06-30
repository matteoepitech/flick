/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/admin/groups
** File description:
** Admin-side group CRUD handlers (list, create, rename, delete).
 */

package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
)

// ListGroupsHandler: Returns every group (admin-only) for the management UI.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func ListGroupsHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, status, err := auth.RequireAdmin(r.Context(), queries, auth.GetTokenFromHTTPRequest(r)); err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		groups, err := queries.ListGroups(r.Context())
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot list groups")
			return
		}

		out := make([]AdminGroupResponse, 0, len(groups))
		for _, group := range groups {
			out = append(out, AdminGroupResponse{
				ID:        group.ID,
				Name:      group.Name,
				CreatedAt: group.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
	}
}

// CreateGroupHandler: Creates a group (admin-only) from the request body and
// returns the created group.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func CreateGroupHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminUser, status, err := auth.RequireAdmin(r.Context(), queries, auth.GetTokenFromHTTPRequest(r))
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var req CreateGroupRequest
		validate := validator.New()

		if err := decoder.Decode(&req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		group, err := queries.CreateGroup(r.Context(), req.Name)
		if err != nil {
			logging.LogInfoError("Cannot create group %q: %v", req.Name, err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create group")
			return
		}

		logging.LogInfoSuccess("Admin %q created group %q (%s)", adminUser.Username, group.Name, group.ID.String())

		out := AdminGroupResponse{
			ID:        group.ID,
			Name:      group.Name,
			CreatedAt: group.CreatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(out)
	}
}

// DeleteGroupHandler: Deletes the group identified by the id path value
// (admin-only).
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func DeleteGroupHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminUser, status, err := auth.RequireAdmin(r.Context(), queries, auth.GetTokenFromHTTPRequest(r))
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var id pgtype.UUID
		if err := id.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		if err := queries.DeleteGroup(r.Context(), id); err != nil {
			logging.LogInfoError("Cannot delete group %q: %v", id.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot delete group")
			return
		}

		logging.LogInfoSuccess("Admin %q deleted group %q", adminUser.Username, id.String())

		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateGroupHandler: Renames the group identified by the id path value
// (admin-only) and returns the updated group.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func UpdateGroupHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminUser, status, err := auth.RequireAdmin(r.Context(), queries, auth.GetTokenFromHTTPRequest(r))
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var id pgtype.UUID
		if err := id.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var req UpdateGroupRequest
		validate := validator.New()

		if err := decoder.Decode(&req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		group, err := queries.UpdateGroupName(r.Context(), database.UpdateGroupNameParams{
			ID:   id,
			Name: req.Name,
		})
		if err != nil {
			logging.LogInfoError("Cannot update group %q: %v", id.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot update group")
			return
		}

		logging.LogInfoSuccess("Admin %q renamed group %q to %q", adminUser.Username, group.ID.String(), group.Name)

		out := AdminGroupResponse{
			ID:        group.ID,
			Name:      group.Name,
			CreatedAt: group.CreatedAt,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
	}
}
