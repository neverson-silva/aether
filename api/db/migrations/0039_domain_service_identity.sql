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
