UPDATE deployments d
SET service_id = db.service_id
FROM databases db
WHERE d.service_id IS NULL
  AND d.app_id = db.id
  AND db.service_id IS NOT NULL;

UPDATE deployments d
SET service_id = c.service_id
FROM compose_apps c
WHERE d.service_id IS NULL
  AND d.app_id = c.id
  AND c.service_id IS NOT NULL;

CREATE OR REPLACE FUNCTION assign_app_resource_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL AND NEW.app_id IS NOT NULL THEN
        SELECT service_id INTO NEW.service_id FROM apps WHERE id = NEW.app_id;
        IF NEW.service_id IS NULL THEN
            SELECT service_id INTO NEW.service_id FROM compose_apps WHERE id = NEW.app_id;
        END IF;
        IF NEW.service_id IS NULL THEN
            SELECT service_id INTO NEW.service_id FROM databases WHERE id = NEW.app_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
