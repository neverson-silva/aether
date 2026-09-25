ALTER TABLE services DROP CONSTRAINT IF EXISTS services_status_check;
ALTER TABLE services ADD CONSTRAINT services_status_check CHECK (status IN ('pending', 'deploying', 'starting', 'stopping', 'running', 'degraded', 'stopped', 'failed', 'unknown'));

CREATE TABLE IF NOT EXISTS service_lifecycle_operations (
    id UUID PRIMARY KEY,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    org_id UUID NOT NULL,
    spec_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('app', 'compose', 'database')),
    action TEXT NOT NULL CHECK (action IN ('start', 'stop')),
    status TEXT NOT NULL CHECK (status IN ('accepted', 'running', 'succeeded', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    lease_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS service_lifecycle_operations_service_idx
    ON service_lifecycle_operations (service_id, created_at DESC);

CREATE INDEX IF NOT EXISTS service_lifecycle_operations_recovery_idx
    ON service_lifecycle_operations (status, lease_until, created_at)
    WHERE status IN ('accepted', 'running');

CREATE UNIQUE INDEX IF NOT EXISTS service_lifecycle_operations_active_idx
    ON service_lifecycle_operations (service_id)
    WHERE status IN ('accepted', 'running');
