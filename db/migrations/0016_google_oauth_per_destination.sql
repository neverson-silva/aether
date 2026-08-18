-- Google OAuth client credentials move to the destination row: every Google
-- Drive destination is an independent connection with its own client config
-- and account. The org-level table created in 0015 is empty (no data was ever
-- saved) and is removed in favor of the per-destination columns.

ALTER TABLE s3_destinations
    ADD COLUMN google_client_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN google_client_secret_enc TEXT NOT NULL DEFAULT '';

DROP TABLE IF EXISTS google_oauth_configs;
