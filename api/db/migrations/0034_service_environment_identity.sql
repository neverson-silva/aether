ALTER TABLE app_env DROP CONSTRAINT IF EXISTS app_env_pkey;
ALTER TABLE app_env ALTER COLUMN app_id DROP NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_env_app_name ON app_env(app_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_env_service_name ON app_env(service_id, name);
