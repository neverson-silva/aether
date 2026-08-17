-- name: CreateOrg :one
INSERT INTO orgs (name, slug, owner_user_id)
VALUES ($1, $2, $3)
RETURNING id, name, slug, avatar, color, owner_user_id, created_at;

-- name: GetOrgByID :one
SELECT id, name, slug, avatar, color, owner_user_id, created_at
FROM orgs
WHERE id = $1;

-- name: ListOrgsForUser :many
SELECT o.id, o.name, o.slug, o.avatar, o.color, o.owner_user_id, o.created_at, m.role AS member_role
FROM members m
JOIN orgs o ON o.id = m.org_id
WHERE m.user_id = $1
ORDER BY o.name;

-- name: CreateMember :exec
INSERT INTO members (org_id, user_id, role)
VALUES ($1, $2, $3);

-- name: GetMember :one
SELECT org_id, user_id, role, created_at
FROM members
WHERE org_id = $1 AND user_id = $2;

-- name: UpdateMemberRole :exec
UPDATE members
SET role = $3
WHERE org_id = $1 AND user_id = $2;

-- name: DeleteMember :exec
DELETE FROM members
WHERE org_id = $1 AND user_id = $2;
