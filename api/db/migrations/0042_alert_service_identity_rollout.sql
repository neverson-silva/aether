ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE alert_events ADD COLUMN IF NOT EXISTS service_id UUID;

UPDATE alert_rules AS r
SET service_id = a.service_id
FROM apps AS a
WHERE r.service_id IS NULL
  AND r.target_app = a.id
  AND a.service_id IS NOT NULL;

UPDATE alert_events AS e
SET service_id = a.service_id
FROM apps AS a
WHERE e.service_id IS NULL
  AND e.app_id = a.id
  AND a.service_id IS NOT NULL;

DO $$ BEGIN ALTER TABLE alert_rules ADD CONSTRAINT alert_rules_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE alert_events ADD CONSTRAINT alert_events_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_alert_rules_service ON alert_rules(service_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_service ON alert_events(service_id, created_at DESC);

CREATE OR REPLACE FUNCTION assign_alert_rule_service_identity() RETURNS trigger AS $$
BEGIN
    IF NEW.service_id IS NULL THEN
        IF NEW.target_app IS NOT NULL THEN
            SELECT a.service_id INTO NEW.service_id FROM apps AS a WHERE a.id = NEW.target_app;
        END IF;
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
