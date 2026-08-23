CREATE OR REPLACE FUNCTION prevent_terminal_deployment_status_change()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.status IN ('ready', 'failed')
       AND NEW.status <> OLD.status
       AND NOT (OLD.status = 'ready' AND NEW.status = 'rolled_back') THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS deployments_terminal_status_guard ON deployments;

CREATE TRIGGER deployments_terminal_status_guard
BEFORE UPDATE ON deployments
FOR EACH ROW
EXECUTE FUNCTION prevent_terminal_deployment_status_change();
