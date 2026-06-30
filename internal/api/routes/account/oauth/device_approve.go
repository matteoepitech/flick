/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/oauth/device_approve
** File description:
** Device authorization flow (web approval)
 */

package oauth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/Flick-Corp/flick/internal/api/routes/account"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// DeviceApproveRequest: The JSON body request.
// Token is the actual session token of the one who is approving.
type DeviceApproveRequest struct {
	UserCode string `json:"user_code" validate:"required"`
	Token    string `json:"token" validate:"required"`
}

// DeviceApproveResponse: The JSON body response on a successful approval.
type DeviceApproveResponse struct {
	Status string `json:"status"`
}

// DeviceApproveHandler: Approve a pending device authorization. Called by the
// web page when a logged in user confirms the user_code shown by the CLI. It
// creates a fresh session and stores its token on the device authorization so
// the CLI can fetch it on its next poll.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func DeviceApproveHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request DeviceApproveRequest
		validate := validator.New()

		if err := decoder.Decode(&request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Identify the user approving the device from their session token.
		session, err := queries.GetSessionByToken(r.Context(), request.Token)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusUnauthorized, "You are not logged in")
			} else {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot approve device")
			}
			return
		}

		// Make sure the device authorization exists and is still valid.
		if _, err := queries.GetDeviceAuthorizationByUserCode(r.Context(), request.UserCode); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusNotFound, "Invalid or expired code")
			} else {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot approve device")
			}
			return
		}

		// Create the session the CLI will end up using.
		token, err := auth.GenerateToken()
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create session")
			return
		}
		if _, err := queries.CreateSession(r.Context(), database.CreateSessionParams{
			Token:     token,
			UserID:    session.UserID,
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(account.SessionDuration), Valid: true},
		}); err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create session")
			return
		}

		if _, err := queries.ApproveDeviceAuthorization(r.Context(), database.ApproveDeviceAuthorizationParams{
			UserCode:     request.UserCode,
			UserID:       session.UserID,
			SessionToken: &token,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusConflict, "Code already used")
			} else {
				logging.LogInfoError("Cannot approve device authorization: %v", err)
				routes.WriteError(w, http.StatusInternalServerError, "Cannot approve device")
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DeviceApproveResponse{Status: "approved"})
	}
}
