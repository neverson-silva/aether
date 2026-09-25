ALTER TABLE services DROP CONSTRAINT IF EXISTS services_status_check;
ALTER TABLE services ADD CONSTRAINT services_status_check CHECK (status IN ('pending', 'deploying', 'starting', 'stopping', 'running', 'degraded', 'stopped', 'failed', 'unknown'));
