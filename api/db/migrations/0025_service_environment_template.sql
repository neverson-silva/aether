ALTER TABLE service_sources ADD COLUMN IF NOT EXISTS environment_template_path TEXT NOT NULL DEFAULT '.env.example';
