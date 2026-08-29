ALTER TABLE deployments ALTER COLUMN app_id DROP NOT NULL;
ALTER TABLE domains ALTER COLUMN app_id DROP NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_service_number
ON deployments(service_id, number)
WHERE service_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_service_host
ON domains(service_id, host)
WHERE service_id IS NOT NULL;
