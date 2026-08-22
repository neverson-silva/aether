-- Monitoring history persistence. Survives API container restarts and
-- supports 6h/24h/7d windows. Retention is enforced by the collector (purge
-- older than 7 days); see internal/monitoring.

CREATE TABLE IF NOT EXISTS monitoring_samples (
    ts        TIMESTAMPTZ NOT NULL,
    host_cpu  DOUBLE PRECISION NOT NULL DEFAULT 0,
    host_mem  DOUBLE PRECISION NOT NULL DEFAULT 0,
    aether_cpu DOUBLE PRECISION NOT NULL DEFAULT 0,
    aether_mem BIGINT NOT NULL DEFAULT 0,
    aether_mem_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    user_cpu  DOUBLE PRECISION NOT NULL DEFAULT 0,
    user_mem  BIGINT NOT NULL DEFAULT 0,
    user_mem_pct DOUBLE PRECISION NOT NULL DEFAULT 0,
    net_rx    DOUBLE PRECISION NOT NULL DEFAULT 0,
    net_tx    DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_monitoring_samples_ts ON monitoring_samples (ts);

CREATE TABLE IF NOT EXISTS monitoring_resource_samples (
    ts         TIMESTAMPTZ NOT NULL,
    resource_id TEXT NOT NULL,
    name       TEXT NOT NULL DEFAULT '',
    owner      TEXT NOT NULL DEFAULT 'unknown',
    cpu        DOUBLE PRECISION NOT NULL DEFAULT 0,
    mem        BIGINT NOT NULL DEFAULT 0,
    net_rx     DOUBLE PRECISION NOT NULL DEFAULT 0,
    net_tx     DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_monitoring_resource_samples_res_ts ON monitoring_resource_samples (resource_id, ts);
