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
INSERT INTO s3_destinations (org_id, name, type, endpoint, bucket, region, account_id, access_key_enc, secret_key_enc,
                             google_client_id, google_client_secret_enc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, org_id, name, type, endpoint, bucket, region, account_id, access_key_enc, secret_key_enc,
          oauth_status, oauth_email, access_token_enc, refresh_token_enc,
          google_client_id, google_client_secret_enc, created_at, updated_at;

-- name: ListS3Destinations :many
SELECT id, org_id, name, type, endpoint, bucket, region, account_id, access_key_enc, secret_key_enc,
       oauth_status, oauth_email, access_token_enc, refresh_token_enc,
       google_client_id, google_client_secret_enc, created_at, updated_at
FROM s3_destinations
WHERE org_id = $1
ORDER BY name;

-- name: GetS3Destination :one
SELECT id, org_id, name, type, endpoint, bucket, region, account_id, access_key_enc, secret_key_enc,
       oauth_status, oauth_email, access_token_enc, refresh_token_enc,
       google_client_id, google_client_secret_enc, created_at, updated_at
FROM s3_destinations
WHERE id = $1 AND org_id = $2;

-- name: UpdateS3Destination :one
UPDATE s3_destinations
SET name = $3, type = $4, endpoint = $5, bucket = $6, region = $7, account_id = $8,
    access_key_enc = $9, secret_key_enc = $10,
    google_client_id = $11, google_client_secret_enc = $12, updated_at = now()
WHERE id = $1 AND org_id = $2
RETURNING id, org_id, name, type, endpoint, bucket, region, account_id, access_key_enc, secret_key_enc,
          oauth_status, oauth_email, access_token_enc, refresh_token_enc,
          google_client_id, google_client_secret_enc, created_at, updated_at;

-- name: UpdateS3DestinationOAuth :exec
UPDATE s3_destinations
SET oauth_status = $3, oauth_email = $4, access_token_enc = $5, refresh_token_enc = $6, updated_at = now()
WHERE id = $1 AND org_id = $2;

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

