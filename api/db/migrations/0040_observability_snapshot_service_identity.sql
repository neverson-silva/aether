ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE alert_events ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE snapshot_schedules ADD COLUMN IF NOT EXISTS service_id UUID;

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

UPDATE snapshots AS s
SET service_id = a.service_id
FROM apps AS a
WHERE s.service_id IS NULL
  AND s.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE snapshot_schedules AS s
SET service_id = a.service_id
FROM apps AS a
WHERE s.service_id IS NULL
  AND s.app_id = a.id
  AND a.service_id IS NOT NULL;

DO $$ BEGIN ALTER TABLE alert_rules ADD CONSTRAINT alert_rules_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE alert_events ADD CONSTRAINT alert_events_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE snapshots ADD CONSTRAINT snapshots_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;
DO $$ BEGIN ALTER TABLE snapshot_schedules ADD CONSTRAINT snapshot_schedules_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE; EXCEPTION WHEN duplicate_object THEN NULL; END $$;

CREATE INDEX IF NOT EXISTS idx_alert_rules_service ON alert_rules(service_id);
CREATE INDEX IF NOT EXISTS idx_alert_events_service ON alert_events(service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshots_service ON snapshots(service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshot_schedules_service ON snapshot_schedules(service_id);

DROP TRIGGER IF EXISTS alert_events_service_identity ON alert_events;
CREATE TRIGGER alert_events_service_identity BEFORE INSERT ON alert_events FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS snapshots_service_identity ON snapshots;
CREATE TRIGGER snapshots_service_identity BEFORE INSERT ON snapshots FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
DROP TRIGGER IF EXISTS snapshot_schedules_service_identity ON snapshot_schedules;
CREATE TRIGGER snapshot_schedules_service_identity BEFORE INSERT ON snapshot_schedules FOR EACH ROW EXECUTE FUNCTION assign_app_resource_service_identity();
