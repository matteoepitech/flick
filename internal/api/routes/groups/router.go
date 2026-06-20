/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/router
** File description:
** Method dispatchers for the group membership endpoints, so a single mux
** pattern backs each path.
 */

package groups

import (
	"net/http"

	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/routes"
)

// GroupMembersHandler: Routes the group members collection by method (GET lists,
// POST adds) so a single mux pattern backs /admin/groups/{id}/members.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func GroupMembersHandler(queries *database.Queries) http.HandlerFunc {
	list := ListGroupMembersHandler(queries)
	add := AddGroupMemberHandler(queries)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			list(w, r)
		case http.MethodPost:
			add(w, r)
		default:
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// GroupMemberHandler: Routes a single group member by method (PATCH sets the
// role, DELETE removes the member) so a single mux pattern backs
// /admin/groups/{id}/members/{userId}.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func GroupMemberHandler(queries *database.Queries) http.HandlerFunc {
	setRole := SetGroupMemberRoleHandler(queries)
	remove := RemoveGroupMemberHandler(queries)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			setRole(w, r)
		case http.MethodDelete:
			remove(w, r)
		default:
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}
