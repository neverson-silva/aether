ALTER TABLE service_sources ADD COLUMN IF NOT EXISTS compose_file TEXT NOT NULL DEFAULT 'docker-compose.yml';
