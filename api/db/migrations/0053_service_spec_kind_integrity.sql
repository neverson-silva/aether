CREATE OR REPLACE FUNCTION enforce_service_spec_kind() RETURNS trigger AS $$
DECLARE
    expected_kind TEXT;
    actual_kind TEXT;
BEGIN
    expected_kind := CASE TG_TABLE_NAME
        WHEN 'apps' THEN 'app'
        WHEN 'compose_apps' THEN 'compose'
        WHEN 'databases' THEN 'database'
    END;

    SELECT kind INTO actual_kind FROM services WHERE id = NEW.service_id;
    IF actual_kind IS NULL THEN
        RAISE EXCEPTION 'service specification requires an existing service';
    END IF;
    IF actual_kind <> expected_kind THEN
        RAISE EXCEPTION 'service specification kind does not match service kind';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS apps_service_spec_kind ON apps;
CREATE TRIGGER apps_service_spec_kind
BEFORE INSERT OR UPDATE OF service_id ON apps
FOR EACH ROW EXECUTE FUNCTION enforce_service_spec_kind();

DROP TRIGGER IF EXISTS compose_apps_service_spec_kind ON compose_apps;
CREATE TRIGGER compose_apps_service_spec_kind
BEFORE INSERT OR UPDATE OF service_id ON compose_apps
FOR EACH ROW EXECUTE FUNCTION enforce_service_spec_kind();

DROP TRIGGER IF EXISTS databases_service_spec_kind ON databases;
CREATE TRIGGER databases_service_spec_kind
BEFORE INSERT OR UPDATE OF service_id ON databases
FOR EACH ROW EXECUTE FUNCTION enforce_service_spec_kind();
