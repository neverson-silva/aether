-- name: GetVolumeByApp :one
SELECT v.id, v.app_id, v.name, v.mount_path, v.service_id
FROM app_volumes v
WHERE (v.service_id = (SELECT service_id FROM apps WHERE apps.id = $1) OR v.app_id = $1) AND v.name = $2;

-- name: ListVolumesByApp :many
SELECT v.id, v.app_id, v.name, v.mount_path, v.service_id
FROM app_volumes v
WHERE v.service_id = (SELECT service_id FROM apps WHERE apps.id = $1) OR v.app_id = $1
ORDER BY v.name;

-- name: CreateVolume :one
INSERT INTO app_volumes (app_id, service_id, name, mount_path)
VALUES ($1, (SELECT service_id FROM apps WHERE id = $1), $2, $3)
RETURNING id, app_id, name, mount_path, service_id;
