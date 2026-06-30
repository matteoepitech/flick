/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/whoami
** File description:
** whoami route
 */

package account

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/memberships"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
)

// WhoamiRequest structure.
type WhoamiRequest struct {
	Token string `json:"token"`
}

// WhoamiResponse: The JSON body returned on a successful whoami.
type WhoamiResponse struct {
	User RegisterResponse `json:"user"`
}

// WhoamiHandler: whoami function route.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func WhoamiHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request WhoamiRequest
		validate := validator.New()

		if err := decoder.Decode(&request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		session, err := queries.GetSessionByToken(r.Context(), request.Token)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusUnauthorized, "Invalid token")
			} else {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot get informations")
			}
			return
		}

		user, err := queries.GetUserByID(r.Context(), session.UserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusUnauthorized, "Invalid user")
			} else {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot get informations")
			}
			return
		}

		if user.Blocked {
			routes.WriteError(w, http.StatusForbidden, "Account blocked")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(WhoamiResponse{
			User: RegisterResponse{
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				Role:      user.Role,
				CreatedAt: user.CreatedAt,
				Blocked:   user.Blocked,
				Groups:    memberships.UserGroupMemberships(r.Context(), queries, user.ID),
			},
		})
	}
}
