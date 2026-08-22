ALTER TABLE scm_connections
    ADD COLUMN IF NOT EXISTS credentials_enc TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS scm_manifest_states (
    state TEXT PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    return_url TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE scm_manifest_states
    ADD COLUMN IF NOT EXISTS return_url TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_scm_manifest_states_expires_at
    ON scm_manifest_states (expires_at);
