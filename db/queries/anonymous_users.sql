-- name: CreateAnonymousUser :one
INSERT INTO anonymous_users DEFAULT VALUES
RETURNING *;

-- name: GetAnonymousUserByID :one
SELECT * FROM anonymous_users
WHERE id = $1;

-- name: DeleteAnonymousUser :exec
DELETE FROM anonymous_users
WHERE id = $1;
