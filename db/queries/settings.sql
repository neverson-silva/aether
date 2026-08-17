-- name: GetBranding :one
SELECT org_id, name, logo_url, primary_color, accent_color, dark_mode, updated_at
FROM branding
WHERE org_id = $1;

-- name: UpsertBranding :one
INSERT INTO branding (org_id, name, logo_url, primary_color, accent_color, dark_mode)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (org_id) DO UPDATE SET
    name = EXCLUDED.name,
    logo_url = EXCLUDED.logo_url,
    primary_color = EXCLUDED.primary_color,
    accent_color = EXCLUDED.accent_color,
    dark_mode = EXCLUDED.dark_mode,
    updated_at = now()
RETURNING org_id, name, logo_url, primary_color, accent_color, dark_mode, updated_at;

-- name: CreateS3Destination :one
INSERT INTO s3_destinations (org_id, name, endpoint, bucket, region, access_key_enc, secret_key_enc)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, name, endpoint, bucket, region, access_key_enc, secret_key_enc, created_at;

-- name: ListS3Destinations :many
SELECT id, org_id, name, endpoint, bucket, region, access_key_enc, secret_key_enc, created_at
FROM s3_destinations
WHERE org_id = $1
ORDER BY name;

-- name: DeleteS3Destination :exec
DELETE FROM s3_destinations
WHERE id = $1 AND org_id = $2;

-- name: CreateOIDCProvider :one
INSERT INTO oidc_providers (org_id, name, issuer, client_id, client_secret_enc, scopes, enabled)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, name, issuer, client_id, client_secret_enc, scopes, enabled, created_at;

-- name: ListOIDCProviders :many
SELECT id, org_id, name, issuer, client_id, client_secret_enc, scopes, enabled, created_at
FROM oidc_providers
WHERE org_id = $1
ORDER BY name;

-- name: ListEnabledOIDCProviders :many
SELECT id, org_id, name, issuer, client_id, client_secret_enc, scopes, enabled, created_at
FROM oidc_providers
WHERE enabled = true
ORDER BY name;

-- name: CountEnabledOIDCProviders :one
SELECT count(*)::int
FROM oidc_providers
WHERE enabled = true;

-- name: GetOIDCProvider :one
SELECT id, org_id, name, issuer, client_id, client_secret_enc, scopes, enabled, created_at
FROM oidc_providers
WHERE id = $1;

-- name: DeleteOIDCProvider :exec
DELETE FROM oidc_providers
WHERE id = $1 AND org_id = $2;

-- name: GetS3Destination :one
SELECT id, org_id, name, endpoint, bucket, region, access_key_enc, secret_key_enc, created_at
FROM s3_destinations
WHERE id = $1 AND org_id = $2;
