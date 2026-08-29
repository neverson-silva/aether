CREATE OR REPLACE FUNCTION assign_alert_rule_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL AND NEW.target_app IS NOT NULL THEN
        SELECT a.service_id INTO NEW.service_id FROM apps AS a WHERE a.id = NEW.target_app;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION assign_alert_event_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL AND NEW.app_id IS NOT NULL THEN
        SELECT a.service_id INTO NEW.service_id FROM apps AS a WHERE a.id = NEW.app_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS alert_rules_service_identity ON alert_rules;
CREATE TRIGGER alert_rules_service_identity BEFORE INSERT ON alert_rules FOR EACH ROW EXECUTE FUNCTION assign_alert_rule_service_identity();
DROP TRIGGER IF EXISTS alert_events_service_identity ON alert_events;
CREATE TRIGGER alert_events_service_identity BEFORE INSERT ON alert_events FOR EACH ROW EXECUTE FUNCTION assign_alert_event_service_identity();
