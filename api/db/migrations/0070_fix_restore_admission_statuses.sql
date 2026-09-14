DROP INDEX IF EXISTS idx_restore_jobs_org_active;

CREATE INDEX IF NOT EXISTS idx_restore_jobs_org_active
    ON restore_jobs(target_database_id, status) WHERE status IN ('queued', 'uploading', 'validating', 'ready', 'preparing', 'downloading', 'restoring', 'verifying');
