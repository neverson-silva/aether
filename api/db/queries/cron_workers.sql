-- name: CreateCronJob :one
INSERT INTO cron_jobs (app_id, name, schedule, command, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, app_id, name, schedule, command, enabled, last_run, next_run, created_at;

-- name: GetCronJob :one
SELECT id, app_id, name, schedule, command, enabled, last_run, next_run, created_at
FROM cron_jobs
WHERE id = $1;

-- name: ListCronJobsByApp :many
SELECT id, app_id, name, schedule, command, enabled, last_run, next_run, created_at
FROM cron_jobs
WHERE app_id = $1
ORDER BY name;

-- name: ListCronJobsByOrg :many
SELECT c.id, c.app_id, c.name, c.schedule, c.command, c.enabled, c.last_run, c.next_run, c.created_at
FROM cron_jobs c
JOIN apps a ON a.id = c.app_id
WHERE a.org_id = $1
ORDER BY c.name;

-- name: UpdateCronJob :one
UPDATE cron_jobs
SET schedule = $3, command = $4, enabled = $5
WHERE id = $1 AND app_id = $2
RETURNING id, app_id, name, schedule, command, enabled, last_run, next_run, created_at;

-- name: DeleteCronJob :exec
DELETE FROM cron_jobs
WHERE id = $1;

-- name: CreateWorker :one
INSERT INTO workers (app_id, name, command, replicas, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, app_id, name, command, replicas, enabled, status, container_id, created_at;

-- name: GetWorker :one
SELECT id, app_id, name, command, replicas, enabled, status, container_id, created_at
FROM workers
WHERE id = $1;

-- name: ListWorkersByApp :many
SELECT id, app_id, name, command, replicas, enabled, status, container_id, created_at
FROM workers
WHERE app_id = $1
ORDER BY name;

-- name: SetWorkerState :exec
UPDATE workers
SET status = $3, container_id = $4
WHERE id = $1 AND app_id = $2;

-- name: UpdateWorker :one
UPDATE workers
SET name = $3, command = $4, replicas = $5, enabled = $6
WHERE id = $1 AND app_id = $2
RETURNING id, app_id, name, command, replicas, enabled, status, container_id, created_at;

-- name: DeleteWorker :exec
DELETE FROM workers
WHERE id = $1;

-- name: GetAppPolicy :one
SELECT app_id, enabled, cpu_min, cpu_max, mem_min_mb, mem_max_mb, scale_up_pct, scale_down_pct, cooldown_min, updated_at
FROM app_policies
WHERE app_id = $1;

-- name: UpsertAppPolicy :one
INSERT INTO app_policies (app_id, enabled, cpu_min, cpu_max, mem_min_mb, mem_max_mb, scale_up_pct, scale_down_pct, cooldown_min)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (app_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    cpu_min = EXCLUDED.cpu_min,
    cpu_max = EXCLUDED.cpu_max,
    mem_min_mb = EXCLUDED.mem_min_mb,
    mem_max_mb = EXCLUDED.mem_max_mb,
    scale_up_pct = EXCLUDED.scale_up_pct,
    scale_down_pct = EXCLUDED.scale_down_pct,
    cooldown_min = EXCLUDED.cooldown_min,
    updated_at = now()
RETURNING app_id, enabled, cpu_min, cpu_max, mem_min_mb, mem_max_mb, scale_up_pct, scale_down_pct, cooldown_min, updated_at;

-- name: CreateAutopilotEvent :exec
INSERT INTO autopilot_events (app_id, action, detail)
VALUES ($1, $2, $3);

-- name: ListAutopilotEvents :many
SELECT id, app_id, action, detail, created_at
FROM autopilot_events
WHERE app_id = $1
ORDER BY created_at DESC
LIMIT $2;
