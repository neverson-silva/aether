-- Restore sources: S3-backed backups and manually uploaded files share the same
-- restore job pipeline.
ALTER TABLE restore_jobs
    ALTER COLUMN backup_id DROP NOT NULL,
    ADD COLUMN source_type TEXT NOT NULL DEFAULT 'backup',
    ADD COLUMN source_filename TEXT NOT NULL DEFAULT '',
    ADD COLUMN source_size BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN source_checksum TEXT NOT NULL DEFAULT '',
    ADD COLUMN source_format TEXT NOT NULL DEFAULT '',
    ADD COLUMN uploaded_bytes BIGINT NOT NULL DEFAULT 0;

ALTER TABLE restore_jobs
    ALTER COLUMN backup_id DROP NOT NULL;
