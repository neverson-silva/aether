-- name: CreateMirror :one
INSERT INTO registry_mirrors (name, source, dest, dest_tls_verify, tags_filter, schedule)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, source, dest, dest_tls_verify, tags_filter, schedule, last_run, status, created_at;

-- name: GetMirror :one
SELECT id, name, source, dest, dest_tls_verify, tags_filter, schedule, last_run, status, created_at
FROM registry_mirrors
WHERE id = $1;

-- name: ListMirrors :many
SELECT id, name, source, dest, dest_tls_verify, tags_filter, schedule, last_run, status, created_at
FROM registry_mirrors
ORDER BY name;

-- name: SetMirrorStatus :exec
UPDATE registry_mirrors
SET status = $2, last_run = now()
WHERE id = $1;

-- name: DeleteMirror :exec
DELETE FROM registry_mirrors
WHERE id = $1;
