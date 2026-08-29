UPDATE services AS s
SET status = 'pending',
    updated_at = now()
WHERE s.status = 'unknown'
  AND ((s.kind = 'app' AND EXISTS (SELECT 1 FROM apps WHERE apps.service_id = s.id))
    OR (s.kind = 'compose' AND EXISTS (SELECT 1 FROM compose_apps WHERE compose_apps.service_id = s.id)));

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
