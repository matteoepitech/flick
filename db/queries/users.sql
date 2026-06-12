-- name: CreateUser :one
WITH first_user AS (
  SELECT NOT EXISTS (SELECT 1 FROM users) AS is_first
)
INSERT INTO users (username, email, password_hash, role)
SELECT $1, $2, $3, (CASE WHEN first_user.is_first THEN 'admin' ELSE 'user' END)::user_role
FROM first_user
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: SetUserRoleByID :exec
UPDATE users
SET role = $2
WHERE id = $1;

-- name: SetUserRoleByEmail :exec
UPDATE users
SET role = $2
WHERE email = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
