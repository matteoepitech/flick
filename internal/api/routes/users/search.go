/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/users/search
** File description:
** User directory search, available to any authenticated user so a group
** maintainer can find someone to add without the full admin user list.
 */

package users

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/routes/account"
)

// UserSearchResult: A minimal user match returned by the search, kept lean so a
// non-admin caller never sees role/blocked/email-internal details beyond what is
// needed to pick someone to add to a group.
type UserSearchResult struct {
	ID       pgtype.UUID `json:"id"`
	Username string      `json:"username"`
	Email    string      `json:"email"`
}

// SearchUsersHandler: Returns users whose username or email matches the `q`
// query parameter. Available to any authenticated (non-blocked) user so a group
// maintainer can find someone to add, without exposing the full admin user list.
// Returns an empty list for queries shorter than two characters.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func SearchUsersHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, status, err := account.Authenticate(r.Context(), queries, account.TokenFromHeader(r)); err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		term := strings.TrimSpace(r.URL.Query().Get("q"))

		out := make([]UserSearchResult, 0)
		if len(term) >= 2 {
			users, err := queries.SearchUsers(r.Context(), "%"+term+"%")
			if err != nil {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot search users")
				return
			}
			for _, user := range users {
				out = append(out, UserSearchResult{
					ID:       user.ID,
					Username: user.Username,
					Email:    user.Email,
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
	}
}
