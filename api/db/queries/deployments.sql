-- name: CreateDeployment :one
INSERT INTO deployments (
    app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13
)
RETURNING id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at;

-- name: GetDeployment :one
SELECT id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE id = $1;

-- name: GetDeploymentByApp :one
SELECT id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE app_id = $1 AND number = $2;

-- name: ListDeployments :many
SELECT id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE app_id = $1
ORDER BY number DESC
LIMIT $2;

-- name: NextDeploymentNumber :one
SELECT COALESCE(MAX(number), 0) + 1
FROM deployments
WHERE app_id = $1;

-- name: LastReadyDeployment :one
SELECT id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE app_id = $1 AND status = 'ready'
ORDER BY number DESC
LIMIT 1;

-- name: MarkDeploymentRolledBack :exec
UPDATE deployments
SET status = 'rolled_back', finished_at = now()
WHERE id = $1 AND status = 'ready';

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = $2,
    error = $3,
    image_ref = $4,
    container_id = $5,
    started_at = COALESCE($6, started_at),
    finished_at = COALESCE($7, finished_at)
WHERE id = $1
  AND status NOT IN ('ready', 'failed', 'rolled_back', 'cancelled');

-- name: ListQueuedDeployments :many
SELECT id, app_id, number, status, trigger, triggered_by, commit_sha, image_ref,
    container_id, server_id, error, env_snapshot, compose_yaml, deploy_spec, compose_hash,
    created_at, started_at, finished_at
FROM deployments
WHERE status = 'queued'
ORDER BY created_at ASC
LIMIT 10;

-- name: GetDeploymentCompose :one
SELECT compose_yaml
FROM deployments
WHERE id = $1;
