CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    global_role   TEXT NOT NULL DEFAULT '' CHECK (global_role IN ('', 'admin')),
    totp_secret   BYTEA,
    totp_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE orgs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL UNIQUE,
    avatar        TEXT,
    color         TEXT,
    description   TEXT NOT NULL DEFAULT '',
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orgs_owner ON orgs(owner_user_id);

CREATE TABLE members (
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'developer', 'member', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX idx_members_user ON members(user_id);

CREATE TABLE api_keys (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    key_hash     TEXT NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_org ON api_keys(org_id);

CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL DEFAULT '',
    details       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_org ON audit_logs(org_id, created_at DESC);
CREATE TABLE projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    color       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_projects_org ON projects(org_id, deleted_at);

CREATE TABLE environments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    color       TEXT NOT NULL DEFAULT '',
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, slug)
);

CREATE INDEX idx_environments_project ON environments(project_id);

CREATE TABLE apps (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id  UUID REFERENCES environments(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    source_type     TEXT NOT NULL DEFAULT 'image' CHECK (source_type IN ('image', 'git', 'upload')),
    image           TEXT NOT NULL DEFAULT '',
    git_url         TEXT NOT NULL DEFAULT '',
    git_branch      TEXT NOT NULL DEFAULT 'main',
    dockerfile      TEXT NOT NULL DEFAULT 'Dockerfile',
    port            INTEGER NOT NULL DEFAULT 80 CHECK (port BETWEEN 1 AND 65535),
    cpus            TEXT NOT NULL DEFAULT '',
    mem_mb          INTEGER NOT NULL DEFAULT 0 CHECK (mem_mb >= 0),
    hc_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    hc_path         TEXT NOT NULL DEFAULT '/',
    hc_interval_ms  INTEGER NOT NULL DEFAULT 5000,
    hc_timeout_ms   INTEGER NOT NULL DEFAULT 2000,
    hc_retries      INTEGER NOT NULL DEFAULT 3,
    webhook_secret  TEXT NOT NULL DEFAULT '',
    build_type      TEXT NOT NULL DEFAULT 'dockerfile',
    preview_domain  TEXT NOT NULL DEFAULT '',
    image_retention INTEGER NOT NULL DEFAULT 0 CHECK (image_retention >= 0),
    storage_mb      INTEGER NOT NULL DEFAULT 0 CHECK (storage_mb >= 0),
    install_command TEXT NOT NULL DEFAULT '',
    build_command   TEXT NOT NULL DEFAULT '',
    start_command   TEXT NOT NULL DEFAULT '',
    root_folder     TEXT NOT NULL DEFAULT '',
    dist_folder     TEXT NOT NULL DEFAULT '',
    watch_paths     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_apps_org ON apps(org_id);
CREATE INDEX idx_apps_project ON apps(project_id);
CREATE UNIQUE INDEX idx_apps_name_org ON apps(org_id, lower(name));

CREATE TABLE app_env (
    app_id  UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name    TEXT NOT NULL,
    value   TEXT NOT NULL,
    secret  BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (app_id, name)
);

CREATE TABLE env_variables (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id UUID NOT NULL REFERENCES environments(id) ON DELETE CASCADE,
    key            TEXT NOT NULL,
    value          TEXT NOT NULL,
    is_secret      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, environment_id, key)
);

CREATE INDEX idx_env_variables_project ON env_variables(project_id, environment_id);
CREATE TABLE deployments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id       UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    number       INTEGER NOT NULL,
    status       TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'building', 'starting', 'health_checking', 'ready', 'failed', 'rolled_back', 'cancelled')),
    trigger      TEXT NOT NULL DEFAULT '',
    triggered_by TEXT NOT NULL DEFAULT '',
    commit_sha   TEXT NOT NULL DEFAULT '',
    image_ref    TEXT NOT NULL DEFAULT '',
    container_id TEXT NOT NULL DEFAULT '',
    server_id    TEXT NOT NULL DEFAULT '',
    error        TEXT NOT NULL DEFAULT '',
    env_snapshot JSONB NOT NULL DEFAULT '{}',
    compose_yaml TEXT NOT NULL DEFAULT '',
    deploy_spec  JSONB,
    compose_hash TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at   TIMESTAMPTZ,
    finished_at  TIMESTAMPTZ,
    UNIQUE (app_id, number)
);

CREATE INDEX idx_deployments_app ON deployments(app_id, number DESC);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE TABLE domains (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id      UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    host        TEXT NOT NULL,
    https       BOOLEAN NOT NULL DEFAULT FALSE,
    cert_status TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (app_id, host)
);

CREATE INDEX idx_domains_app ON domains(app_id);
CREATE INDEX idx_domains_host ON domains(host);

CREATE TABLE previews (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id        UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    branch        TEXT NOT NULL,
    deployment_id UUID REFERENCES deployments(id) ON DELETE SET NULL,
    container_id  TEXT NOT NULL DEFAULT '',
    domain        TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'failed', 'removed')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (app_id, branch)
);

CREATE INDEX idx_previews_app ON previews(app_id);
CREATE TABLE cron_jobs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    schedule   TEXT NOT NULL,
    command    TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    last_run   TIMESTAMPTZ,
    next_run   TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cron_app ON cron_jobs(app_id);

CREATE TABLE workers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id       UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    command      TEXT NOT NULL,
    replicas     INTEGER NOT NULL DEFAULT 1 CHECK (replicas BETWEEN 1 AND 20),
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    status       TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped', 'running')),
    container_id TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workers_app ON workers(app_id);

CREATE TABLE app_policies (
    app_id        UUID PRIMARY KEY REFERENCES apps(id) ON DELETE CASCADE,
    enabled       BOOLEAN NOT NULL DEFAULT FALSE,
    cpu_min       REAL NOT NULL DEFAULT 0.25 CHECK (cpu_min > 0),
    cpu_max       REAL NOT NULL DEFAULT 4 CHECK (cpu_max >= cpu_min),
    mem_min_mb    INTEGER NOT NULL DEFAULT 128 CHECK (mem_min_mb >= 0),
    mem_max_mb    INTEGER NOT NULL DEFAULT 2048 CHECK (mem_max_mb >= mem_min_mb),
    scale_up_pct  INTEGER NOT NULL DEFAULT 80 CHECK (scale_up_pct BETWEEN 1 AND 100),
    scale_down_pct INTEGER NOT NULL DEFAULT 15 CHECK (scale_down_pct BETWEEN 1 AND 100),
    cooldown_min  INTEGER NOT NULL DEFAULT 15 CHECK (cooldown_min >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE autopilot_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    action     TEXT NOT NULL,
    detail     TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_autopilot_app ON autopilot_events(app_id, created_at DESC);

CREATE TABLE databases (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id       UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    engine       TEXT NOT NULL CHECK (engine IN ('postgres', 'mysql', 'mariadb', 'redis', 'mongodb', 'mssql', 'oracle')),
    version      TEXT NOT NULL DEFAULT '',
    port         INTEGER NOT NULL DEFAULT 0,
    db_name      TEXT NOT NULL,
    db_user      TEXT NOT NULL,
    pass_enc     TEXT NOT NULL,
    mem_mb       INTEGER NOT NULL DEFAULT 0 CHECK (mem_mb >= 0),
    storage_mb   INTEGER NOT NULL DEFAULT 0 CHECK (storage_mb >= 0),
    status       TEXT NOT NULL DEFAULT 'creating' CHECK (status IN ('creating', 'running', 'stopped', 'failed', 'deleting')),
    container_id TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, name)
);

CREATE INDEX idx_databases_org ON databases(org_id);
CREATE INDEX idx_databases_project ON databases(project_id);

CREATE TABLE backups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    database_id UUID REFERENCES databases(id) ON DELETE CASCADE,
    app_id      UUID REFERENCES apps(id) ON DELETE SET NULL,
    path        TEXT NOT NULL,
    size        BIGINT NOT NULL DEFAULT 0 CHECK (size >= 0),
    kind        TEXT NOT NULL DEFAULT 'state' CHECK (kind IN ('state', 'db', 'volume')),
    dest        TEXT NOT NULL DEFAULT 'local',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_backups_org ON backups(org_id, created_at DESC);
CREATE INDEX idx_backups_database ON backups(database_id);

CREATE TABLE app_volumes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id     UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    mount_path TEXT NOT NULL,
    UNIQUE (app_id, name)
);

CREATE INDEX idx_volumes_app ON app_volumes(app_id);

CREATE TABLE templates (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',
    category       TEXT NOT NULL DEFAULT '',
    icon           TEXT NOT NULL DEFAULT '',
    version        TEXT NOT NULL DEFAULT '1',
    definition     TEXT NOT NULL,
    readme         TEXT NOT NULL DEFAULT '',
    homepage       TEXT NOT NULL DEFAULT '',
    github         TEXT NOT NULL DEFAULT '',
    license        TEXT NOT NULL DEFAULT 'MIT',
    installs       INTEGER NOT NULL DEFAULT 0 CHECK (installs >= 0),
    featured       BOOLEAN NOT NULL DEFAULT FALSE,
    verified       BOOLEAN NOT NULL DEFAULT TRUE,
    editors_choice BOOLEAN NOT NULL DEFAULT FALSE,
    tags           TEXT[] NOT NULL DEFAULT '{}',
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE compose_apps (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    compose    TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'stopped' CHECK (status IN ('stopped', 'running', 'error')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_compose_org ON compose_apps(org_id);

CREATE TABLE gitops (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    repo_url      TEXT NOT NULL,
    branch        TEXT NOT NULL DEFAULT 'main',
    path          TEXT NOT NULL DEFAULT 'aether.yml',
    target_org_id UUID REFERENCES orgs(id) ON DELETE SET NULL,
    apply_mode    TEXT NOT NULL DEFAULT 'manual' CHECK (apply_mode IN ('manual', 'auto')),
    last_sha      TEXT NOT NULL DEFAULT '',
    last_status   TEXT NOT NULL DEFAULT 'pending' CHECK (last_status IN ('pending', 'synced', 'error')),
    drift_added   INTEGER NOT NULL DEFAULT 0 CHECK (drift_added >= 0),
    drift_changed INTEGER NOT NULL DEFAULT 0 CHECK (drift_changed >= 0),
    drift_removed INTEGER NOT NULL DEFAULT 0 CHECK (drift_removed >= 0),
    last_sync     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_gitops_org ON gitops(org_id);

CREATE TABLE alert_rules (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    metric     TEXT NOT NULL,
    threshold  REAL NOT NULL,
    window_s   INTEGER NOT NULL DEFAULT 30 CHECK (window_s > 0),
    severity   TEXT NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'critical')),
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    target_app UUID REFERENCES apps(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_alert_rules_org ON alert_rules(org_id);

CREATE TABLE alert_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    rule_id     UUID REFERENCES alert_rules(id) ON DELETE CASCADE,
    app_id      UUID REFERENCES apps(id) ON DELETE SET NULL,
    app_name    TEXT NOT NULL DEFAULT '',
    severity    TEXT NOT NULL,
    message     TEXT NOT NULL,
    value       REAL NOT NULL DEFAULT 0,
    threshold   REAL NOT NULL DEFAULT 0,
    metric      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);

CREATE INDEX idx_alert_events_org ON alert_events(org_id, created_at DESC);

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    type       TEXT NOT NULL,
    message    TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '{}',
    read       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_org ON notifications(org_id, created_at DESC);

CREATE TABLE notification_channels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('email', 'webhook', 'slack', 'telegram')),
    config_enc TEXT NOT NULL,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_channels_org ON notification_channels(org_id);

CREATE TABLE snapshots (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    app_id      UUID REFERENCES apps(id) ON DELETE SET NULL,
    volume      TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL DEFAULT '',
    size        BIGINT NOT NULL DEFAULT 0 CHECK (size >= 0),
    chunks      INTEGER NOT NULL DEFAULT 0 CHECK (chunks >= 0),
    dedup_saved BIGINT NOT NULL DEFAULT 0 CHECK (dedup_saved >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_snapshots_org ON snapshots(org_id, created_at DESC);

CREATE TABLE snapshot_schedules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    app_id      UUID REFERENCES apps(id) ON DELETE SET NULL,
    volume      TEXT NOT NULL,
    name_prefix TEXT NOT NULL DEFAULT '',
    cron        TEXT NOT NULL,
    retention   INTEGER NOT NULL DEFAULT 7 CHECK (retention >= 0),
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    last_run    TIMESTAMPTZ,
    next_run    TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_schedules_org ON snapshot_schedules(org_id);

CREATE TABLE clusters (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    labels     TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_clusters_org ON clusters(org_id);

CREATE TABLE servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    host            TEXT NOT NULL DEFAULT '',
    role            TEXT NOT NULL DEFAULT 'agent',
    status          TEXT NOT NULL DEFAULT 'registered' CHECK (status IN ('registered', 'online', 'offline')),
    version         TEXT NOT NULL DEFAULT '',
    labels          TEXT[] NOT NULL DEFAULT '{}',
    cpu_cores       INTEGER NOT NULL DEFAULT 0 CHECK (cpu_cores >= 0),
    mem_total_bytes BIGINT NOT NULL DEFAULT 0 CHECK (mem_total_bytes >= 0),
    load            REAL NOT NULL DEFAULT 0 CHECK (load >= 0),
    last_heartbeat  TIMESTAMPTZ,
    cluster_id      UUID REFERENCES clusters(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_servers_cluster ON servers(cluster_id);

CREATE TABLE registry_settings (
    id           INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    host         TEXT NOT NULL DEFAULT '127.0.0.1',
    port         INTEGER NOT NULL DEFAULT 5000,
    container_id TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'stopped'
);

CREATE TABLE pipelines (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    app_id     UUID REFERENCES apps(id) ON DELETE SET NULL,
    name       TEXT NOT NULL,
    trigger    TEXT NOT NULL DEFAULT 'manual' CHECK (trigger IN ('manual', 'auto', 'webhook')),
    stages     JSONB NOT NULL DEFAULT '[]',
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pipelines_org ON pipelines(org_id);

CREATE TABLE pipeline_runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pipeline_id UUID NOT NULL REFERENCES pipelines(id) ON DELETE CASCADE,
    status      TEXT NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'success', 'failed')),
    trigger     TEXT NOT NULL DEFAULT '',
    log         TEXT NOT NULL DEFAULT '',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_pipeline_runs ON pipeline_runs(pipeline_id, started_at DESC);

CREATE TABLE branding (
    org_id        UUID PRIMARY KEY REFERENCES orgs(id) ON DELETE CASCADE,
    name          TEXT NOT NULL DEFAULT '',
    logo_url      TEXT NOT NULL DEFAULT '',
    primary_color TEXT NOT NULL DEFAULT '',
    accent_color  TEXT NOT NULL DEFAULT '',
    dark_mode     BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE s3_destinations (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    endpoint       TEXT NOT NULL,
    bucket         TEXT NOT NULL,
    region         TEXT NOT NULL DEFAULT 'us-east-1',
    access_key_enc TEXT NOT NULL,
    secret_key_enc TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_s3_org ON s3_destinations(org_id);

CREATE TABLE oidc_providers (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    issuer           TEXT NOT NULL,
    client_id        TEXT NOT NULL,
    client_secret_enc TEXT NOT NULL DEFAULT '',
    scopes           TEXT NOT NULL DEFAULT 'openid email profile',
    enabled          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oidc_org ON oidc_providers(org_id);

CREATE TABLE out_webhooks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL,
    secret_enc TEXT NOT NULL DEFAULT '',
    events     TEXT[] NOT NULL DEFAULT '{}',
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_webhooks_org ON out_webhooks(org_id);

CREATE TABLE registry_mirrors (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    source          TEXT NOT NULL,
    dest            TEXT NOT NULL,
    dest_tls_verify BOOLEAN NOT NULL DEFAULT TRUE,
    tags_filter     TEXT NOT NULL DEFAULT '',
    schedule        TEXT NOT NULL DEFAULT '',
    last_run        TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'idle' CHECK (status IN ('idle', 'syncing', 'synced', 'error')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE project_assignments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (org_id, user_id, project_id)
);

CREATE INDEX idx_assignments_user ON project_assignments(user_id);

CREATE TABLE server_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash TEXT NOT NULL UNIQUE,
    server_id  UUID REFERENCES servers(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ
);

CREATE TABLE variable_audit (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id UUID REFERENCES environments(id) ON DELETE SET NULL,
    user_id        UUID REFERENCES users(id) ON DELETE SET NULL,
    action         TEXT NOT NULL,
    key            TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_var_audit_project ON variable_audit(project_id, created_at DESC);
