CREATE TABLE IF NOT EXISTS oidc_auth_states (
    state_key TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    nonce TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_oidc_auth_states_expiry ON oidc_auth_states(expires_at);
