-- name: UpsertEnvVariable :one
INSERT INTO env_variables (project_id, environment_id, key, value, is_secret)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (project_id, environment_id, key) WHERE environment_id IS NOT NULL
DO UPDATE SET value = EXCLUDED.value, is_secret = EXCLUDED.is_secret, updated_at = now()
RETURNING id, project_id, environment_id, key, value, is_secret, created_at, updated_at;

-- name: UpsertProjectVariable :one
INSERT INTO env_variables (project_id, environment_id, key, value, is_secret)
VALUES ($1, NULL, $2, $3, $4)
ON CONFLICT (project_id, key) WHERE environment_id IS NULL
DO UPDATE SET value = EXCLUDED.value, is_secret = EXCLUDED.is_secret, updated_at = now()
RETURNING id, project_id, environment_id, key, value, is_secret, created_at, updated_at;

-- name: ListEnvVariables :many
SELECT id, project_id, environment_id, key, value, is_secret, created_at, updated_at
FROM env_variables
WHERE project_id = $1 AND environment_id = $2
ORDER BY key;

-- name: ListProjectVariables :many
SELECT id, project_id, environment_id, key, value, is_secret, created_at, updated_at
FROM env_variables
WHERE project_id = $1 AND environment_id IS NULL
ORDER BY key;

-- name: DeleteEnvVariable :exec
DELETE FROM env_variables
WHERE project_id = $1 AND environment_id = $2 AND key = $3;

-- name: DeleteProjectVariable :exec
DELETE FROM env_variables
WHERE project_id = $1 AND environment_id IS NULL AND key = $2;

-- name: RecordVariableAudit :exec
INSERT INTO variable_audit (project_id, environment_id, user_id, action, key)
VALUES ($1, $2, $3, $4, $5);

-- name: ListVariableAudit :many
SELECT id, project_id, environment_id, user_id, action, key, created_at
FROM variable_audit
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: SetDefaultEnvironment :exec
UPDATE environments
SET is_default = (id = $2), updated_at = now()
WHERE project_id = $1;
