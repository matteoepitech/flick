/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/admin/router
** File description:
** Method dispatchers for the admin group endpoints, so a single mux pattern
** backs each path.
 */

package admin

import (
	"net/http"

	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/routes"
)

// GroupsHandler: Routes the collection endpoint by method (GET lists, POST
// creates) so a single mux pattern backs /admin/groups.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func GroupsHandler(queries *database.Queries) http.HandlerFunc {
	list := ListGroupsHandler(queries)
	create := CreateGroupHandler(queries)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			list(w, r)
		case http.MethodPost:
			create(w, r)
		default:
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}

// GroupHandler: Routes the single-group endpoint by method (PATCH renames,
// DELETE removes) so a single mux pattern backs /admin/groups/{id}.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func GroupHandler(queries *database.Queries) http.HandlerFunc {
	update := UpdateGroupHandler(queries)
	del := DeleteGroupHandler(queries)
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPatch:
			update(w, r)
		case http.MethodDelete:
			del(w, r)
		default:
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	}
}
