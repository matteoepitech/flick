/*
** FLICK PROJECT, 2026
** flick/internal/api/auth/auth
** File description:
** Shared authorization guards for the management routes.
 */

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GetTokenFromHTTPRequest: Extracts the bearer token from the Authorization header.
// Returns an empty string when the header is missing or malformed.
//
// Params:
// - r (*http.Request): The incoming request.
//
// Returns:
// - result1 (string): The bearer token, or "" if absent.
func GetTokenFromHTTPRequest(r *http.Request) string {
	const prefix = "Bearer "

	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

// ResolveUser: Resolves a session token to its active user, without requiring
// any particular role. It is the shared first step of every authorization gate.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - token (string): The session token to authenticate.
//
// Returns:
// - result1 (database.User): The authenticated user.
// - result2 (int): The HTTP status to return when err != nil (0 on success).
// - result3 (error): A user-facing error, or nil on success.
func ResolveUser(ctx context.Context, queries *database.Queries, token string) (database.User, int, error) {
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

	if user.Blocked {
		return database.User{}, http.StatusForbidden, fmt.Errorf("Account blocked")
	}

	return user, http.StatusOK, nil
}

// RequireAdmin: Resolves a session token to its user and ensures that user is
// an active admin. It is the authorization gate for instance-wide management.
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
	user, status, err := ResolveUser(ctx, queries, token)
	if err != nil {
		return database.User{}, status, err
	}

	if user.Role != database.UserRoleAdmin {
		return database.User{}, http.StatusForbidden, fmt.Errorf("Admin privileges required")
	}

	return user, http.StatusOK, nil
}

// RequireGroupMaintainer: Authorizes a token to manage a specific group. A global
// admin always passes; otherwise the user must be a maintainer or owner of that
// group. Use it for group-scoped actions such as managing members.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - token (string): The session token to authenticate.
// - groupID (pgtype.UUID): The group the action targets.
//
// Returns:
// - result1 (database.User): The authenticated maintainer user.
// - result2 (int): The HTTP status to return when err != nil (0 on success).
// - result3 (error): A user-facing error, or nil when the user may manage the group.
func RequireGroupMaintainer(ctx context.Context, queries *database.Queries, token string, groupID pgtype.UUID) (database.User, int, error) {
	user, status, err := ResolveUser(ctx, queries, token)
	if err != nil {
		return database.User{}, status, err
	}

	if user.Role == database.UserRoleAdmin {
		return user, http.StatusOK, nil
	}

	role, err := queries.GetRoleInGroup(ctx, database.GetRoleInGroupParams{
		UserID:  user.ID,
		GroupID: groupID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return database.User{}, http.StatusForbidden, fmt.Errorf("You must be in this group to manage it")
		}
		return database.User{}, http.StatusInternalServerError, err
	}

	if role != database.GroupRoleMaintainer && role != database.GroupRoleOwner {
		return database.User{}, http.StatusForbidden, fmt.Errorf("You must be in this group to manage it")
	}

	return user, http.StatusOK, nil
}

// RequireGroupOwner: Authorizes a token to perform owner-level actions on a
// specific group, such as changing members' roles. A global admin always
// passes; otherwise the user must be the owner of that group. Maintainers are
// rejected, so they cannot promote themselves.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - token (string): The session token to authenticate.
// - groupID (pgtype.UUID): The group the action targets.
//
// Returns:
// - result1 (database.User): The authenticated owning user.
// - result2 (int): The HTTP status to return when err != nil (0 on success).
// - result3 (error): A user-facing error, or nil when the user owns the group.
func RequireGroupOwner(ctx context.Context, queries *database.Queries, token string, groupID pgtype.UUID) (database.User, int, error) {
	user, status, err := ResolveUser(ctx, queries, token)
	if err != nil {
		return database.User{}, status, err
	}

	if user.Role == database.UserRoleAdmin {
		return user, http.StatusOK, nil
	}

	role, err := queries.GetRoleInGroup(ctx, database.GetRoleInGroupParams{
		UserID:  user.ID,
		GroupID: groupID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return database.User{}, http.StatusForbidden, fmt.Errorf("Only the group owner can do this")
		}
		return database.User{}, http.StatusInternalServerError, err
	}

	if role != database.GroupRoleOwner {
		return database.User{}, http.StatusForbidden, fmt.Errorf("Only the group owner can do this")
	}

	return user, http.StatusOK, nil
}

// RequireGroupMember: Authorizes a token to access a specific group's content
// (e.g. listing or downloading its shared files). A global admin always passes;
// otherwise the user only needs to belong to that group, whatever their role.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - token (string): The session token to authenticate.
// - groupID (pgtype.UUID): The group the action targets.
//
// Returns:
// - result1 (database.User): The authenticated member user.
// - result2 (int): The HTTP status to return when err != nil (0 on success).
// - result3 (error): A user-facing error, or nil when the user belongs to the group.
func RequireGroupMember(ctx context.Context, queries *database.Queries, token string, groupID pgtype.UUID) (database.User, int, error) {
	user, status, err := ResolveUser(ctx, queries, token)
	if err != nil {
		return database.User{}, status, err
	}

	if user.Role == database.UserRoleAdmin {
		return user, http.StatusOK, nil
	}

	_, err = queries.GetRoleInGroup(ctx, database.GetRoleInGroupParams{
		UserID:  user.ID,
		GroupID: groupID,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return database.User{}, http.StatusForbidden, fmt.Errorf("You must belong to this group")
	}

	if err != nil {
		return database.User{}, http.StatusInternalServerError, err
	}

	return user, http.StatusOK, nil
}
