-- name: ListServiceSourcesByRepository :many
SELECT service_sources.id, service_sources.service_id, service_sources.connection_id, scm_connections.organization_id,
       service_sources.repository_id, service_sources.repository_owner, service_sources.repository_name,
       service_sources.repository_full_name, service_sources.default_branch, service_sources.branch,
       service_sources.auto_deploy, service_sources.root_directory, service_sources.watch_paths,
       service_sources.ignore_paths, service_sources.watch_root_files, service_sources.created_at,
       service_sources.updated_at
FROM service_sources
JOIN scm_connections ON scm_connections.id = service_sources.connection_id
WHERE scm_connections.provider = $1
  AND scm_connections.installation_id = $2
  AND service_sources.repository_id = $3;

-- name: GetServiceSource :one
SELECT service_sources.id, service_sources.service_id, service_sources.connection_id, scm_connections.organization_id,
       service_sources.repository_id, service_sources.repository_owner, service_sources.repository_name,
       service_sources.repository_full_name, service_sources.default_branch, service_sources.branch,
       service_sources.auto_deploy, service_sources.root_directory, service_sources.watch_paths,
       service_sources.ignore_paths, service_sources.watch_root_files, service_sources.created_at,
       service_sources.updated_at
FROM service_sources
JOIN scm_connections ON scm_connections.id = service_sources.connection_id
JOIN apps ON apps.id = service_sources.service_id
WHERE service_sources.service_id = $1
  AND apps.org_id = $2
  AND scm_connections.organization_id = $2;

-- name: UpsertServiceSource :one
INSERT INTO service_sources (
    service_id, connection_id, repository_id, repository_owner, repository_name,
    repository_full_name, default_branch, branch, auto_deploy, root_directory,
    watch_paths, ignore_paths, watch_root_files
)
SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
WHERE EXISTS (SELECT 1 FROM apps WHERE apps.id = $1 AND apps.org_id = $14)
  AND EXISTS (SELECT 1 FROM scm_connections WHERE scm_connections.id = $2 AND scm_connections.organization_id = $14)
ON CONFLICT (service_id) DO UPDATE
SET connection_id = EXCLUDED.connection_id,
    repository_id = EXCLUDED.repository_id,
    repository_owner = EXCLUDED.repository_owner,
    repository_name = EXCLUDED.repository_name,
    repository_full_name = EXCLUDED.repository_full_name,
    default_branch = EXCLUDED.default_branch,
    branch = EXCLUDED.branch,
    auto_deploy = EXCLUDED.auto_deploy,
    root_directory = EXCLUDED.root_directory,
    watch_paths = EXCLUDED.watch_paths,
    ignore_paths = EXCLUDED.ignore_paths,
    watch_root_files = EXCLUDED.watch_root_files,
    updated_at = now()
RETURNING id, service_id, connection_id, repository_id, repository_owner, repository_name,
          repository_full_name, default_branch, branch, auto_deploy, root_directory,
          watch_paths, ignore_paths, watch_root_files, created_at, updated_at;

-- name: DeleteServiceSource :exec
DELETE FROM service_sources
WHERE service_id = $1
  AND EXISTS (SELECT 1 FROM apps WHERE apps.id = $1 AND apps.org_id = $2);

-- name: UpsertSCMConnection :one
INSERT INTO scm_connections (organization_id, provider, external_account_id, external_account_name, installation_id, status, metadata, credentials_enc)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (organization_id, provider, installation_id) DO UPDATE
SET external_account_id = EXCLUDED.external_account_id,
    external_account_name = EXCLUDED.external_account_name,
    status = EXCLUDED.status,
    metadata = EXCLUDED.metadata,
    credentials_enc = CASE WHEN EXCLUDED.credentials_enc <> '' THEN EXCLUDED.credentials_enc ELSE scm_connections.credentials_enc END,
    updated_at = now()
RETURNING id, organization_id, provider, external_account_id, external_account_name,
          installation_id, status, metadata, credentials_enc, created_at, updated_at;

-- name: ListSCMConnectionsByOrganization :many
SELECT id, organization_id, provider, external_account_id, external_account_name,
       installation_id, status, metadata, credentials_enc, created_at, updated_at
FROM scm_connections
WHERE organization_id = $1 AND provider = $2
ORDER BY created_at DESC;

-- name: GetSCMConnectionByInstallation :one
SELECT id, organization_id, provider, external_account_id, external_account_name,
       installation_id, status, metadata, credentials_enc, created_at, updated_at
FROM scm_connections
WHERE provider = $1 AND installation_id = $2;

-- name: GetSCMConnectionByID :one
SELECT id, organization_id, provider, external_account_id, external_account_name,
       installation_id, status, metadata, credentials_enc, created_at, updated_at
FROM scm_connections
WHERE id = $1;

-- name: DeleteSCMConnection :exec
DELETE FROM scm_connections
WHERE id = $1 AND organization_id = $2;

-- name: AttachSCMConnectionInstallation :one
UPDATE scm_connections
SET installation_id = $2, external_account_id = $3, external_account_name = $4,
    status = 'active', updated_at = now()
WHERE id = $1 AND installation_id = ''
RETURNING id, organization_id, provider, external_account_id, external_account_name,
          installation_id, status, metadata, credentials_enc, created_at, updated_at;

-- name: CreateSCMManifestState :exec
INSERT INTO scm_manifest_states (state, organization_id, user_id, return_url, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: ConsumeSCMManifestState :one
DELETE FROM scm_manifest_states
WHERE state = $1 AND expires_at > now()
RETURNING state, organization_id, user_id, return_url, expires_at, created_at;

-- name: ClaimSCMWebhookDelivery :one
INSERT INTO scm_webhook_deliveries (provider, delivery_id, event_type, installation_id, repository_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (provider, delivery_id) DO NOTHING
RETURNING id, provider, delivery_id, event_type, installation_id, repository_id,
          received_at, processed_at, status, error;

-- name: CompleteSCMWebhookDelivery :exec
UPDATE scm_webhook_deliveries
SET processed_at = now(), status = $2, error = $3
WHERE id = $1;
