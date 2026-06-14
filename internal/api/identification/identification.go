/*
** FLICK PROJECT, 2026
** flick/internal/api/identification/identification
** File description:
** Identification go file
 */

package identification

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/logging"
	"github.com/matteoepitech/flick/internal/api/routes"
)

// IdentifyResponse: The JSON body returned when an anonymous user is created.
type IdentifyResponse struct {
	UserID pgtype.UUID `json:"user_id"`
}

// IdentifyHandler: Create an anonymous user and return its UUID. Used by the
// CLI on first upload, when no credentials file exists yet.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func IdentifyHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		// TODO: make a check for the rate limit.

		user, err := queries.CreateAnonymousUser(r.Context())
		if err != nil {
			logging.LogInfoError("Cannot create id for anonymous user: %v", err)
			routes.WriteError(w, http.StatusPreconditionFailed, "Could not create an id")
			return
		}

		logging.LogInfoSuccess("Created anonymous user %q", user.ID.String())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(IdentifyResponse{UserID: user.ID})
	}
}
