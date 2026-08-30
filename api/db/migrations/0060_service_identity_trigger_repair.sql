ALTER TABLE services ALTER COLUMN status SET DEFAULT 'pending';

UPDATE services AS s
SET status = 'pending',
    updated_at = now()
WHERE s.status = 'unknown'
  AND NOT EXISTS (SELECT 1 FROM deployments AS d WHERE d.service_id = s.id);

CREATE OR REPLACE FUNCTION create_app_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'app', 'pending', NEW.created_at, NEW.updated_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_compose_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'compose', 'pending', NEW.created_at, NEW.updated_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_database_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'database', 'pending', NEW.created_at, NEW.updated_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_app_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services SET name = NEW.name, project_id = NEW.project_id, environment_id = NEW.environment_id, updated_at = now() WHERE id = NEW.service_id;
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

CREATE OR REPLACE FUNCTION sync_database_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        project_id = NEW.project_id,
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

DROP TRIGGER IF EXISTS apps_service_identity ON apps;
CREATE TRIGGER apps_service_identity BEFORE INSERT ON apps FOR EACH ROW EXECUTE FUNCTION create_app_service_identity();
DROP TRIGGER IF EXISTS compose_apps_service_identity ON compose_apps;
CREATE TRIGGER compose_apps_service_identity BEFORE INSERT ON compose_apps FOR EACH ROW EXECUTE FUNCTION create_compose_service_identity();
DROP TRIGGER IF EXISTS databases_service_identity ON databases;
CREATE TRIGGER databases_service_identity BEFORE INSERT ON databases FOR EACH ROW EXECUTE FUNCTION create_database_service_identity();

DROP TRIGGER IF EXISTS apps_service_identity_sync ON apps;
CREATE TRIGGER apps_service_identity_sync AFTER UPDATE OF name, project_id, environment_id, updated_at ON apps FOR EACH ROW EXECUTE FUNCTION sync_app_service_identity();
DROP TRIGGER IF EXISTS compose_apps_service_identity_sync ON compose_apps;
CREATE TRIGGER compose_apps_service_identity_sync AFTER UPDATE OF name, project_id, environment_id, status ON compose_apps FOR EACH ROW EXECUTE FUNCTION sync_compose_service_identity();
DROP TRIGGER IF EXISTS databases_service_identity_sync ON databases;
CREATE TRIGGER databases_service_identity_sync AFTER UPDATE OF name, project_id, environment_id, status ON databases FOR EACH ROW EXECUTE FUNCTION sync_database_service_identity();
