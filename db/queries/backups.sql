-- name: CreateBackup :one
INSERT INTO backups (org_id, database_id, app_id, path, size, kind, dest)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, database_id, app_id, path, size, kind, dest, created_at;

-- name: GetBackup :one
SELECT id, org_id, database_id, app_id, path, size, kind, dest, created_at
FROM backups
WHERE id = $1;

-- name: ListBackupsByOrg :many
SELECT id, org_id, database_id, app_id, path, size, kind, dest, created_at
FROM backups
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListBackupsByDatabase :many
SELECT id, org_id, database_id, app_id, path, size, kind, dest, created_at
FROM backups
WHERE database_id = $1
ORDER BY created_at DESC
LIMIT $2;
