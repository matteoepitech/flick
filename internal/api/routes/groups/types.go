/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/types
** File description:
** Request and response payloads for group membership management.
 */

package groups

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/matteoepitech/flick/internal/api/database"
)

// AddMemberRequest: The POST payload to add a user to a group. The group id is
// read from the path, not the body.
type AddMemberRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// SetMemberRoleRequest: The PATCH payload to change a member's role inside a
// group. The group and user ids are read from the path, not the body.
type SetMemberRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=member maintainer owner"`
}

// GroupMemberResponse: A group member as returned to the management UI. It
// carries both the user's global role and their role inside the group, and
// never includes the password hash.
type GroupMemberResponse struct {
	ID        pgtype.UUID        `json:"id"`
	Username  string             `json:"username"`
	Email     string             `json:"email"`
	Role      database.UserRole  `json:"role"`
	Blocked   bool               `json:"blocked"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	GroupRole database.GroupRole `json:"group_role"`
}
