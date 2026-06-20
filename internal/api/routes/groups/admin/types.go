/*
** FLICK PROJECT, 2026
** flick/internal/api/routes/groups/admin/types
** File description:
** Request and response payloads for admin-side group management.
 */

package admin

import (
	"github.com/jackc/pgx/v5/pgtype"
)

// AdminGroupResponse: A group as returned to the management UI.
type AdminGroupResponse struct {
	ID        pgtype.UUID        `json:"id"`
	Name      string             `json:"name"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
}

// CreateGroupRequest: The POST payload to create a group. The admin token is
// read from the Authorization header, not the body.
type CreateGroupRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}

// UpdateGroupRequest: The PATCH payload to rename a group. The group id is read
// from the path, not the body.
type UpdateGroupRequest struct {
	Name string `json:"name" validate:"required,min=1"`
}
