ALTER TABLE apps
    ADD CONSTRAINT apps_memory_limit CHECK (mem_mb BETWEEN 0 AND 2048) NOT VALID;

ALTER TABLE apps
    ADD CONSTRAINT apps_storage_limit CHECK (storage_mb BETWEEN 0 AND 102400) NOT VALID;

ALTER TABLE databases
    ADD CONSTRAINT databases_memory_limit CHECK (mem_mb BETWEEN 0 AND 2048) NOT VALID;

ALTER TABLE databases
    ADD CONSTRAINT databases_storage_limit CHECK (storage_mb BETWEEN 0 AND 102400) NOT VALID;

CREATE OR REPLACE FUNCTION enforce_organization_storage_quota()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    used_storage bigint;
BEGIN
    PERFORM pg_advisory_xact_lock(hashtextextended(NEW.org_id::text, 0));
    IF TG_TABLE_NAME = 'apps' THEN
        SELECT COALESCE((SELECT SUM(storage_mb) FROM apps WHERE org_id = NEW.org_id AND id <> NEW.id), 0)
             + COALESCE((SELECT SUM(storage_mb) FROM databases WHERE org_id = NEW.org_id), 0)
          INTO used_storage;
    ELSE
        SELECT COALESCE((SELECT SUM(storage_mb) FROM apps WHERE org_id = NEW.org_id), 0)
             + COALESCE((SELECT SUM(storage_mb) FROM databases WHERE org_id = NEW.org_id AND id <> NEW.id), 0)
          INTO used_storage;
    END IF;
    IF used_storage < 0 OR used_storage > 512000 - NEW.storage_mb THEN
        RAISE EXCEPTION 'organization storage quota exceeded' USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS apps_storage_quota_guard ON apps;
CREATE TRIGGER apps_storage_quota_guard
BEFORE INSERT OR UPDATE OF org_id, storage_mb ON apps
FOR EACH ROW EXECUTE FUNCTION enforce_organization_storage_quota();

DROP TRIGGER IF EXISTS databases_storage_quota_guard ON databases;
CREATE TRIGGER databases_storage_quota_guard
BEFORE INSERT OR UPDATE OF org_id, storage_mb ON databases
FOR EACH ROW EXECUTE FUNCTION enforce_organization_storage_quota();
