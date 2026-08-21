CREATE TABLE backup_configurations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id      UUID NOT NULL UNIQUE REFERENCES databases(id) ON DELETE CASCADE,
    enabled          BOOLEAN NOT NULL DEFAULT false,
    destination_id   UUID NOT NULL REFERENCES s3_destinations(id) ON DELETE CASCADE,
    path_prefix      TEXT NOT NULL DEFAULT 'databases',
    schedule_type    TEXT NOT NULL DEFAULT 'daily',
    schedule_minute  INT NOT NULL DEFAULT 0,
    schedule_at      TEXT NOT NULL DEFAULT '03:00',
    schedule_day     TEXT NOT NULL DEFAULT 'sunday',
    schedule_start   TEXT NOT NULL DEFAULT '',
    schedule_cron    TEXT NOT NULL DEFAULT '',
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    retention_type   TEXT NOT NULL DEFAULT 'all',
    next_run_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE backup_jobs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_id      UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    configuration_id UUID REFERENCES backup_configurations(id) ON DELETE SET NULL,
    trigger_type     TEXT NOT NULL DEFAULT 'manual',
    status           TEXT NOT NULL DEFAULT 'queued',
    engine           TEXT NOT NULL DEFAULT '',
    engine_version   TEXT NOT NULL DEFAULT '',
    format           TEXT NOT NULL DEFAULT '',
    destination_id   UUID REFERENCES s3_destinations(id) ON DELETE SET NULL,
    storage_key      TEXT NOT NULL DEFAULT '',
    size_bytes       BIGINT NOT NULL DEFAULT 0,
    checksum         TEXT NOT NULL DEFAULT '',
    error_code       TEXT NOT NULL DEFAULT '',
    error_message    TEXT NOT NULL DEFAULT '',
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_backup_jobs_database ON backup_jobs(database_id, created_at DESC);
CREATE INDEX idx_backup_jobs_status ON backup_jobs(status);

CREATE TABLE restore_jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_id           UUID NOT NULL REFERENCES backup_jobs(id) ON DELETE CASCADE,
    target_database_id  UUID NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'queued',
    error_code          TEXT NOT NULL DEFAULT '',
    error_message       TEXT NOT NULL DEFAULT '',
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_restore_jobs_target ON restore_jobs(target_database_id, created_at DESC);
