-- Provisionamento assíncrono de domínios: retry com backoff + estado do cert.
ALTER TABLE domains
    ADD COLUMN retry_count  INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN last_error   TEXT        NOT NULL DEFAULT '',
    ADD COLUMN next_retry_at TIMESTAMPTZ;
