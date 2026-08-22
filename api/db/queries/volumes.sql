-- name: GetVolumeByApp :one
SELECT id, app_id, name, mount_path
FROM app_volumes
WHERE app_id = $1 AND name = $2;

-- name: ListVolumesByApp :many
SELECT id, app_id, name, mount_path
FROM app_volumes
WHERE app_id = $1
ORDER BY name;

-- name: CreateVolume :one
INSERT INTO app_volumes (app_id, name, mount_path)
VALUES ($1, $2, $3)
RETURNING id, app_id, name, mount_path;
