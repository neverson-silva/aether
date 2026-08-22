-- name: CreateDatabase :one
INSERT INTO databases (org_id, project_id, name, engine, version, port, db_name, db_user, pass_enc, mem_mb, storage_mb)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, org_id, project_id, name, engine, version, port, db_name, db_user, pass_enc, mem_mb, storage_mb, status, container_id, created_at;

-- name: GetDatabase :one
SELECT id, org_id, project_id, name, engine, version, port, db_name, db_user, pass_enc, mem_mb, storage_mb, status, container_id, created_at
FROM databases
WHERE id = $1;

-- name: ListDatabasesByOrg :many
SELECT id, org_id, project_id, name, engine, version, port, db_name, db_user, pass_enc, mem_mb, storage_mb, status, container_id, created_at
FROM databases
WHERE org_id = $1
ORDER BY name;

-- name: UpdateDatabaseStatus :exec
UPDATE databases
SET status = $2, container_id = $3
WHERE id = $1;

-- name: UpdateDatabasePort :exec
UPDATE databases
SET port = $2
WHERE id = $1;

-- name: DeleteDatabase :exec
DELETE FROM databases
WHERE id = $1 AND org_id = $2;
