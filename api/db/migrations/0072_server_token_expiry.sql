UPDATE server_tokens
SET expires_at = created_at + INTERVAL '24 hours'
WHERE expires_at IS NULL;

ALTER TABLE server_tokens
    ALTER COLUMN expires_at SET DEFAULT (now() + INTERVAL '24 hours'),
    ALTER COLUMN expires_at SET NOT NULL;
