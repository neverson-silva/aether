-- name: GetOrg :one
SELECT id, name, slug, avatar, color, description, owner_user_id, created_at
FROM orgs
WHERE id = $1;

-- name: UpdateOrg :one
UPDATE orgs
SET name = $2, description = $3, avatar = $4, color = $5
WHERE id = $1
RETURNING id, name, slug, avatar, color, description, owner_user_id, created_at;

-- name: DeleteOrg :exec
DELETE FROM orgs
WHERE id = $1 AND owner_user_id = $2;

-- name: GetOrgRole :one
SELECT m.role
FROM members m
WHERE m.org_id = $1 AND m.user_id = $2;

-- name: SetProjectAssignment :exec
INSERT INTO project_assignments (org_id, user_id, project_id)
VALUES ($1, $2, $3)
ON CONFLICT (org_id, user_id, project_id) DO NOTHING;

-- name: RemoveProjectAssignment :exec
DELETE FROM project_assignments
WHERE org_id = $1 AND user_id = $2 AND project_id = $3;

-- name: ListProjectAssignments :many
SELECT id, org_id, user_id, project_id, created_at
FROM project_assignments
WHERE org_id = $1
ORDER BY created_at DESC;

-- name: ListOrgMembers :many
SELECT m.org_id, m.user_id, m.role, m.created_at, u.email, u.name
FROM members m
JOIN users u ON u.id = m.user_id
WHERE m.org_id = $1
ORDER BY u.name;
