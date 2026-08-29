ALTER TABLE compose_apps DROP CONSTRAINT IF EXISTS compose_apps_status_check;
ALTER TABLE compose_apps ADD CONSTRAINT compose_apps_status_check CHECK (status IN ('pending', 'deploying', 'stopped', 'running', 'error'));

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
