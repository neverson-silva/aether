ALTER TABLE services DROP CONSTRAINT IF EXISTS services_status_check;
UPDATE services SET status = 'unknown' WHERE status NOT IN ('pending', 'deploying', 'running', 'degraded', 'stopped', 'failed', 'unknown');
ALTER TABLE services ADD CONSTRAINT services_status_check CHECK (status IN ('pending', 'deploying', 'running', 'degraded', 'stopped', 'failed', 'unknown'));

UPDATE services AS s
SET status = CASE LOWER(c.status)
    WHEN 'pending' THEN 'pending'
    WHEN 'deploying' THEN 'deploying'
    WHEN 'running' THEN 'running'
    WHEN 'stopped' THEN 'stopped'
    WHEN 'error' THEN 'failed'
    ELSE 'unknown'
END,
updated_at = now()
FROM compose_apps AS c
WHERE c.service_id = s.id;

UPDATE services AS s
SET status = CASE LOWER(d.status)
    WHEN 'creating' THEN 'pending'
    WHEN 'starting' THEN 'deploying'
    WHEN 'running' THEN 'running'
    WHEN 'stopped' THEN 'stopped'
    WHEN 'failed' THEN 'failed'
    ELSE 'unknown'
END,
updated_at = now()
FROM databases AS d
WHERE d.service_id = s.id;

CREATE OR REPLACE FUNCTION create_compose_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'compose', CASE LOWER(NEW.status)
            WHEN 'pending' THEN 'pending'
            WHEN 'deploying' THEN 'deploying'
            WHEN 'running' THEN 'running'
            WHEN 'stopped' THEN 'stopped'
            WHEN 'error' THEN 'failed'
            ELSE 'unknown'
        END, NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_compose_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        project_id = NEW.project_id,
        environment_id = NEW.environment_id,
        status = CASE LOWER(NEW.status)
            WHEN 'pending' THEN 'pending'
            WHEN 'deploying' THEN 'deploying'
            WHEN 'running' THEN 'running'
            WHEN 'stopped' THEN 'stopped'
            WHEN 'error' THEN 'failed'
            ELSE 'unknown'
        END,
        updated_at = now()
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_database_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'database', CASE LOWER(NEW.status)
            WHEN 'creating' THEN 'pending'
            WHEN 'starting' THEN 'deploying'
            WHEN 'running' THEN 'running'
            WHEN 'stopped' THEN 'stopped'
            WHEN 'failed' THEN 'failed'
            ELSE 'unknown'
        END, NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
