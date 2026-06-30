/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/login
** File description:
** Login route
 */

package account

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/memberships"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SessionDuration is the session lifetime before the token expires.
const SessionDuration = 7 * 24 * time.Hour // 7 days

// LoginRequest structure.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse: The JSON body returned on a successful login.
type LoginResponse struct {
	Token     string             `json:"token"`
	ExpiresAt pgtype.Timestamptz `json:"expires_at"`
	User      RegisterResponse   `json:"user"`
}

// LoginHandler: Login function route.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func LoginHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request LoginRequest
		validate := validator.New()

		if err := decoder.Decode(&request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		user, err := queries.GetUserByEmail(r.Context(), request.Email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
			} else {
				routes.WriteError(w, http.StatusInternalServerError, "Cannot be logged in")
			}
			return
		}
		if verifyPassword(request.Password, user.PasswordHash) == false {
			routes.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		token, err := auth.GenerateToken()
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create session")
			return
		}

		session, err := queries.CreateSession(r.Context(), database.CreateSessionParams{
			Token:     token,
			UserID:    user.ID,
			ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(SessionDuration), Valid: true},
		})
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot create session")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponse{
			Token:     session.Token,
			ExpiresAt: session.ExpiresAt,
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
