CREATE OR REPLACE FUNCTION assign_service_resource_identity() RETURNS trigger AS $$
DECLARE
    payload JSONB := to_jsonb(NEW);
    candidate UUID;
    legacy_id UUID;
BEGIN
    IF payload ? 'service_id' AND payload->>'service_id' IS NULL THEN
        candidate := NULL;

        IF payload ? 'database_id' AND NULLIF(payload->>'database_id', '') IS NOT NULL THEN
            legacy_id := (payload->>'database_id')::UUID;
            SELECT service_id INTO candidate FROM databases WHERE id = legacy_id;
        END IF;

        IF candidate IS NULL AND payload ? 'target_database_id' AND NULLIF(payload->>'target_database_id', '') IS NOT NULL THEN
            legacy_id := (payload->>'target_database_id')::UUID;
            SELECT service_id INTO candidate FROM databases WHERE id = legacy_id;
        END IF;

        IF candidate IS NULL AND payload ? 'target_app' AND NULLIF(payload->>'target_app', '') IS NOT NULL THEN
            legacy_id := (payload->>'target_app')::UUID;
            SELECT service_id INTO candidate FROM apps WHERE id = legacy_id;
            IF candidate IS NULL THEN
                SELECT service_id INTO candidate FROM compose_apps WHERE id = legacy_id;
            END IF;
            IF candidate IS NULL THEN
                SELECT service_id INTO candidate FROM databases WHERE id = legacy_id;
            END IF;
        END IF;

        IF candidate IS NULL AND payload ? 'app_id' AND NULLIF(payload->>'app_id', '') IS NOT NULL THEN
            legacy_id := (payload->>'app_id')::UUID;
            SELECT service_id INTO candidate FROM apps WHERE id = legacy_id;
            IF candidate IS NULL THEN
                SELECT service_id INTO candidate FROM compose_apps WHERE id = legacy_id;
            END IF;
            IF candidate IS NULL THEN
                SELECT service_id INTO candidate FROM databases WHERE id = legacy_id;
            END IF;
        END IF;

        IF candidate IS NOT NULL THEN
            NEW.service_id := candidate;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS backups_service_identity ON backups;
DROP TRIGGER IF EXISTS backup_configurations_service_identity ON backup_configurations;
DROP TRIGGER IF EXISTS backup_jobs_service_identity ON backup_jobs;
DROP TRIGGER IF EXISTS restore_jobs_service_identity ON restore_jobs;
DROP TRIGGER IF EXISTS app_env_service_identity ON app_env;
DROP TRIGGER IF EXISTS deployments_service_identity ON deployments;
DROP TRIGGER IF EXISTS domains_service_identity ON domains;
DROP TRIGGER IF EXISTS previews_service_identity ON previews;
DROP TRIGGER IF EXISTS cron_jobs_service_identity ON cron_jobs;
DROP TRIGGER IF EXISTS app_volumes_service_identity ON app_volumes;
DROP TRIGGER IF EXISTS workers_service_identity ON workers;
DROP TRIGGER IF EXISTS app_policies_service_identity ON app_policies;
DROP TRIGGER IF EXISTS autopilot_events_service_identity ON autopilot_events;
DROP TRIGGER IF EXISTS pipelines_service_identity ON pipelines;
DROP TRIGGER IF EXISTS alert_rules_service_identity ON alert_rules;
DROP TRIGGER IF EXISTS alert_events_service_identity ON alert_events;
DROP TRIGGER IF EXISTS snapshots_service_identity ON snapshots;
DROP TRIGGER IF EXISTS snapshot_schedules_service_identity ON snapshot_schedules;

CREATE TRIGGER backups_service_identity BEFORE INSERT ON backups FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER backup_configurations_service_identity BEFORE INSERT ON backup_configurations FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER backup_jobs_service_identity BEFORE INSERT ON backup_jobs FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER restore_jobs_service_identity BEFORE INSERT ON restore_jobs FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER app_env_service_identity BEFORE INSERT ON app_env FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER deployments_service_identity BEFORE INSERT ON deployments FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER domains_service_identity BEFORE INSERT ON domains FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER previews_service_identity BEFORE INSERT ON previews FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER cron_jobs_service_identity BEFORE INSERT ON cron_jobs FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER app_volumes_service_identity BEFORE INSERT ON app_volumes FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER workers_service_identity BEFORE INSERT ON workers FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER app_policies_service_identity BEFORE INSERT ON app_policies FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER autopilot_events_service_identity BEFORE INSERT ON autopilot_events FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER pipelines_service_identity BEFORE INSERT ON pipelines FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER alert_rules_service_identity BEFORE INSERT ON alert_rules FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER alert_events_service_identity BEFORE INSERT ON alert_events FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER snapshots_service_identity BEFORE INSERT ON snapshots FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
CREATE TRIGGER snapshot_schedules_service_identity BEFORE INSERT ON snapshot_schedules FOR EACH ROW EXECUTE FUNCTION assign_service_resource_identity();
