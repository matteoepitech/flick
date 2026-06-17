/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/admin/auth
** File description:
** Shared admin authorization guard for the management routes.
 */

package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/matteoepitech/flick/internal/api/database"
)

// TokenFromHeader: Extracts the bearer token from the Authorization header.
// Returns an empty string when the header is missing or malformed.
//
// Params:
// - r (*http.Request): The incoming request.
//
// Returns:
// - result1 (string): The bearer token, or "" if absent.
func TokenFromHeader(r *http.Request) string {
	const prefix = "Bearer "

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// RequireAdmin: Resolves a session token to its user and ensures that user is
// an active admin. It is the single authorization gate for management routes.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - token (string): The session token to authenticate.
//
// Returns:
// - result1 (database.User): The authenticated admin user.
// - result2 (int): The HTTP status to return when err != nil (0 on success).
// - result3 (error): A user-facing error, or nil when the user is an admin.
func RequireAdmin(ctx context.Context, queries *database.Queries, token string) (database.User, int, error) {
	session, err := queries.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return database.User{}, http.StatusUnauthorized, fmt.Errorf("Invalid or expired token")
		}
		return database.User{}, http.StatusInternalServerError, err
	}

	if session.ExpiresAt.Valid && session.ExpiresAt.Time.Before(time.Now()) {
		return database.User{}, http.StatusUnauthorized, fmt.Errorf("Invalid or expired token")
	}

	user, err := queries.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return database.User{}, http.StatusUnauthorized, fmt.Errorf("Invalid or expired token")
		}
		return database.User{}, http.StatusInternalServerError, err
	}

	if user.Role != database.UserRoleAdmin || user.Blocked {
		return database.User{}, http.StatusForbidden, fmt.Errorf("Admin privileges required")
	}

	return user, http.StatusOK, nil
}
