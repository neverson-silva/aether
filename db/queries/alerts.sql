-- name: ListAlertRules :many
SELECT id, org_id, name, metric, threshold, window_s, severity, enabled, target_app, created_at
FROM alert_rules
WHERE org_id = $1
ORDER BY name;

-- name: GetAlertRule :one
SELECT id, org_id, name, metric, threshold, window_s, severity, enabled, target_app, created_at
FROM alert_rules
WHERE id = $1;

-- name: CreateAlertRule :one
INSERT INTO alert_rules (org_id, name, metric, threshold, window_s, severity, target_app)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, org_id, name, metric, threshold, window_s, severity, enabled, target_app, created_at;

-- name: SetAlertRuleEnabled :exec
UPDATE alert_rules
SET enabled = $2
WHERE id = $1;

-- name: DeleteAlertRule :exec
DELETE FROM alert_rules
WHERE id = $1 AND org_id = $2;

-- name: CreateAlertEvent :one
INSERT INTO alert_events (org_id, rule_id, app_id, app_name, severity, message, value, threshold, metric)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, org_id, rule_id, app_id, app_name, severity, message, value, threshold, metric, created_at, resolved_at;

-- name: ListAlertEventsByOrg :many
SELECT id, org_id, rule_id, app_id, app_name, severity, message, value, threshold, metric, created_at, resolved_at
FROM alert_events
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ResolveAlertEvent :exec
UPDATE alert_events
SET resolved_at = now()
WHERE id = $1 AND resolved_at IS NULL;

-- name: CreateNotification :one
INSERT INTO notifications (org_id, type, message, payload)
VALUES ($1, $2, $3, $4)
RETURNING id, org_id, type, message, payload, read, created_at;

-- name: ListNotifications :many
SELECT id, org_id, type, message, payload, read, created_at
FROM notifications
WHERE org_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: CountUnreadNotifications :one
SELECT count(*)
FROM notifications
WHERE org_id = $1 AND read = FALSE;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET read = TRUE
WHERE id = $1 AND org_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET read = TRUE
WHERE org_id = $1 AND read = FALSE;

-- name: CreateChannel :one
INSERT INTO notification_channels (org_id, name, type, config_enc)
VALUES ($1, $2, $3, $4)
RETURNING id, org_id, name, type, config_enc, enabled, created_at;

-- name: ListChannelsByOrg :many
SELECT id, org_id, name, type, config_enc, enabled, created_at
FROM notification_channels
WHERE org_id = $1
ORDER BY name;

-- name: DeleteChannel :exec
DELETE FROM notification_channels
WHERE id = $1 AND org_id = $2;
