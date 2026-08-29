UPDATE services AS s
SET environment_id = d.environment_id,
    updated_at = now()
FROM databases AS d
WHERE d.service_id = s.id
  AND s.kind = 'database'
  AND s.environment_id IS DISTINCT FROM d.environment_id;

CREATE OR REPLACE FUNCTION create_database_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        NEW.service_id := gen_random_uuid();
        INSERT INTO services (id, org_id, project_id, environment_id, name, kind, status, created_at, updated_at)
        VALUES (NEW.service_id, NEW.org_id, NEW.project_id, NEW.environment_id, NEW.name, 'database', CASE LOWER(NEW.status)
            WHEN 'creating' THEN 'pending'
            WHEN 'starting' THEN 'deploying'
            WHEN 'running' THEN 'running'
            WHEN 'stopped' THEN 'stopped'
            WHEN 'failed' THEN 'failed'
            ELSE 'unknown'
        END, NEW.created_at, NEW.created_at)
        ON CONFLICT (id) DO NOTHING;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
