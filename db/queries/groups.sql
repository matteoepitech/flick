-- name: CreateGroup :one
INSERT INTO groups (name)
VALUES ($1)
RETURNING *;

-- name: GetGroupByID :one
SELECT * FROM groups
WHERE id = $1;

-- name: GetGroupByName :one
SELECT * FROM groups
WHERE name = $1;

-- name: ListGroups :many
SELECT * FROM groups
ORDER BY created_at DESC;

-- name: DeleteGroup :exec
DELETE FROM groups
WHERE id = $1;

-- name: AddUserToGroup :exec
INSERT INTO user_groups (user_id, group_id)
VALUES ($1, $2)
ON CONFLICT (user_id, group_id) DO NOTHING;

-- name: RemoveUserFromGroup :exec
DELETE FROM user_groups
WHERE user_id = $1 AND group_id = $2;

-- name: ListGroupsForUser :many
SELECT g.* FROM groups g
JOIN user_groups ug ON ug.group_id = g.id
WHERE ug.user_id = $1
ORDER BY g.name;

-- name: ListUsersInGroup :many
SELECT u.* FROM users u
JOIN user_groups ug ON ug.user_id = u.id
WHERE ug.group_id = $1
ORDER BY u.username;

-- name: ListGroupsForUserWithRole :many
SELECT g.id, g.name, g.created_at, ug.role AS group_role
FROM groups g
JOIN user_groups ug ON ug.group_id = g.id
WHERE ug.user_id = $1
ORDER BY g.name;

-- name: ListGroupMembers :many
SELECT u.id, u.username, u.email, u.role, u.blocked, u.created_at, ug.role AS group_role
FROM users u
JOIN user_groups ug ON ug.user_id = u.id
WHERE ug.group_id = $1
ORDER BY u.username;

-- name: GetRoleInGroup :one
SELECT role FROM user_groups
WHERE user_id = $1 AND group_id = $2;


-- name: SetRoleInGroup :exec
UPDATE user_groups ug
SET role = $3
WHERE ug.user_id = $1 AND ug.group_id = $2;

-- name: UpdateGroupName :one
UPDATE groups
SET name = $2
WHERE id = $1
RETURNING *;
