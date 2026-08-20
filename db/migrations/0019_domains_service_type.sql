ALTER TABLE domains DROP CONSTRAINT IF EXISTS domains_app_id_fkey;
ALTER TABLE domains ADD COLUMN service_type TEXT NOT NULL DEFAULT 'app';