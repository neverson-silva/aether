ALTER TABLE services ALTER COLUMN status SET DEFAULT 'pending';

UPDATE services AS s
SET status = 'pending',
    updated_at = now()
WHERE s.status = 'unknown'
  AND NOT EXISTS (
      SELECT 1
      FROM deployments AS d
      WHERE d.service_id = s.id
  );

DROP TRIGGER IF EXISTS apps_service_identity_sync ON apps;
DROP TRIGGER IF EXISTS compose_apps_service_identity_sync ON compose_apps;
DROP TRIGGER IF EXISTS databases_service_identity_sync ON databases;

DROP FUNCTION IF EXISTS sync_app_service_identity();
DROP FUNCTION IF EXISTS sync_compose_service_identity();
DROP FUNCTION IF EXISTS sync_database_service_identity();

CREATE OR REPLACE FUNCTION create_database_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'database', 'pending', NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
