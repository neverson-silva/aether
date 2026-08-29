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

UPDATE domains AS d
SET service_id = a.service_id
FROM apps AS a
WHERE d.service_id IS NULL
  AND d.service_type = 'app'
  AND d.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE domains AS d
SET service_id = c.service_id
FROM compose_apps AS c
WHERE d.service_id IS NULL
  AND d.service_type = 'compose'
  AND d.app_id = c.id
  AND c.service_id IS NOT NULL;

UPDATE domains AS d
SET service_id = db.service_id
FROM databases AS db
WHERE d.service_id IS NULL
  AND d.service_type IN ('db', 'database')
  AND d.app_id = db.id
  AND db.service_id IS NOT NULL;

UPDATE app_env AS e
SET service_id = a.service_id
FROM apps AS a
WHERE e.service_id IS NULL
  AND e.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE app_volumes AS v
SET service_id = a.service_id
FROM apps AS a
WHERE v.service_id IS NULL
  AND v.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE cron_jobs AS j
SET service_id = a.service_id
FROM apps AS a
WHERE j.service_id IS NULL
  AND j.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE previews AS p
SET service_id = a.service_id
FROM apps AS a
WHERE p.service_id IS NULL
  AND p.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE workers AS w
SET service_id = a.service_id
FROM apps AS a
WHERE w.service_id IS NULL
  AND w.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE app_policies AS p
SET service_id = a.service_id
FROM apps AS a
WHERE p.service_id IS NULL
  AND p.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE autopilot_events AS e
SET service_id = a.service_id
FROM apps AS a
WHERE e.service_id IS NULL
  AND e.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE pipelines AS p
SET service_id = a.service_id
FROM apps AS a
WHERE p.service_id IS NULL
  AND p.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE backups AS b
SET service_id = a.service_id
FROM apps AS a
WHERE b.service_id IS NULL
  AND b.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE backups AS b
SET service_id = db.service_id
FROM databases AS db
WHERE b.service_id IS NULL
  AND b.database_id = db.id
  AND db.service_id IS NOT NULL;

UPDATE backup_configurations AS b
SET service_id = db.service_id
FROM databases AS db
WHERE b.service_id IS NULL
  AND b.database_id = db.id
  AND db.service_id IS NOT NULL;

UPDATE backup_jobs AS b
SET service_id = db.service_id
FROM databases AS db
WHERE b.service_id IS NULL
  AND b.database_id = db.id
  AND db.service_id IS NOT NULL;

UPDATE restore_jobs AS r
SET service_id = db.service_id
FROM databases AS db
WHERE r.service_id IS NULL
  AND r.target_database_id = db.id
  AND db.service_id IS NOT NULL;

UPDATE alert_rules AS r
SET service_id = a.service_id
FROM apps AS a
WHERE r.service_id IS NULL
  AND r.target_app = a.id
  AND a.service_id IS NOT NULL;

UPDATE alert_rules AS r
SET service_id = c.service_id
FROM compose_apps AS c
WHERE r.service_id IS NULL
  AND r.target_app = c.id
  AND c.service_id IS NOT NULL;

UPDATE alert_rules AS r
SET service_id = db.service_id
FROM databases AS db
WHERE r.service_id IS NULL
  AND r.target_app = db.id
  AND db.service_id IS NOT NULL;

UPDATE alert_events AS e
SET service_id = a.service_id
FROM apps AS a
WHERE e.service_id IS NULL
  AND e.app_id = a.id
  AND a.service_id IS NOT NULL;

UPDATE alert_events AS e
SET service_id = c.service_id
FROM compose_apps AS c
WHERE e.service_id IS NULL
  AND e.app_id = c.id
  AND c.service_id IS NOT NULL;

UPDATE alert_events AS e
SET service_id = db.service_id
FROM databases AS db
WHERE e.service_id IS NULL
  AND e.app_id = db.id
  AND db.service_id IS NOT NULL;

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
