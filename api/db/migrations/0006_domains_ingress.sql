-- Ingress por Server (Traefik) + Domain como entidade de roteamento.
ALTER TABLE domains
    ADD COLUMN server_id      UUID REFERENCES servers(id) ON DELETE SET NULL,
    ADD COLUMN container_port INTEGER  NOT NULL DEFAULT 80,
    ADD COLUMN path            TEXT     NOT NULL DEFAULT '/',
    ADD COLUMN internal_path   TEXT     NOT NULL DEFAULT '/',
    ADD COLUMN strip_path      BOOLEAN  NOT NULL DEFAULT FALSE,
    ADD COLUMN status          TEXT     NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING','PROVISIONING','ACTIVE','ERROR','REMOVING','REMOVED')),
    ADD COLUMN updated_at      TIMESTAMPTZ NOT NULL DEFAULT now();
