-- S3 destinations: explicit provider type + Google Drive OAuth support.
-- Existing destinations are backfilled by inferring the provider from the
-- endpoint (AWS/R2 signatures); everything else stays custom-s3. No data is
-- dropped or rewritten.

ALTER TABLE s3_destinations
    ADD COLUMN type TEXT NOT NULL DEFAULT 'custom-s3',
    ADD COLUMN account_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN oauth_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN oauth_email TEXT NOT NULL DEFAULT '',
    ADD COLUMN access_token_enc TEXT NOT NULL DEFAULT '',
    ADD COLUMN refresh_token_enc TEXT NOT NULL DEFAULT '',
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

UPDATE s3_destinations
SET type = CASE
    WHEN endpoint LIKE '%amazonaws.com%' THEN 'aws'
    WHEN endpoint LIKE '%r2.cloudflarestorage.com%' THEN 'cloudflare-r2'
    ELSE 'custom-s3'
END;
