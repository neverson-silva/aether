CREATE TABLE IF NOT EXISTS services (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    project_id      UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id  UUID REFERENCES environments(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL CHECK (kind IN ('app', 'compose', 'database')),
    status          TEXT NOT NULL DEFAULT 'unknown',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_services_org ON services(org_id);
CREATE INDEX IF NOT EXISTS idx_services_project ON services(project_id);
CREATE INDEX IF NOT EXISTS idx_services_environment ON services(environment_id);
ALTER TABLE apps ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE compose_apps ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE databases ADD COLUMN IF NOT EXISTS service_id UUID;

UPDATE apps SET service_id = gen_random_uuid() WHERE service_id IS NULL;

INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
SELECT service_id, org_id, project_id, environment_id, name, 'app', CASE LOWER(COALESCE(latest_deployment.status, ''))
    WHEN 'queued' THEN 'deploying' WHEN 'pending' THEN 'deploying' WHEN 'building' THEN 'deploying'
    WHEN 'deploying' THEN 'deploying' WHEN 'starting' THEN 'deploying' WHEN 'ready' THEN 'running'
    WHEN 'running' THEN 'running' WHEN 'healthy' THEN 'running' WHEN 'stopped' THEN 'stopped'
    WHEN 'exited' THEN 'stopped' WHEN 'failed' THEN 'failed' WHEN 'error' THEN 'failed'
    WHEN 'degraded' THEN 'degraded' ELSE 'unknown' END, created_at, updated_at
FROM apps
LEFT JOIN LATERAL (
    SELECT status
    FROM deployments
    WHERE deployments.app_id = apps.id
    ORDER BY number DESC
    LIMIT 1
) latest_deployment ON TRUE
WHERE apps.service_id IS NOT NULL
ON CONFLICT (id) DO NOTHING;

UPDATE compose_apps SET service_id = gen_random_uuid() WHERE service_id IS NULL;

INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
SELECT service_id, org_id, project_id, environment_id, name, 'compose', CASE LOWER(status)
    WHEN 'queued' THEN 'deploying' WHEN 'pending' THEN 'deploying' WHEN 'building' THEN 'deploying'
    WHEN 'deploying' THEN 'deploying' WHEN 'starting' THEN 'deploying' WHEN 'ready' THEN 'running'
    WHEN 'running' THEN 'running' WHEN 'healthy' THEN 'running' WHEN 'stopped' THEN 'stopped'
    WHEN 'exited' THEN 'stopped' WHEN 'failed' THEN 'failed' WHEN 'error' THEN 'failed'
    WHEN 'degraded' THEN 'degraded' ELSE 'unknown' END, created_at, created_at
FROM compose_apps
WHERE compose_apps.service_id IS NOT NULL
ON CONFLICT (id) DO NOTHING;

UPDATE databases SET service_id = gen_random_uuid() WHERE service_id IS NULL;

INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
SELECT service_id, org_id, project_id, NULL, name, 'database', CASE LOWER(status)
    WHEN 'queued' THEN 'deploying' WHEN 'pending' THEN 'deploying' WHEN 'building' THEN 'deploying'
    WHEN 'deploying' THEN 'deploying' WHEN 'starting' THEN 'deploying' WHEN 'ready' THEN 'running'
    WHEN 'running' THEN 'running' WHEN 'healthy' THEN 'running' WHEN 'stopped' THEN 'stopped'
    WHEN 'exited' THEN 'stopped' WHEN 'failed' THEN 'failed' WHEN 'error' THEN 'failed'
    WHEN 'degraded' THEN 'degraded' ELSE 'unknown' END, created_at, created_at
FROM databases
WHERE databases.service_id IS NOT NULL
ON CONFLICT (id) DO NOTHING;

DO $$ BEGIN
    ALTER TABLE apps ADD CONSTRAINT apps_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
    ALTER TABLE compose_apps ADD CONSTRAINT compose_apps_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN
    ALTER TABLE databases ADD CONSTRAINT databases_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_apps_service_id ON apps(service_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_compose_apps_service_id ON compose_apps(service_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_databases_service_id ON databases(service_id);
