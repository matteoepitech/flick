-- name: CreateGroupUpload :one
INSERT INTO group_uploads (group_id, folder_id, code, uploader_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetGroupUploadByID :one
SELECT id, group_id, folder_id, code, uploader_id, created_at FROM group_uploads
WHERE id = $1;

-- name: ListGroupUploadsByFolder :many
SELECT gu.id, gu.code, gu.uploader_id, u.username AS uploader_username, gu.created_at
FROM group_uploads gu
JOIN users u ON u.id = gu.uploader_id
WHERE gu.group_id = $1 AND gu.folder_id IS NOT DISTINCT FROM $2
ORDER BY gu.created_at DESC;

-- name: ListGroupUploadCodesInFolderTree :many
WITH RECURSIVE subtree(id) AS (
    SELECT gf.id FROM group_folders gf WHERE gf.id = $1
    UNION ALL
    SELECT gf.id FROM group_folders gf JOIN subtree s ON gf.parent_id = s.id
)
SELECT gu.code FROM group_uploads gu WHERE gu.folder_id IN (SELECT s.id FROM subtree s);

-- name: DeleteGroupUpload :exec
DELETE FROM group_uploads
WHERE id = $1;
