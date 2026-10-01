-- name: CreateDatabase :one
INSERT INTO databases (org_id, project_id, environment_id, name, engine, version, port, internal_port, public_access, external_port, data_volume, data_volume_target, db_name, db_user, pass_enc, cpus, mem_mb, storage_mb)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, FALSE, 0, '', '', $9, $10, $11, $12, $13, $14)
RETURNING id, org_id, project_id, name, engine, version, port, internal_port, public_access, external_port, data_volume, data_volume_target, db_name, db_user, pass_enc, cpus, mem_mb, storage_mb, status, container_id, created_at, service_id, environment_id;

-- name: GetDatabase :one
SELECT id, org_id, project_id, name, engine, version, port, internal_port, public_access, external_port, data_volume, data_volume_target, db_name, db_user, pass_enc, cpus, mem_mb, storage_mb, status, container_id, created_at, service_id, environment_id
FROM databases
WHERE id = $1;

-- name: ListDatabasesByOrg :many
SELECT id, org_id, project_id, name, engine, version, port, internal_port, public_access, external_port, data_volume, data_volume_target, db_name, db_user, pass_enc, cpus, mem_mb, storage_mb, status, container_id, created_at, service_id, environment_id
FROM databases
WHERE org_id = $1
ORDER BY name;

-- name: UpdateDatabaseStatus :exec
UPDATE databases
SET status = $2, container_id = $3
WHERE id = $1;

-- name: UpdateDatabaseNetwork :exec
UPDATE databases
SET public_access = $2, external_port = $3
WHERE id = $1;

-- name: UpdateDatabaseDataVolume :exec
UPDATE databases
SET data_volume = $2, data_volume_target = $3
WHERE id = $1;

-- name: DeleteDatabase :exec
DELETE FROM databases
WHERE id = $1 AND org_id = $2;
