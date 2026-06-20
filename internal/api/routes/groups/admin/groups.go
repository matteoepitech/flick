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

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/routes/account"
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
		if _, status, err := account.RequireAdmin(r.Context(), queries, account.TokenFromHeader(r)); err != nil {
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
		if _, status, err := account.RequireAdmin(r.Context(), queries, account.TokenFromHeader(r)); err != nil {
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
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create group")
			return
		}

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
		if _, status, err := account.RequireAdmin(r.Context(), queries, account.TokenFromHeader(r)); err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var id pgtype.UUID
		if err := id.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		if err := queries.DeleteGroup(r.Context(), id); err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot delete group")
			return
		}

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
		if _, status, err := account.RequireAdmin(r.Context(), queries, account.TokenFromHeader(r)); err != nil {
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
			routes.WriteError(w, http.StatusInternalServerError, "Cannot update group")
			return
		}

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
