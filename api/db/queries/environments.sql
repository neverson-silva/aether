-- name: CreateEnvironment :one
INSERT INTO environments (project_id, name, slug, description, color, is_default)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, project_id, name, slug, description, color, is_default, created_at, updated_at;

-- name: GetEnvironment :one
SELECT id, project_id, name, slug, description, color, is_default, created_at, updated_at
FROM environments
WHERE id = $1 AND project_id = $2;

-- name: ListEnvironments :many
SELECT id, project_id, name, slug, description, color, is_default, created_at, updated_at
FROM environments
WHERE project_id = $1
ORDER BY is_default DESC, created_at ASC;

-- name: DefaultEnvironment :one
SELECT id, project_id, name, slug, description, color, is_default, created_at, updated_at
FROM environments
WHERE project_id = $1
ORDER BY is_default DESC, created_at ASC
LIMIT 1;

-- name: UpdateEnvironment :one
UPDATE environments
SET name = $3, slug = $4, description = $5, color = $6, updated_at = now()
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, name, slug, description, color, is_default, created_at, updated_at;

-- name: DeleteEnvironment :exec
DELETE FROM environments
WHERE id = $1 AND project_id = $2 AND is_default = FALSE;
