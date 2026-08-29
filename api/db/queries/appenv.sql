-- name: UpsertServiceEnv :exec
INSERT INTO app_env (service_id, name, value, secret)
VALUES ($1, $2, $3, $4)
ON CONFLICT (service_id, name) DO UPDATE SET value = EXCLUDED.value, secret = EXCLUDED.secret;

-- name: ListServiceEnv :many
SELECT service_id, name, value, secret
FROM app_env
WHERE service_id = $1
ORDER BY name;

-- name: DeleteServiceEnv :exec
DELETE FROM app_env
WHERE service_id = $1 AND name = $2;
