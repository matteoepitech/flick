/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/oauth/device_token
** File description:
** Device authorization flow (polling stage)
 */

package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/go-playground/validator/v10"
)

// DeviceTokenRequest: The JSON body request.
type DeviceTokenRequest struct {
	DeviceCode string `json:"device_code" validate:"required"`
}

// DeviceTokenResponse: The JSON body reponse when the device is approved.
type DeviceTokenResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// DeviceTokenPending: The JSON body returned while the device is still waiting.
type DeviceTokenPendingResponse struct {
	Status string `json:"status"`
}

// DeviceTokenHandler: Check if the user_code has been approved or not.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func DeviceTokenHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request DeviceTokenRequest
		var validate = validator.New()

		if err := decoder.Decode(&request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		auth, err := queries.GetDeviceAuthorizationByDeviceCode(r.Context(), request.DeviceCode)
		if err != nil {
			routes.WriteError(w, http.StatusNotFound, "Invalid device code: "+err.Error())
			return
		}

		switch auth.Status {
		case database.OauthStatusPending:
			writePendingResponse(w)
			return
		case database.OauthStatusDenied:
			writeDeniedResponse(w)
			return
		case database.OauthStatusApproved:
			writeApprovedResponse(w, auth)
			return
		default:
			routes.WriteError(w, http.StatusInternalServerError, "Unknown authorization status")
			return
		}
	}
}

// writePendingResponse: Write the JSON response while the device is still waiting.
//
// Params:
// - w (http.ResponseWriter): The response writer.
func writePendingResponse(w http.ResponseWriter) {
	data, err := json.Marshal(DeviceTokenPendingResponse{Status: "pending"})
	if err != nil {
		logging.LogInfoError("Cannot encode device token pending response: %v", err)
		routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// writeDeniedResponse: Write the error response when the authorization was denied.
//
// Params:
// - w (http.ResponseWriter): The response writer.
func writeDeniedResponse(w http.ResponseWriter) {
	routes.WriteError(w, http.StatusForbidden, "Authorization denied")
}

// writeApprovedResponse: Write the JSON response when the device is approved.
//
// Params:
// - w (http.ResponseWriter): The response writer.
// - auth (database.DeviceAuthorization): The approved device authorization.
func writeApprovedResponse(w http.ResponseWriter, auth database.DeviceAuthorization) {
	if auth.SessionToken == nil {
		routes.WriteError(w, http.StatusInternalServerError, "Approved device has no session token")
		return
	}

	data, err := json.Marshal(DeviceTokenResponse{
		Token:  *auth.SessionToken,
		UserID: auth.UserID.String(),
	})
	if err != nil {
		logging.LogInfoError("Cannot encode device token approved response: %v", err)
		routes.WriteError(w, http.StatusInternalServerError, "Cannot encode response")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
