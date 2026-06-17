/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/admin/users
** File description:
** Admin-side user management. A single PATCH route applies partial updates so
** every management action (block, change role, rename, ...) goes through one
** endpoint instead of one route per field.
 */

package admin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
	"github.com/matteoepitech/flick/internal/api/routes"
	"github.com/matteoepitech/flick/internal/api/routes/account"
)

// UpdateUserRequest: The PATCH payload. Every mutable field is a pointer so a
// missing field (nil) means "leave untouched", matching PATCH semantics. The
// admin token is read from the Authorization header, not the body.
type UpdateUserRequest struct {
	Username *string `json:"username" validate:"omitempty,min=1"`
	Email    *string `json:"email" validate:"omitempty,email"`
	Role     *string `json:"role" validate:"omitempty,oneof=user admin"`
	Password *string `json:"password" validate:"omitempty,min=8"`
	Blocked  *bool   `json:"blocked"`
}

// AdminUserResponse: The updated user without the password hash.
type AdminUserResponse struct {
	ID        pgtype.UUID        `json:"id"`
	Username  string             `json:"username"`
	Email     string             `json:"email"`
	Role      database.UserRole  `json:"role"`
	Blocked   bool               `json:"blocked"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

// writeAuthError: Writes the standard client-facing message for a RequireAdmin
// failure, keyed on the HTTP status, so internal error details never leak.
//
// Params:
// - w (http.ResponseWriter): The response writer.
// - status (int): The status returned by RequireAdmin.
func writeAuthError(w http.ResponseWriter, status int) {
	switch status {
	case http.StatusUnauthorized:
		routes.WriteError(w, http.StatusUnauthorized, "Invalid token")
	case http.StatusForbidden:
		routes.WriteError(w, http.StatusForbidden, "Admin privileges required")
	default:
		routes.WriteError(w, http.StatusInternalServerError, "Cannot authorize request")
	}
}

// ListUsersHandler: Returns every user (admin-only) for the management UI.
// The password hash is never included in the response.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func ListUsersHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, status, err := RequireAdmin(r.Context(), queries, TokenFromHeader(r)); err != nil {
			writeAuthError(w, status)
			return
		}

		users, err := queries.ListUsers(r.Context())
		if err != nil {
			routes.WriteError(w, http.StatusInternalServerError, "Cannot list users")
			return
		}

		out := make([]AdminUserResponse, 0, len(users))
		for _, user := range users {
			out = append(out, AdminUserResponse{
				ID:        user.ID,
				Username:  user.Username,
				Email:     user.Email,
				Role:      user.Role,
				Blocked:   user.Blocked,
				CreatedAt: user.CreatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(out)
	}
}

// UpdateUserHandler: Partially updates the user identified by the id path value.
//
// Params:
// - queries (*database.Queries): The database queries.
//
// Returns:
// - result1 (http.HandlerFunc): The handler function.
func UpdateUserHandler(queries *database.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var targetID pgtype.UUID
		if err := targetID.Scan(r.PathValue("id")); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid user id")
			return
		}

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		var request UpdateUserRequest
		validate := validator.New()

		if err := decoder.Decode(&request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
			return
		}
		if err := validate.Struct(request); err != nil {
			routes.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}

		admin, status, err := RequireAdmin(r.Context(), queries, TokenFromHeader(r))
		if err != nil {
			writeAuthError(w, status)
			return
		}

		// check to ensure that the admin doesn't lock himself..
		if admin.ID == targetID {
			if request.Blocked != nil && *request.Blocked {
				routes.WriteError(w, http.StatusForbidden, "You cannot block your own account")
				return
			}
			if request.Role != nil && *request.Role != string(database.UserRoleAdmin) {
				routes.WriteError(w, http.StatusForbidden, "You cannot revoke your own admin role")
				return
			}
		}

		params := database.UpdateUserParams{
			ID:       targetID,
			Username: request.Username,
			Email:    request.Email,
			Blocked:  request.Blocked,
		}
		if request.Role != nil {
			role := database.UserRole(*request.Role)
			params.Role = &role
		}

		if request.Password != nil {
			hashed := account.HashPassword(*request.Password)
			params.PasswordHash = &hashed
		}

		user, err := queries.UpdateUser(r.Context(), params)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				routes.WriteError(w, http.StatusNotFound, "User not found")
				return
			}
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				if pgErr.ConstraintName == "users_username_key" {
					routes.WriteError(w, http.StatusConflict, "Username already taken")
				} else {
					routes.WriteError(w, http.StatusConflict, "Email already registered")
				}
				return
			}
			routes.WriteError(w, http.StatusInternalServerError, "Cannot update user")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AdminUserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			Blocked:   user.Blocked,
			CreatedAt: user.CreatedAt,
		})
	}
}
