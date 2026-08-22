ALTER TABLE backup_configurations
    DROP CONSTRAINT IF EXISTS backup_configurations_database_id_key;

CREATE INDEX IF NOT EXISTS idx_backup_configurations_database
    ON backup_configurations(database_id, created_at DESC);
