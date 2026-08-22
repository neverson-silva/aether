-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, global_role)
VALUES ($1, $2, $3, $4)
RETURNING id, email, name, password_hash, global_role, totp_enabled, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, global_role, totp_enabled, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, password_hash, global_role, totp_enabled, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsersByOrg :many
SELECT u.id, u.email, u.name, u.global_role, u.totp_enabled, u.created_at, u.updated_at, m.role AS member_role
FROM members m
JOIN users u ON u.id = m.user_id
WHERE m.org_id = $1
ORDER BY u.name;

-- name: SetTOTP :exec
UPDATE users
SET totp_secret = $2, totp_enabled = TRUE
WHERE id = $1;

-- name: DisableTOTP :exec
UPDATE users
SET totp_secret = NULL, totp_enabled = FALSE
WHERE id = $1;

-- name: GetUserWithSecret :one
SELECT id, email, name, password_hash, global_role, totp_enabled, totp_secret, created_at, updated_at
FROM users
WHERE id = $1;

-- name: HasUsers :one
SELECT EXISTS(SELECT 1 FROM users LIMIT 1);
