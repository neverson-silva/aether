-- name: GetBackupConfigurationByDatabase :one
SELECT id, database_id, enabled, destination_id, path_prefix, schedule_type,
       schedule_minute, schedule_at, schedule_day, schedule_start, schedule_cron,
       timezone, retention_type, next_run_at, created_at, updated_at
FROM backup_configurations
WHERE database_id = $1;

-- name: UpsertBackupConfiguration :one
INSERT INTO backup_configurations (
    database_id, enabled, destination_id, path_prefix, schedule_type,
    schedule_minute, schedule_at, schedule_day, schedule_start, schedule_cron,
    timezone, retention_type, next_run_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now())
ON CONFLICT (database_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    destination_id = EXCLUDED.destination_id,
    path_prefix = EXCLUDED.path_prefix,
    schedule_type = EXCLUDED.schedule_type,
    schedule_minute = EXCLUDED.schedule_minute,
    schedule_at = EXCLUDED.schedule_at,
    schedule_day = EXCLUDED.schedule_day,
    schedule_start = EXCLUDED.schedule_start,
    schedule_cron = EXCLUDED.schedule_cron,
    timezone = EXCLUDED.timezone,
    retention_type = EXCLUDED.retention_type,
    next_run_at = EXCLUDED.next_run_at,
    updated_at = now()
RETURNING id, database_id, enabled, destination_id, path_prefix, schedule_type,
          schedule_minute, schedule_at, schedule_day, schedule_start, schedule_cron,
          timezone, retention_type, next_run_at, created_at, updated_at;

-- name: DeleteBackupConfiguration :exec
DELETE FROM backup_configurations WHERE database_id = $1;

-- name: ListEnabledBackupConfigurations :many
SELECT id, database_id, enabled, destination_id, path_prefix, schedule_type,
       schedule_minute, schedule_at, schedule_day, schedule_start, schedule_cron,
       timezone, retention_type, next_run_at, created_at, updated_at
FROM backup_configurations
WHERE enabled = true
ORDER BY created_at;

-- name: SetBackupConfigurationNextRun :exec
UPDATE backup_configurations
SET next_run_at = $2, updated_at = now()
WHERE id = $1;

-- name: CreateBackupJob :one
INSERT INTO backup_jobs (
    database_id, configuration_id, trigger_type, status, engine, engine_version,
    format, destination_id, storage_key, size_bytes, checksum, error_code,
    error_message, started_at, completed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
RETURNING id, database_id, configuration_id, trigger_type, status, engine,
          engine_version, format, destination_id, storage_key, size_bytes,
          checksum, error_code, error_message, started_at, completed_at, created_at;

-- name: GetBackupJob :one
SELECT id, database_id, configuration_id, trigger_type, status, engine,
       engine_version, format, destination_id, storage_key, size_bytes,
       checksum, error_code, error_message, started_at, completed_at, created_at
FROM backup_jobs
WHERE id = $1;

-- name: UpdateBackupJob :one
UPDATE backup_jobs
SET status = $2,
    engine = $3,
    engine_version = $4,
    format = $5,
    storage_key = $6,
    size_bytes = $7,
    checksum = $8,
    error_code = $9,
    error_message = $10,
    started_at = $11,
    completed_at = $12
WHERE id = $1
RETURNING id, database_id, configuration_id, trigger_type, status, engine,
          engine_version, format, destination_id, storage_key, size_bytes,
          checksum, error_code, error_message, started_at, completed_at, created_at;

-- name: ListBackupJobsByDatabase :many
SELECT id, database_id, configuration_id, trigger_type, status, engine,
       engine_version, format, destination_id, storage_key, size_bytes,
       checksum, error_code, error_message, started_at, completed_at, created_at
FROM backup_jobs
WHERE database_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListActiveBackupJobsByDatabase :many
SELECT id, database_id, configuration_id, trigger_type, status, engine,
       engine_version, format, destination_id, storage_key, size_bytes,
       checksum, error_code, error_message, started_at, completed_at, created_at
FROM backup_jobs
WHERE database_id = $1 AND status IN ('queued', 'preparing', 'running', 'uploading', 'verifying', 'cancelling')
ORDER BY created_at DESC;

-- name: ListBackupJobsDue :many
SELECT id, database_id, configuration_id, trigger_type, status, engine,
       engine_version, format, destination_id, storage_key, size_bytes,
       checksum, error_code, error_message, started_at, completed_at, created_at
FROM backup_jobs
WHERE status = 'queued'
ORDER BY created_at
LIMIT $1;

-- name: CreateRestoreJob :one
INSERT INTO restore_jobs (
    backup_id, target_database_id, status, error_code, error_message, started_at, completed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, backup_id, target_database_id, status, error_code, error_message,
          started_at, completed_at, created_at;

-- name: GetRestoreJob :one
SELECT id, backup_id, target_database_id, status, error_code, error_message,
       started_at, completed_at, created_at
FROM restore_jobs
WHERE id = $1;

-- name: UpdateRestoreJob :one
UPDATE restore_jobs
SET status = $2, error_code = $3, error_message = $4, started_at = $5, completed_at = $6
WHERE id = $1
RETURNING id, backup_id, target_database_id, status, error_code, error_message,
          started_at, completed_at, created_at;

-- name: ListRestoreJobsByTarget :many
SELECT id, backup_id, target_database_id, status, error_code, error_message,
       started_at, completed_at, created_at
FROM restore_jobs
WHERE target_database_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListRestoreJobsDue :many
SELECT id, backup_id, target_database_id, status, error_code, error_message,
       started_at, completed_at, created_at
FROM restore_jobs
WHERE status = 'queued'
ORDER BY created_at
LIMIT $1;
