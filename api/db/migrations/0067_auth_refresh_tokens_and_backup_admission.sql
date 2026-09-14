CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_session ON auth_refresh_tokens(session_id, used_at, expires_at);

CREATE INDEX IF NOT EXISTS idx_backup_jobs_org_active
    ON backup_jobs(database_id, status) WHERE status IN ('queued', 'preparing', 'running', 'uploading', 'verifying');

CREATE INDEX IF NOT EXISTS idx_restore_jobs_org_active
    ON restore_jobs(target_database_id, status) WHERE status IN ('queued', 'uploading', 'validating', 'ready', 'preparing', 'downloading', 'restoring', 'verifying');
