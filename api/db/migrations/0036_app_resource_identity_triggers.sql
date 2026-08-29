ALTER TABLE previews ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE cron_jobs ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE app_volumes ADD COLUMN IF NOT EXISTS service_id UUID;

CREATE OR REPLACE FUNCTION assign_app_resource_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL AND NEW.app_id IS NOT NULL THEN
        SELECT service_id INTO NEW.service_id FROM apps WHERE id = NEW.app_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS app_env_service_identity ON app_env;
CREATE TRIGGER app_env_service_identity BEFORE INSERT ON app_env FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS deployments_service_identity ON deployments;
CREATE TRIGGER deployments_service_identity BEFORE INSERT ON deployments FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS domains_service_identity ON domains;
CREATE TRIGGER domains_service_identity BEFORE INSERT ON domains FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS previews_service_identity ON previews;
CREATE TRIGGER previews_service_identity BEFORE INSERT ON previews FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS cron_jobs_service_identity ON cron_jobs;
CREATE TRIGGER cron_jobs_service_identity BEFORE INSERT ON cron_jobs FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS app_volumes_service_identity ON app_volumes;
CREATE TRIGGER app_volumes_service_identity BEFORE INSERT ON app_volumes FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS workers_service_identity ON workers;
CREATE TRIGGER workers_service_identity BEFORE INSERT ON workers FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS app_policies_service_identity ON app_policies;
CREATE TRIGGER app_policies_service_identity BEFORE INSERT ON app_policies FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS autopilot_events_service_identity ON autopilot_events;
CREATE TRIGGER autopilot_events_service_identity BEFORE INSERT ON autopilot_events FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS pipelines_service_identity ON pipelines;
CREATE TRIGGER pipelines_service_identity BEFORE INSERT ON pipelines FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
