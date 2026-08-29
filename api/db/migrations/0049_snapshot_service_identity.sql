ALTER TABLE snapshots ADD COLUMN IF NOT EXISTS service_id UUID;
ALTER TABLE snapshot_schedules ADD COLUMN IF NOT EXISTS service_id UUID;

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

DO $$ BEGIN
    ALTER TABLE snapshots ADD CONSTRAINT snapshots_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE snapshot_schedules ADD CONSTRAINT snapshot_schedules_service_id_fkey FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_snapshots_service ON snapshots(service_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_snapshot_schedules_service ON snapshot_schedules(service_id);
