-- name: CreateSnapshot :one
INSERT INTO snapshots (org_id, app_id, service_id, volume, name, size, chunks, dedup_saved)
VALUES ($1, $2, (SELECT apps.service_id FROM apps WHERE apps.id = $2), $3, $4, $5, $6, $7)
RETURNING id, org_id, app_id, volume, name, size, chunks, dedup_saved, created_at, service_id;

-- name: GetSnapshot :one
SELECT id, org_id, app_id, volume, name, size, chunks, dedup_saved, created_at, service_id
FROM snapshots
WHERE id = $1;

-- name: ListSnapshotsByOrg :many
SELECT id, org_id, app_id, volume, name, size, chunks, dedup_saved, created_at, service_id
FROM snapshots
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: DeleteSnapshot :exec
DELETE FROM snapshots
WHERE id = $1 AND org_id = $2;

-- name: CreateSnapshotSchedule :one
INSERT INTO snapshot_schedules (org_id, app_id, service_id, volume, name_prefix, cron, retention, enabled)
VALUES ($1, $2, (SELECT apps.service_id FROM apps WHERE apps.id = $2), $3, $4, $5, $6, $7)
RETURNING id, org_id, app_id, volume, name_prefix, cron, retention, enabled, last_run, next_run, created_at, service_id;

-- name: GetSnapshotSchedule :one
SELECT id, org_id, app_id, volume, name_prefix, cron, retention, enabled, last_run, next_run, created_at, service_id
FROM snapshot_schedules
WHERE id = $1;

-- name: ListSchedulesByOrg :many
SELECT id, org_id, app_id, volume, name_prefix, cron, retention, enabled, last_run, next_run, created_at, service_id
FROM snapshot_schedules
WHERE org_id = $1
ORDER BY name_prefix;

-- name: DeleteSnapshotSchedule :exec
DELETE FROM snapshot_schedules
WHERE id = $1 AND org_id = $2;

-- name: CreateSnapshotForService :one
INSERT INTO snapshots (org_id, service_id, volume, name, size, chunks, dedup_saved)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, app_id, volume, name, size, chunks, dedup_saved, created_at, service_id;

-- name: ListSnapshotsByService :many
SELECT id, org_id, app_id, volume, name, size, chunks, dedup_saved, created_at, service_id
FROM snapshots
WHERE org_id = $1 AND service_id = $2
ORDER BY created_at DESC
LIMIT $3;

-- name: CreateSnapshotScheduleForService :one
INSERT INTO snapshot_schedules (org_id, service_id, volume, name_prefix, cron, retention, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, app_id, volume, name_prefix, cron, retention, enabled, last_run, next_run, created_at, service_id;

-- name: ListSchedulesByService :many
SELECT id, org_id, app_id, volume, name_prefix, cron, retention, enabled, last_run, next_run, created_at, service_id
FROM snapshot_schedules
WHERE org_id = $1 AND service_id = $2
ORDER BY name_prefix;
