-- name: CreateApp :one
INSERT INTO apps (
    org_id, project_id, environment_id, name, source_type, image, git_url, git_branch, upload_id,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, build_type, preview_domain, image_retention, storage_mb, install_command, build_command,
    start_command, root_folder, dist_folder, watch_paths
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15, $16, $17,
    $18, $19, $20, $21, $22, $23,
    $24, $25, $26, $27, $28
)
RETURNING id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id;

-- name: GetApp :one
SELECT id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id
FROM apps
WHERE id = $1 AND org_id = $2;

-- name: GetAppByName :one
SELECT id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id
FROM apps
WHERE org_id = $1 AND lower(name) = lower($2);

-- name: GetAppByID :one
SELECT id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id
FROM apps
WHERE id = $1;

-- name: ListAppsByProject :many
SELECT id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id
FROM apps
WHERE org_id = $1 AND project_id = $2
ORDER BY name;

-- name: ListAppsByOrg :many
SELECT id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id
FROM apps
WHERE org_id = $1
ORDER BY name;

-- name: UpdateApp :one
UPDATE apps
SET name = $3, image = $4, git_url = $5, git_branch = $6, upload_id = $7, dockerfile = $8, port = $9,
    cpus = $10, mem_mb = $11, hc_enabled = $12, hc_path = $13, hc_interval_ms = $14,
    hc_timeout_ms = $15, hc_retries = $16, build_type = $17, environment_id = $18,
    preview_domain = $19, image_retention = $20, storage_mb = $21, install_command = $22,
    build_command = $23, start_command = $24, root_folder = $25, dist_folder = $26,
    watch_paths = $27,
    updated_at = now()
WHERE id = $1 AND org_id = $2
RETURNING id, org_id, project_id, environment_id, name, source_type, image, git_url, git_branch,
    dockerfile, port, cpus, mem_mb, hc_enabled, hc_path, hc_interval_ms, hc_timeout_ms,
    hc_retries, webhook_secret, build_type, preview_domain, image_retention, storage_mb,
    install_command, build_command, start_command, root_folder, dist_folder, watch_paths,
    created_at, updated_at, upload_id;

-- name: DeleteApp :exec
DELETE FROM apps
WHERE id = $1 AND org_id = $2;

-- name: SetWebhookSecret :exec
UPDATE apps
SET webhook_secret = $3, updated_at = now()
WHERE id = $1 AND org_id = $2;

