CREATE TABLE IF NOT EXISTS scm_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    external_account_id TEXT NOT NULL DEFAULT '',
    external_account_name TEXT NOT NULL DEFAULT '',
    installation_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, provider, installation_id)
);

CREATE TABLE IF NOT EXISTS service_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES scm_connections(id) ON DELETE CASCADE,
    repository_id TEXT NOT NULL,
    repository_owner TEXT NOT NULL DEFAULT '',
    repository_name TEXT NOT NULL DEFAULT '',
    repository_full_name TEXT NOT NULL DEFAULT '',
    default_branch TEXT NOT NULL DEFAULT 'main',
    branch TEXT NOT NULL DEFAULT 'main',
    auto_deploy BOOLEAN NOT NULL DEFAULT false,
    root_directory TEXT NOT NULL DEFAULT '',
    watch_paths TEXT[] NOT NULL DEFAULT '{}',
    ignore_paths TEXT[] NOT NULL DEFAULT '{}',
    watch_root_files BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (service_id),
    UNIQUE (connection_id, repository_id, service_id)
);

CREATE INDEX IF NOT EXISTS idx_service_sources_repository
    ON service_sources (connection_id, repository_id);

CREATE TABLE IF NOT EXISTS scm_webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    delivery_id TEXT NOT NULL,
    event_type TEXT NOT NULL DEFAULT '',
    installation_id TEXT NOT NULL DEFAULT '',
    repository_id TEXT NOT NULL DEFAULT '',
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'received',
    error TEXT NOT NULL DEFAULT '',
    UNIQUE (provider, delivery_id)
);
