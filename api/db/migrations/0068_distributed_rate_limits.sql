CREATE TABLE IF NOT EXISTS rate_limit_buckets (
    bucket_key TEXT PRIMARY KEY,
    window_started TIMESTAMPTZ NOT NULL,
    request_count INTEGER NOT NULL,
    window_seconds INTEGER NOT NULL,
    max_requests INTEGER NOT NULL
);
