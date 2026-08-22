-- Escopos separados de variáveis: projeto (environment_id NULL) vs ambiente.
ALTER TABLE env_variables ALTER COLUMN environment_id DROP NOT NULL;

DO $$
DECLARE cname text;
BEGIN
  SELECT conname INTO cname FROM pg_constraint
    WHERE conrelid = 'env_variables'::regclass AND contype = 'u' LIMIT 1;
  IF cname IS NOT NULL THEN
    EXECUTE format('ALTER TABLE env_variables DROP CONSTRAINT %I', cname);
  END IF;
END $$;

CREATE UNIQUE INDEX uniq_env_variables_project ON env_variables (project_id, key) WHERE environment_id IS NULL;
CREATE UNIQUE INDEX uniq_env_variables_env ON env_variables (project_id, environment_id, key) WHERE environment_id IS NOT NULL;
