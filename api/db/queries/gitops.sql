-- name: CreateGitOps :one
INSERT INTO gitops (org_id, name, repo_url, branch, path, apply_mode)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, org_id, name, repo_url, branch, path, target_org_id, apply_mode, last_sha, last_status, drift_added, drift_changed, drift_removed, last_sync, created_at;

-- name: GetGitOps :one
SELECT id, org_id, name, repo_url, branch, path, target_org_id, apply_mode, last_sha, last_status, drift_added, drift_changed, drift_removed, last_sync, created_at
FROM gitops
WHERE id = $1;

-- name: ListGitOpsByOrg :many
SELECT id, org_id, name, repo_url, branch, path, target_org_id, apply_mode, last_sha, last_status, drift_added, drift_changed, drift_removed, last_sync, created_at
FROM gitops
WHERE org_id = $1
ORDER BY name;

-- name: UpdateGitOpsSync :exec
UPDATE gitops
SET last_sha = $2,
    last_status = $3,
    drift_added = $4,
    drift_changed = $5,
    drift_removed = $6,
    last_sync = now()
WHERE id = $1;

-- name: DeleteGitOps :exec
DELETE FROM gitops
WHERE id = $1 AND org_id = $2;
