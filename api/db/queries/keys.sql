-- name: CreateAPIKey :one
INSERT INTO api_keys (org_id, name, key_hash, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING id, org_id, name, key_hash, expires_at, last_used_at, created_at;

-- name: GetAPIKeyByHash :one
SELECT id, org_id, name, key_hash, expires_at, last_used_at, created_at
FROM api_keys
WHERE key_hash = $1;

-- name: ListAPIKeysByOrg :many
SELECT id, org_id, name, expires_at, last_used_at, created_at
FROM api_keys
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: TouchAPIKey :exec
UPDATE api_keys
SET last_used_at = now()
WHERE id = $1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys
WHERE id = $1 AND org_id = $2;

-- name: CreateAuditLog :exec
INSERT INTO audit_logs (org_id, user_id, action, resource_type, resource_id, details)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditLogsByOrg :many
SELECT id, org_id, user_id, action, resource_type, resource_id, details, created_at
FROM audit_logs
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2;
