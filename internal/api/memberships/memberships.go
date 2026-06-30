/*
** FLICK PROJECT, 2026
** flick/internal/api/memberships
** File description:
** Group memberships carried on the authenticated user (login/whoami).
 */

package memberships

import (
	"context"

	"github.com/Flick-Corp/flick/internal/api/database"
	"github.com/jackc/pgx/v5/pgtype"
)

// GroupMembershipResponse: A group the user belongs to, with their role inside it.
type GroupMembershipResponse struct {
	ID   pgtype.UUID        `json:"id"`
	Name string             `json:"name"`
	Role database.GroupRole `json:"role"`
}

// UserGroupMemberships: Resolves the groups a user belongs to, with their role in each.
//
// Params:
// - ctx (context.Context): The request context.
// - queries (*database.Queries): The database queries.
// - userID (pgtype.UUID): The user whose memberships to resolve.
//
// Returns:
// - result1 ([]GroupMembershipResponse): The user's group memberships.
func UserGroupMemberships(ctx context.Context, queries *database.Queries, userID pgtype.UUID) []GroupMembershipResponse {
	rows, err := queries.ListUserMemberships(ctx, userID)
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
