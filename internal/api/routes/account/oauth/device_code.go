/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/oauth/device_code
** File description:
** Device authorization flow (CLI login via the web)
 */

package oauth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/code"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/jackc/pgx/v5/pgtype"
)

// Lifetime of a device authorization before the user_code expires.
const deviceCodeDuration = 5 * time.Minute

// Seconds the CLI should wait between two polls on /device/token.
const devicePollInterval = 3

// DeviceCodeResponse: The JSON body returned when a device authorization is created.
type DeviceCodeResponse struct {
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	ExpiresIn  int    `json:"expires_in"`
	Interval   int    `json:"interval"`
}

// DeviceCodeHandler: Create a device authorization ticket. Called first by the
// CLI: it returns an opaque device_code (the CLI secret) and a short, human
// readable user_code that the user types on the activation web page.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func DeviceCodeHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		deviceCode, err := auth.GenerateToken()
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create device code")
			return
		}

		userCode := code.CodeGen()
		expiresAt := pgtype.Timestamptz{Time: time.Now().Add(deviceCodeDuration), Valid: true}

		auth, err := queries.CreateDeviceAuthorization(r.Context(), database.CreateDeviceAuthorizationParams{
			DeviceCode: deviceCode,
			UserCode:   userCode,
			ExpiresAt:  expiresAt,
		})
		if err != nil {
			logging.LogInfoError("Cannot create device authorization: %v", err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create device authorization")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(DeviceCodeResponse{
			DeviceCode: auth.DeviceCode,
			UserCode:   auth.UserCode,
			ExpiresIn:  int(deviceCodeDuration.Seconds()),
			Interval:   devicePollInterval,
		})
	}
}
