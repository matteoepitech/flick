/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/account/memberships
** File description:
** Group memberships carried on the authenticated user (login/whoami).
 */

package account

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
)

// GroupMembershipResponse: A group the user belongs to, with their role inside
// it. Carried on the login/whoami user so the dashboard can show a member their
// groups and gate the "My groups" tab.
type GroupMembershipResponse struct {
	ID   pgtype.UUID        `json:"id"`
	Name string             `json:"name"`
	Role database.GroupRole `json:"role"`
}

// userGroupMemberships: Resolves the groups a user belongs to, with their role
// in each. Returns an empty (non-nil) slice when the user is in no group so the
// JSON serialises as [] rather than null.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - userID (pgtype.UUID): The user whose memberships to resolve.
//
// Returns:
// - result1 ([]GroupMembershipResponse): The user's group memberships.
func userGroupMemberships(ctx context.Context, queries *database.Queries, userID pgtype.UUID) []GroupMembershipResponse {
	rows, err := queries.ListGroupsForUserWithRole(ctx, userID)
	if err != nil {
		return []GroupMembershipResponse{}
	}

	out := make([]GroupMembershipResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, GroupMembershipResponse{
			ID:   row.ID,
			Name: row.Name,
			Role: row.GroupRole,
		})
	}
	return out
}
