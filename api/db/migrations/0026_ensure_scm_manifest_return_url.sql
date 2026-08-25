ALTER TABLE scm_manifest_states
    ADD COLUMN IF NOT EXISTS return_url TEXT NOT NULL DEFAULT '';
