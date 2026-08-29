ALTER TABLE previews ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE cron_jobs ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE app_volumes ADD COLUMN IF NOT EXISTS service_id UUID;

UPDATE previews AS p
SET service_id = a.service_id
FROM apps AS a
WHERE p.service_id IS NULL
  AND p.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE cron_jobs AS j
SET service_id = a.service_id
FROM apps AS a
WHERE j.service_id IS NULL
  AND j.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE app_volumes AS v
SET service_id = a.service_id
FROM apps AS a
WHERE v.service_id IS NULL
  AND v.app_id = a.id
  AND a.service_id IS NOT NULL;

DO $$ BEGIN ALTER TABLE previews ADD CONSTRAINT previews_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE cron_jobs ADD CONSTRAINT cron_jobs_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE app_volumes ADD CONSTRAINT app_volumes_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_previews_service ON previews(service_id);
CREATE INDEX IF NOT EXISTS idx_cron_service ON cron_jobs(service_id);
CREATE INDEX IF NOT EXISTS idx_app_volumes_service ON app_volumes(service_id);

CREATE OR REPLACE FUNCTION assign_app_resource_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL AND NEW.app_id IS NOT NULL THEN
        SELECT service_id INTO NEW.service_id FROM apps WHERE id = NEW.app_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS previews_service_identity ON previews;
CREATE TRIGGER previews_service_identity BEFORE INSERT ON previews FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS cron_jobs_service_identity ON cron_jobs;
CREATE TRIGGER cron_jobs_service_identity BEFORE INSERT ON cron_jobs FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS app_volumes_service_identity ON app_volumes;
CREATE TRIGGER app_volumes_service_identity BEFORE INSERT ON app_volumes FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
