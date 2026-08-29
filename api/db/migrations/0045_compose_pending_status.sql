ALTER TABLE compose_apps DROP CONSTRAINT IF EXISTS compose_apps_status_check;
ALTER TABLE compose_apps ADD CONSTRAINT compose_apps_status_check CHECK (status IN ('pending', 'stopped', 'running', 'error'));
