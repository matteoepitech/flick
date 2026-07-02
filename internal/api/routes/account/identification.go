/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/identification
** File description:
** Identification route
 */

package account

import (
	"encoding/json"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/jackc/pgx/v5/pgtype"
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
			routes.WriteError(w, http.StatusPreconditionFailed, "Could not identify anonymous user")
			return
		}

		logging.LogInfoSuccess("Created anonymous user %q", user.ID.String())

		data, err := json.Marshal(IdentifyResponse{UserID: user.ID})
		if err != nil {
			logging.LogInfoError("Cannot encode identify response: %v", err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(data)
	}
}
