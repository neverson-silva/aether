-- name: CreateDomain :one
INSERT INTO domains (app_id, server_id, service_type, host, https, path, internal_path, strip_path, container_port, status, cert_status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id, app_id, server_id, service_type, host, https, path, internal_path, strip_path, container_port, status, cert_status, retry_count, last_error, next_retry_at, created_at, updated_at;

-- name: ListDomains :many
SELECT id, app_id, server_id, service_type, host, https, path, internal_path, strip_path, container_port, status, cert_status, retry_count, last_error, next_retry_at, created_at, updated_at
FROM domains
WHERE app_id = $1
ORDER BY host;

-- name: GetDomainByHost :one
SELECT id, app_id, server_id, service_type, host, https, path, internal_path, strip_path, container_port, status, cert_status, retry_count, last_error, next_retry_at, created_at, updated_at
FROM domains
WHERE app_id = $1 AND host = $2;

-- name: GetDomainByID :one
SELECT id, app_id, server_id, service_type, host, https, path, internal_path, strip_path, container_port, status, cert_status, retry_count, last_error, next_retry_at, created_at, updated_at
FROM domains
WHERE id = $1;

-- name: UpdateDomainStatus :exec
UPDATE domains
SET status = $3,
    cert_status = $4,
    updated_at = now()
WHERE id = $1 AND app_id = $2;

-- name: UpdateDomainFields :exec
UPDATE domains
SET host = $3,
    https = $4,
    path = $5,
    internal_path = $6,
    strip_path = $7,
    container_port = $8,
    status = 'PROVISIONING',
    retry_count = 0,
    last_error = '',
    next_retry_at = NULL,
    updated_at = now()
WHERE id = $1 AND app_id = $2;

-- name: UpdateDomainProvision :exec
UPDATE domains
SET status = $3,
    cert_status = $4,
    last_error = $5,
    next_retry_at = $6,
    retry_count = $7,
    updated_at = now()
WHERE id = $1 AND app_id = $2;

-- name: ListProvisioningDomains :many
SELECT id, app_id, server_id, service_type, host, https, path, internal_path, strip_path, container_port, status, cert_status, retry_count, last_error, next_retry_at, created_at, updated_at
FROM domains
WHERE status IN ('PENDING', 'PROVISIONING', 'ERROR')
  AND retry_count < $1
  AND (next_retry_at IS NULL OR next_retry_at <= $2)
ORDER BY created_at;

-- name: DeleteDomain :exec
DELETE FROM domains
WHERE id = $1 AND app_id = $2;

-- name: CreatePreview :one
INSERT INTO previews (app_id, branch, deployment_id, domain)
VALUES ($1, $2, $3, $4)
RETURNING id, app_id, branch, deployment_id, container_id, domain, status, created_at;

-- name: ListPreviews :many
SELECT id, app_id, branch, deployment_id, container_id, domain, status, created_at
FROM previews
WHERE app_id = $1
ORDER BY created_at DESC;

-- name: GetPreview :one
SELECT id, app_id, branch, deployment_id, container_id, domain, status, created_at
FROM previews
WHERE id = $1 AND app_id = $2;

-- name: GetPreviewByID :one
SELECT id, app_id, branch, deployment_id, container_id, domain, status, created_at
FROM previews
WHERE id = $1;

-- name: UpdatePreviewResult :exec
UPDATE previews
SET deployment_id = $3,
    container_id = $4,
    status = $5
WHERE id = $1 AND app_id = $2;

-- name: DeletePreview :exec
DELETE FROM previews
WHERE id = $1 AND app_id = $2;

-- name: ListCertificatesByOrg :many
SELECT d.id, d.app_id, a.name AS app_name, d.host, d.https, d.cert_status, d.created_at
FROM domains d
JOIN apps a ON a.id = d.app_id
JOIN projects p ON p.id = a.project_id
WHERE p.org_id = $1
ORDER BY d.host;
