ALTER TABLE databases ADD COLUMN IF NOT EXISTS environment_id UUID REFERENCES environments(id) ON DELETE SET NULL;

UPDATE databases AS d
SET environment_id = COALESCE(
    (SELECT e.id FROM environments AS e WHERE e.project_id = d.project_id AND e.is_default ORDER BY e.created_at, e.id LIMIT 1),
    (SELECT e.id FROM environments AS e WHERE e.project_id = d.project_id ORDER BY e.created_at, e.id LIMIT 1)
)
WHERE d.environment_id IS NULL;

UPDATE services AS s
SET environment_id = d.environment_id,
    updated_at = now()
FROM databases AS d
WHERE d.service_id = s.id
  AND d.environment_id IS NOT NULL
  AND s.environment_id IS DISTINCT FROM d.environment_id;

CREATE INDEX IF NOT EXISTS idx_databases_environment ON databases(environment_id);

CREATE OR REPLACE FUNCTION create_database_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'database', 'unknown', NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_database_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        environment_id = NEW.environment_id,
        status = CASE LOWER(NEW.status)
            WHEN 'creating' THEN 'pending'
            WHEN 'starting' THEN 'deploying'
            WHEN 'running' THEN 'running'
            WHEN 'stopped' THEN 'stopped'
            WHEN 'failed' THEN 'failed'
            ELSE 'unknown'
        END,
        updated_at = now()
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS databases_service_identity_sync ON databases;
CREATE TRIGGER databases_service_identity_sync AFTER UPDATE OF name, status, environment_id ON databases FOR EACH ROW EXECUTE FUNCTION sync_database_service_identity();
