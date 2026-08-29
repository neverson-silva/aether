CREATE OR REPLACE FUNCTION sync_app_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        environment_id = NEW.environment_id,
        updated_at = NEW.updated_at
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION sync_compose_service_identity() RETURNS trigger AS $$
BEGIN
    UPDATE services
    SET name = NEW.name,
        environment_id = NEW.environment_id,
        status = CASE LOWER(NEW.status) WHEN 'pending' THEN 'pending' WHEN 'running' THEN 'running' WHEN 'stopped' THEN 'stopped' WHEN 'error' THEN 'failed' ELSE 'unknown' END,
        updated_at = now()
    WHERE id = NEW.service_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS apps_service_identity_sync ON apps;
CREATE TRIGGER apps_service_identity_sync AFTER UPDATE OF name, updated_at, environment_id ON apps FOR EACH ROW EXECUTE FUNCTION sync_app_service_identity();
DROP TRIGGER IF EXISTS compose_apps_service_identity_sync ON compose_apps;
CREATE TRIGGER compose_apps_service_identity_sync AFTER UPDATE OF name, status, environment_id ON compose_apps FOR EACH ROW EXECUTE FUNCTION sync_compose_service_identity();
