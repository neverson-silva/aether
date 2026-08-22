-- name: CreateCluster :one
INSERT INTO clusters (org_id, name, labels)
VALUES ($1, $2, $3)
RETURNING id, org_id, name, labels, created_at;

-- name: GetCluster :one
SELECT id, org_id, name, labels, created_at
FROM clusters
WHERE id = $1;

-- name: ListClustersByOrg :many
SELECT id, org_id, name, labels, created_at
FROM clusters
WHERE org_id = $1
ORDER BY name;

-- name: DeleteCluster :exec
DELETE FROM clusters
WHERE id = $1 AND org_id = $2;

-- name: ListServers :many
SELECT id, name, host, role, status, version, labels, cpu_cores, mem_total_bytes, load, last_heartbeat, cluster_id, created_at
FROM servers
ORDER BY name;

-- name: GetServer :one
SELECT id, name, host, role, status, version, labels, cpu_cores, mem_total_bytes, load, last_heartbeat, cluster_id, created_at
FROM servers
WHERE id = $1;

-- name: SetServerCluster :exec
UPDATE servers
SET cluster_id = $2
WHERE id = $1;

-- name: DeleteServer :exec
DELETE FROM servers
WHERE id = $1;

-- name: GetRegistrySettings :one
SELECT id, enabled, host, port, container_id, status
FROM registry_settings
WHERE id = 1;

-- name: UpsertRegistrySettings :one
INSERT INTO registry_settings (id, enabled)
VALUES (1, $1)
ON CONFLICT (id) DO UPDATE SET enabled = EXCLUDED.enabled
RETURNING id, enabled, host, port, container_id, status;

-- name: CreateServerToken :exec
INSERT INTO server_tokens (token_hash)
VALUES ($1);
