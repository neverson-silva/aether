ALTER TABLE service_sources ADD COLUMN IF NOT EXISTS legacy_app_id UUID;

DO $$ BEGIN ALTER TABLE service_sources DROP CONSTRAINT service_sources_service_id_fkey; EXCEPTION WHEN undefined_object THEN NULL; END $$;

UPDATE service_sources s
SET legacy_app_id = s.service_id
WHERE s.legacy_app_id IS NULL;

UPDATE service_sources s
SET service_id = a.service_id
FROM apps a
WHERE s.service_id = a.id;

DO $$ BEGIN ALTER TABLE service_sources ADD CONSTRAINT service_sources_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE service_sources ADD CONSTRAINT service_sources_legacy_app_id_fkey FOREIGN KEY (legacy_app_id) REFERENCES apps(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_service_sources_legacy_app ON service_sources(legacy_app_id);
