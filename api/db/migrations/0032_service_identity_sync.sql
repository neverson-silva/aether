CREATE OR REPLACE FUNCTION sync_app_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name, updated_at = NEW.updated_at
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_compose_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        status = CASE LOWER(NEW.status) WHEN 'running' THEN 'running' WHEN 'stopped' THEN 'stopped' WHEN 'error' THEN 'failed' ELSE 'unknown' END,
        updated_at = NEW.created_at
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_database_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        status = CASE LOWER(NEW.status) WHEN 'creating' THEN 'pending' WHEN 'starting' THEN 'deploying' WHEN 'running' THEN 'running' WHEN 'stopped' THEN 'stopped' WHEN 'failed' THEN 'failed' ELSE 'unknown' END,
        updated_at = NEW.created_at
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS apps_service_identity_sync ON apps;
CREATE TRIGGER apps_service_identity_sync AFTER UPDATE OF name, updated_at ON apps FOR EACH ROW EXECUTE FUNCTION sync_app_service_identity();
DROP TRIGGER IF EXISTS compose_apps_service_identity_sync ON compose_apps;
CREATE TRIGGER compose_apps_service_identity_sync AFTER UPDATE OF name, status ON compose_apps FOR EACH ROW EXECUTE FUNCTION sync_compose_service_identity();
DROP TRIGGER IF EXISTS databases_service_identity_sync ON databases;
CREATE TRIGGER databases_service_identity_sync AFTER UPDATE OF name, status ON databases FOR EACH ROW EXECUTE FUNCTION sync_database_service_identity();
