WITH defaults AS (
    SELECT DISTINCT ON (project_id) project_id, id
    FROM environments
    ORDER BY project_id, is_default DESC, created_at, id
)
UPDATE services AS s
SET environment_id = d.id, updated_at = now()
FROM defaults AS d
WHERE s.project_id = d.project_id
  AND s.environment_id IS NULL
  AND s.deleted_at IS NULL;

UPDATE apps AS a
SET environment_id = s.environment_id, updated_at = now()
FROM services AS s
WHERE a.service_id = s.id
  AND a.environment_id IS NULL
  AND s.environment_id IS NOT NULL;

UPDATE compose_apps AS c
SET environment_id = s.environment_id
FROM services AS s
WHERE c.service_id = s.id
  AND c.environment_id IS NULL
  AND s.environment_id IS NOT NULL;

UPDATE databases AS d
SET environment_id = s.environment_id
FROM services AS s
WHERE d.service_id = s.id
  AND d.environment_id IS NULL
  AND s.environment_id IS NOT NULL;
