/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/register
** File description:
** Register route
 */

package account

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/routes"
)

// RegisterRequest structure.
type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RegisterResponse: The JSON body returned on a successful registration.
type RegisterResponse struct {
	ID        pgtype.UUID        `json:"id"`
	Username  string             `json:"username"`
	Email     string             `json:"email"`
	Role      database.UserRole  `json:"role"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

// RegisterHandler: Register function route.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func RegisterHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			routes.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request RegisterRequest
		var validate = validator.New()

		if err := decoder.Decode(&request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		user, err := queries.CreateUser(r.Context(), database.CreateUserParams{
			Username:     request.Username,
			Email:        request.Email,
			PasswordHash: hashPassword(request.Password),
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				if pgErr.ConstraintName == "users_username_key" {
					routes.WriteError(w, http.StatusConflict, "Username already taken")
				} else {
					routes.WriteError(w, http.StatusConflict, "Email already registered")
				}
				return
			}
			routes.WriteError(w, http.StatusInternalServerError, "Cannot be registered")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(RegisterResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			CreatedAt: user.CreatedAt,
		})
	}
}
