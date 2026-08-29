CREATE OR REPLACE FUNCTION assign_backup_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL AND NEW.database_id IS NOT NULL THEN
        SELECT service_id INTO NEW.service_id FROM databases WHERE id = NEW.database_id;
    ELSIF NEW.service_id IS NULL AND NEW.app_id IS NOT NULL THEN
        SELECT service_id INTO NEW.service_id FROM apps WHERE id = NEW.app_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION assign_database_backup_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        SELECT service_id INTO NEW.service_id FROM databases WHERE id = NEW.database_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION assign_restore_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        SELECT service_id INTO NEW.service_id FROM databases WHERE id = NEW.target_database_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS backups_service_identity ON backups;
CREATE TRIGGER backups_service_identity BEFORE INSERT ON backups FOR EACH ROW EXECUTE FUNCTION assign_backup_service_identity();
DROP TRIGGER IF EXISTS backup_configurations_service_identity ON backup_configurations;
CREATE TRIGGER backup_configurations_service_identity BEFORE INSERT ON backup_configurations FOR EACH ROW EXECUTE FUNCTION assign_database_backup_service_identity();
DROP TRIGGER IF EXISTS backup_jobs_service_identity ON backup_jobs;
CREATE TRIGGER backup_jobs_service_identity BEFORE INSERT ON backup_jobs FOR EACH ROW EXECUTE FUNCTION assign_database_backup_service_identity();
DROP TRIGGER IF EXISTS restore_jobs_service_identity ON restore_jobs;
CREATE TRIGGER restore_jobs_service_identity BEFORE INSERT ON restore_jobs FOR EACH ROW EXECUTE FUNCTION assign_restore_service_identity();
