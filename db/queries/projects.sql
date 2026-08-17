-- name: CreateProject :one
INSERT INTO projects (org_id, name, slug, description, color)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, org_id, name, slug, description, color, created_at, updated_at;

-- name: GetProject :one
SELECT id, org_id, name, slug, description, color, created_at, updated_at
FROM projects
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL;

-- name: ListProjects :many
SELECT id, org_id, name, slug, description, color, created_at, updated_at
FROM projects
WHERE org_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateProject :one
UPDATE projects
SET name = $3, slug = $4, description = $5, color = $6, updated_at = now()
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL
RETURNING id, org_id, name, slug, description, color, created_at, updated_at;

-- name: DeleteProject :exec
UPDATE projects
SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND org_id = $2 AND deleted_at IS NULL;
