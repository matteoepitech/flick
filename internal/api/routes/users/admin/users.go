/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/users/admin/users
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

	"github.com/Flick-Corp/flick/internal/api/auth"
	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/Flick-Corp/flick/internal/api/logging"
	"github.com/Flick-Corp/flick/internal/api/routes"
	"github.com/Flick-Corp/flick/internal/api/routes/account"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
		if _, status, err := auth.RequireAdmin(r.Context(), queries, auth.GetTokenFromHTTPRequest(r)); err != nil {
			routes.WriteError(w, status, err.Error())
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

		adminUser, status, err := auth.RequireAdmin(r.Context(), queries, auth.GetTokenFromHTTPRequest(r))
		if err != nil {
			routes.WriteError(w, status, err.Error())
			return
		}

		// check to ensure that the admin doesn't lock himself..
		if adminUser.ID == targetID {
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
			logging.LogInfoError("Cannot update user %q: %v", targetID.String(), err)
			routes.WriteError(w, http.StatusInternalServerError, "Cannot update user")
			return
		}

		logging.LogInfoSuccess("Admin %q updated user %q (%s)", adminUser.Username, user.Username, user.ID.String())

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
