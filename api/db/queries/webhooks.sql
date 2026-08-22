-- name: CreateOutWebhook :one
INSERT INTO out_webhooks (org_id, name, url, secret_enc, events, enabled)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, org_id, name, url, secret_enc, events, enabled, created_at;

-- name: GetOutWebhook :one
SELECT id, org_id, name, url, secret_enc, events, enabled, created_at
FROM out_webhooks
WHERE id = $1;

-- name: ListOutWebhooksByOrg :many
SELECT id, org_id, name, url, secret_enc, events, enabled, created_at
FROM out_webhooks
WHERE org_id = $1
ORDER BY name;

-- name: ListEnabledWebhooksByEvent :many
SELECT id, org_id, name, url, secret_enc, events, enabled, created_at
FROM out_webhooks
WHERE enabled = TRUE AND $1::text = ANY(events)
ORDER BY name;

-- name: DeleteOutWebhook :exec
DELETE FROM out_webhooks
WHERE id = $1 AND org_id = $2;
