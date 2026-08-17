-- name: ListTemplates :many
SELECT id, name, description, category, icon, version, definition, compose_yaml, readme, homepage, github,
    license, installs, featured, verified, editors_choice, tags, updated_at
FROM templates
WHERE ($1 = '' OR category = $1)
  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
  AND ($3 = FALSE OR featured = TRUE)
  AND ($4 = FALSE OR verified = TRUE)
  AND ($5 = FALSE OR editors_choice = TRUE)
ORDER BY featured DESC, installs DESC, name;

-- name: GetTemplate :one
SELECT id, name, description, category, icon, version, definition, compose_yaml, readme, homepage, github,
    license, installs, featured, verified, editors_choice, tags, updated_at
FROM templates
WHERE id = $1;

-- name: IncrementTemplateInstalls :exec
UPDATE templates
SET installs = installs + 1
WHERE id = $1;

-- name: CreateComposeApp :one
INSERT INTO compose_apps (org_id, project_id, environment_id, name, compose, status)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, org_id, project_id, environment_id, name, compose, status, created_at;

-- name: GetComposeApp :one
SELECT id, org_id, project_id, environment_id, name, compose, status, created_at
FROM compose_apps
WHERE id = $1;

-- name: ListComposeAppsByOrg :many
SELECT id, org_id, project_id, environment_id, name, compose, status, created_at
FROM compose_apps
WHERE org_id = $1
ORDER BY name;

-- name: DeleteComposeApp :exec
DELETE FROM compose_apps
WHERE id = $1 AND org_id = $2;

-- name: SetComposeStatus :exec
UPDATE compose_apps
SET status = $2
WHERE id = $1;
