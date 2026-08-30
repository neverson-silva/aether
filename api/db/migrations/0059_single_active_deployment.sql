UPDATE deployments AS d
SET service_id = a.service_id
FROM apps AS a
WHERE d.service_id IS NULL
  AND d.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE deployments AS d
SET service_id = c.service_id
FROM compose_apps AS c
WHERE d.service_id IS NULL
  AND d.app_id = c.id
  AND c.service_id IS NOT NULL;

UPDATE deployments AS d
SET service_id = db.service_id
FROM databases AS db
WHERE d.service_id IS NULL
  AND d.app_id = db.id
  AND db.service_id IS NOT NULL;

WITH active_deployments AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY service_id ORDER BY created_at, id) AS position
    FROM deployments
    WHERE service_id IS NOT NULL
      AND status IN ('queued', 'building', 'starting', 'health_checking')
)
UPDATE deployments AS d
SET status = 'cancelled',
    error = 'superseded during deployment queue cutover',
    finished_at = now()
FROM active_deployments AS a
WHERE d.id = a.id
  AND a.position > 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_deployments_one_active_service
ON deployments(service_id)
WHERE service_id IS NOT NULL
  AND status IN ('queued', 'building', 'starting', 'health_checking');
