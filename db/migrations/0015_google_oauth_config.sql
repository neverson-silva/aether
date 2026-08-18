-- Google Drive OAuth client configuration per org, stored encrypted. The
-- redirect URI is derived from AETHER_PUBLIC_URL and shared across orgs.

CREATE TABLE IF NOT EXISTS google_oauth_configs (
    org_id           UUID PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
    client_id        TEXT NOT NULL,
    client_secret_enc TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
