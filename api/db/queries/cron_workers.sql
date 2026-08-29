-- name: CreateCronJob :one
INSERT INTO cron_jobs (app_id, service_id, name, schedule, command, enabled)
SELECT $1, apps.service_id, $2, $3, $4, $5 FROM apps WHERE apps.id = $1
RETURNING id, app_id, name, schedule, command, enabled, last_run, next_run, created_at, service_id;

-- name: GetCronJob :one
SELECT id, app_id, name, schedule, command, enabled, last_run, next_run, created_at, service_id
FROM cron_jobs
WHERE id = $1;

-- name: ListCronJobsByApp :many
SELECT id, app_id, name, schedule, command, enabled, last_run, next_run, created_at, service_id
FROM cron_jobs
WHERE service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $1)
ORDER BY name;

-- name: ListCronJobsByOrg :many
SELECT c.id, c.app_id, c.name, c.schedule, c.command, c.enabled, c.last_run, c.next_run, c.created_at, c.service_id
FROM cron_jobs c
JOIN services s ON s.id = c.service_id
WHERE s.org_id = $1
ORDER BY c.name;

-- name: UpdateCronJob :one
UPDATE cron_jobs
SET schedule = $3, command = $4, enabled = $5
WHERE cron_jobs.id = $1 AND cron_jobs.service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $2)
RETURNING id, app_id, name, schedule, command, enabled, last_run, next_run, created_at, service_id;

-- name: DeleteCronJob :exec
DELETE FROM cron_jobs
WHERE id = $1;

-- name: CreateWorker :one
INSERT INTO workers (app_id, service_id, name, command, replicas, enabled)
SELECT $1, apps.service_id, $2, $3, $4, $5 FROM apps WHERE apps.id = $1
RETURNING id, app_id, name, command, replicas, enabled, status, container_id, created_at, service_id;

-- name: GetWorker :one
SELECT id, app_id, name, command, replicas, enabled, status, container_id, created_at, service_id
FROM workers
WHERE id = $1;

-- name: ListWorkersByApp :many
SELECT id, app_id, name, command, replicas, enabled, status, container_id, created_at, service_id
FROM workers
WHERE service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $1)
ORDER BY name;

-- name: SetWorkerState :exec
UPDATE workers
SET status = $3, container_id = $4
WHERE workers.id = $1 AND workers.service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $2);

-- name: UpdateWorker :one
UPDATE workers
SET name = $3, command = $4, replicas = $5, enabled = $6
WHERE workers.id = $1 AND workers.service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $2)
RETURNING id, app_id, name, command, replicas, enabled, status, container_id, created_at, service_id;

-- name: DeleteWorker :exec
DELETE FROM workers
WHERE id = $1;

-- name: GetAppPolicy :one
SELECT app_id, enabled, cpu_min, cpu_max, mem_min_mb, mem_max_mb, scale_up_pct, scale_down_pct, cooldown_min, updated_at, service_id
FROM app_policies
WHERE service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $1);

-- name: UpsertAppPolicy :one
INSERT INTO app_policies (app_id, service_id, enabled, cpu_min, cpu_max, mem_min_mb, mem_max_mb, scale_up_pct, scale_down_pct, cooldown_min)
SELECT $1, apps.service_id, $2, $3, $4, $5, $6, $7, $8, $9 FROM apps WHERE apps.id = $1
ON CONFLICT (app_id) DO UPDATE SET
    service_id = EXCLUDED.service_id,
    enabled = EXCLUDED.enabled,
    cpu_min = EXCLUDED.cpu_min,
    cpu_max = EXCLUDED.cpu_max,
    mem_min_mb = EXCLUDED.mem_min_mb,
    mem_max_mb = EXCLUDED.mem_max_mb,
    scale_up_pct = EXCLUDED.scale_up_pct,
    scale_down_pct = EXCLUDED.scale_down_pct,
    cooldown_min = EXCLUDED.cooldown_min,
    updated_at = now()
RETURNING app_id, enabled, cpu_min, cpu_max, mem_min_mb, mem_max_mb, scale_up_pct, scale_down_pct, cooldown_min, updated_at, service_id;

-- name: CreateAutopilotEvent :exec
INSERT INTO autopilot_events (app_id, service_id, action, detail)
SELECT $1, apps.service_id, $2, $3 FROM apps WHERE apps.id = $1;

-- name: ListAutopilotEvents :many
SELECT id, app_id, action, detail, created_at, service_id
FROM autopilot_events
WHERE service_id = (SELECT apps.service_id FROM apps WHERE apps.id = $1)
ORDER BY created_at DESC
LIMIT $2;
