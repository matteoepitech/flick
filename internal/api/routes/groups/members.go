/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/members
** File description:
** Group membership handlers (list, add, remove, set role), usable by a global
** admin or a group's own maintainers/owners.
 */

package groups

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AddGroupMemberHandler: Adds a user to the group identified by the id path
// value. Accessible to a global admin or a maintainer/owner of that group.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func AddGroupMemberHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var groupID pgtype.UUID
		if err := groupID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		// A global admin or a maintainer/owner of this group may add members.
		caller, status, err := auth.RequireGroupMaintainer(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID)
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var req AddMemberRequest
		validate := validator.New()

		if err := decoder.Decode(&req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		var userID pgtype.UUID
		if err := userID.Scan(req.UserID); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid user id")
			return
		}

		if err := queries.AddUserToGroup(r.Context(), database.AddUserToGroupParams{
			UserID:  userID,
			GroupID: groupID,
		}); err != nil {
			logging.LogInfoError("Cannot add user %q to group %q: %v", userID.String(), groupID.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot add user to group")
			return
		}

		logging.LogInfoSuccess("%q added user %q to group %q", caller.Username, userID.String(), groupID.String())

		w.WriteHeader(http.StatusNoContent)
	}
}

// ListGroupMembersHandler: Returns the members of the group identified by the id
// path value, with each member's role inside the group. Accessible to a global
// admin or a maintainer/owner of that group. The password hash is never
// included.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func ListGroupMembersHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var groupID pgtype.UUID
		if err := groupID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		if _, status, err := auth.RequireGroupMaintainer(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID); err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		members, err := queries.ListGroupMembers(r.Context(), groupID)
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot list group members")
			return
		}

		out := make([]GroupMemberResponse, 0, len(members))
		for _, member := range members {
			out = append(out, GroupMemberResponse{
				ID:        member.ID,
				Username:  member.Username,
				Email:     member.Email,
				Role:      member.Role,
				Blocked:   member.Blocked,
				CreatedAt: member.CreatedAt,
				GroupRole: member.GroupRole,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
	}
}

// RemoveGroupMemberHandler: Removes the user identified by the userId path value
// from the group identified by the id path value. Accessible to a global admin
// or a maintainer/owner of that group.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func RemoveGroupMemberHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		var userID pgtype.UUID
		if err := userID.Scan(r.PathValue("userId")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid user id")
			return
		}

		// A maintainer/owner cannot remove themselves; only a global admin can.
		if caller.Role != database.UserRoleAdmin && caller.ID == userID {
			routes.WriteError(w, http.StatusForbidden, "You cannot remove yourself from the group")
			return
		}

		// A maintainer cannot remove an owner; only an owner or a global admin
		// may remove an owner.
		if caller.Role != database.UserRoleAdmin {
			targetRole, err := queries.GetRoleInGroup(r.Context(), database.GetRoleInGroupParams{
				UserID:  userID,
				GroupID: groupID,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot remove user from group")
				return
			}
			if err == nil && targetRole == database.GroupRoleOwner {
				callerRole, err := queries.GetRoleInGroup(r.Context(), database.GetRoleInGroupParams{
					UserID:  caller.ID,
					GroupID: groupID,
				})
				if err != nil {
					routes.WriteError(w, http.StatusInternalServerError, "Cannot remove user from group")
					return
				}
				if callerRole != database.GroupRoleOwner {
					routes.WriteError(w, http.StatusForbidden, "A maintainer cannot remove an owner")
					return
				}
			}
		}

		if err := queries.RemoveUserFromGroup(r.Context(), database.RemoveUserFromGroupParams{
			UserID:  userID,
			GroupID: groupID,
		}); err != nil {
			logging.LogInfoError("Cannot remove user %q from group %q: %v", userID.String(), groupID.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot remove user from group")
			return
		}

		logging.LogInfoSuccess("%q removed user %q from group %q", caller.Username, userID.String(), groupID.String())

		w.WriteHeader(http.StatusNoContent)
	}
}

// SetGroupMemberRoleHandler: Changes a member's role inside the group. Reserved
// to a global admin or the group owner; maintainers cannot change roles, so they
// can never promote themselves.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func SetGroupMemberRoleHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var groupID pgtype.UUID
		if err := groupID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid group id")
			return
		}

		// Only a global admin or the group owner may change roles.
		caller, status, err := auth.RequireGroupOwner(r.Context(), queries, auth.GetTokenFromHTTPRequest(r), groupID)
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		var userID pgtype.UUID
		if err := userID.Scan(r.PathValue("userId")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid user id")
			return
		}

		// An owner cannot change their own role; only a global admin can.
		if caller.Role != database.UserRoleAdmin && caller.ID == userID {
			routes.WriteError(w, http.StatusForbidden, "You cannot change your own role")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var req SetMemberRoleRequest
		validate := validator.New()

		if err := decoder.Decode(&req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(req); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		if err := queries.SetRoleInGroup(r.Context(), database.SetRoleInGroupParams{
			UserID:  userID,
			GroupID: groupID,
			Role:    database.GroupRole(req.Role),
		}); err != nil {
			logging.LogInfoError("Cannot set role of user %q in group %q: %v", userID.String(), groupID.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot set member role")
			return
		}

		logging.LogInfoSuccess("%q set user %q role to %q in group %q", caller.Username, userID.String(), req.Role, groupID.String())

		w.WriteHeader(http.StatusNoContent)
	}
}
