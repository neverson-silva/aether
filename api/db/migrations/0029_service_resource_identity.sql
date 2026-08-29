ALTER TABLE deployments ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE domains ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE app_env ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE cron_jobs ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE previews ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE app_volumes ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE backups ADD COLUMN IF NOT EXISTS service_id UUID;

UPDATE deployments d
SET service_id = a.service_id
FROM apps a
WHERE d.app_id = a.id AND d.service_id IS NULL;

UPDATE domains d
SET service_id = a.service_id
FROM apps a
WHERE d.service_id IS NULL AND d.service_type = 'app' AND d.app_id = a.id;

UPDATE domains d
SET service_id = ca.service_id
FROM compose_apps ca
WHERE d.service_id IS NULL AND d.service_type = 'compose' AND d.app_id = ca.id;

UPDATE domains d
SET service_id = db.service_id
FROM databases db
WHERE d.service_id IS NULL AND d.service_type = 'db' AND d.app_id = db.id;

UPDATE app_env e
SET service_id = a.service_id
FROM apps a
WHERE e.app_id = a.id AND e.service_id IS NULL;

UPDATE cron_jobs j
SET service_id = a.service_id
FROM apps a
WHERE j.app_id = a.id AND j.service_id IS NULL;

UPDATE previews p
SET service_id = a.service_id
FROM apps a
WHERE p.app_id = a.id AND p.service_id IS NULL;

UPDATE app_volumes v
SET service_id = a.service_id
FROM apps a
WHERE v.app_id = a.id AND v.service_id IS NULL;

UPDATE backups b
SET service_id = a.service_id
FROM apps a
WHERE b.service_id IS NULL AND b.app_id = a.id;

UPDATE backups b
SET service_id = db.service_id
FROM databases db
WHERE b.service_id IS NULL AND b.database_id = db.id;

DO $$ BEGIN ALTER TABLE deployments ADD CONSTRAINT deployments_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE domains ADD CONSTRAINT domains_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE app_env ADD CONSTRAINT app_env_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE cron_jobs ADD CONSTRAINT cron_jobs_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE previews ADD CONSTRAINT previews_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE app_volumes ADD CONSTRAINT app_volumes_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE backups ADD CONSTRAINT backups_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_deployments_service ON deployments(service_id, number DESC);
CREATE INDEX IF NOT EXISTS idx_domains_service ON domains(service_id);
CREATE INDEX IF NOT EXISTS idx_app_env_service ON app_env(service_id);
CREATE INDEX IF NOT EXISTS idx_cron_jobs_service ON cron_jobs(service_id);
CREATE INDEX IF NOT EXISTS idx_previews_service ON previews(service_id);
CREATE INDEX IF NOT EXISTS idx_app_volumes_service ON app_volumes(service_id);
CREATE INDEX IF NOT EXISTS idx_backups_service ON backups(service_id);
