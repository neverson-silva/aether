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
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'compose', 'pending', NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_database_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.name, 'database', 'unknown', NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS apps_service_identity ON apps;
CREATE TRIGGER apps_service_identity BEFORE INSERT ON apps FOR EACH ROW EXECUTE FUNCTION create_app_service_identity();
DROP TRIGGER IF EXISTS compose_apps_service_identity ON compose_apps;
CREATE TRIGGER compose_apps_service_identity BEFORE INSERT ON compose_apps FOR EACH ROW EXECUTE FUNCTION create_compose_service_identity();
DROP TRIGGER IF EXISTS databases_service_identity ON databases;
CREATE TRIGGER databases_service_identity BEFORE INSERT ON databases FOR EACH ROW EXECUTE FUNCTION create_database_service_identity();
