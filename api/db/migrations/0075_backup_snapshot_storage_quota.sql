CREATE OR REPLACE FUNCTION enforce_organization_archive_quota()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    organization_id uuid;
    used_bytes bigint;
BEGIN
    IF TG_TABLE_NAME = 'snapshots' THEN
        organization_id := NEW.org_id;
        PERFORM pg_advisory_xact_lock(hashtextextended(organization_id::text, 0));
        SELECT COALESCE(SUM(size), 0)
          INTO used_bytes
          FROM snapshots s
         WHERE s.org_id = organization_id
           AND s.id <> NEW.id;
        IF used_bytes < 0 OR used_bytes > 536870912000 - NEW.size THEN
            RAISE EXCEPTION 'organization snapshot quota exceeded' USING ERRCODE = 'check_violation';
        END IF;
    ELSE
        SELECT org_id INTO organization_id FROM databases WHERE id = NEW.database_id;
        PERFORM pg_advisory_xact_lock(hashtextextended(organization_id::text, 0));
        SELECT COALESCE(SUM(size_bytes), 0)
          INTO used_bytes
          FROM backup_jobs bj
          JOIN databases d ON d.id = bj.database_id
         WHERE d.org_id = organization_id
           AND bj.status = 'completed'
           AND bj.id <> NEW.id;
        IF NEW.status = 'completed' AND (used_bytes < 0 OR used_bytes > 536870912000 - NEW.size_bytes) THEN
            RAISE EXCEPTION 'organization backup quota exceeded' USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END;
$$;

ALTER TABLE backup_jobs
    ADD CONSTRAINT backup_jobs_size_limit CHECK (size_bytes >= 0) NOT VALID;

DROP TRIGGER IF EXISTS backup_jobs_storage_quota_guard ON backup_jobs;
CREATE TRIGGER backup_jobs_storage_quota_guard
BEFORE INSERT OR UPDATE OF database_id, status, size_bytes ON backup_jobs
FOR EACH ROW EXECUTE FUNCTION enforce_organization_archive_quota();

DROP TRIGGER IF EXISTS snapshots_storage_quota_guard ON snapshots;
CREATE TRIGGER snapshots_storage_quota_guard
BEFORE INSERT OR UPDATE OF org_id, size ON snapshots
FOR EACH ROW EXECUTE FUNCTION enforce_organization_archive_quota();
