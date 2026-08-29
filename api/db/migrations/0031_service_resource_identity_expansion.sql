ALTER TABLE workers ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE app_policies ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE autopilot_events ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE backup_configurations ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE backup_jobs ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE restore_jobs ADD COLUMN IF NOT EXISTS service_id UUID;

UPDATE workers w
SET service_id = a.service_id
FROM apps a
WHERE w.app_id = a.id AND w.service_id IS NULL;

UPDATE app_policies p
SET service_id = a.service_id
FROM apps a
WHERE p.app_id = a.id AND p.service_id IS NULL;

UPDATE autopilot_events e
SET service_id = a.service_id
FROM apps a
WHERE e.app_id = a.id AND e.service_id IS NULL;

UPDATE pipelines p
SET service_id = a.service_id
FROM apps a
WHERE p.app_id = a.id AND p.service_id IS NULL;

UPDATE backup_configurations b
SET service_id = d.service_id
FROM databases d
WHERE b.database_id = d.id AND b.service_id IS NULL;

UPDATE backup_jobs b
SET service_id = d.service_id
FROM databases d
WHERE b.database_id = d.id AND b.service_id IS NULL;

UPDATE restore_jobs r
SET service_id = d.service_id
FROM databases d
WHERE r.target_database_id = d.id AND r.service_id IS NULL;

DO $$ BEGIN ALTER TABLE workers ADD CONSTRAINT workers_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE app_policies ADD CONSTRAINT app_policies_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE autopilot_events ADD CONSTRAINT autopilot_events_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE pipelines ADD CONSTRAINT pipelines_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE backup_configurations ADD CONSTRAINT backup_configurations_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE backup_jobs ADD CONSTRAINT backup_jobs_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE restore_jobs ADD CONSTRAINT restore_jobs_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_workers_service ON workers(service_id);
CREATE INDEX IF NOT EXISTS idx_app_policies_service ON app_policies(service_id);
CREATE INDEX IF NOT EXISTS idx_autopilot_events_service ON autopilot_events(service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipelines_service ON pipelines(service_id);
CREATE INDEX IF NOT EXISTS idx_backup_configurations_service ON backup_configurations(service_id);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_service ON backup_jobs(service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_restore_jobs_service ON restore_jobs(service_id);
