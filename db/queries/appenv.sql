-- name: UpsertAppEnv :exec
INSERT INTO app_env (app_id, name, value, secret)
VALUES ($1, $2, $3, $4)
ON CONFLICT (app_id, name) DO UPDATE SET value = EXCLUDED.value, secret = EXCLUDED.secret;

-- name: ListAppEnv :many
SELECT app_id, name, value, secret
FROM app_env
WHERE app_id = $1
ORDER BY name;

-- name: DeleteAppEnv :exec
DELETE FROM app_env
WHERE app_id = $1 AND name = $2;
