-- Suporte a templates com compose YAML pronto (multi-serviço).
-- Permite executar o docker-compose oficial de aplicações complexas (ex: AFFiNE)
-- em vez de gerar compose a partir de `definition`.

ALTER TABLE templates ADD COLUMN compose_yaml TEXT NOT NULL DEFAULT '';

-- AFFiNE: workspace open source estilo Notion (docs + whiteboard + databases).
-- Compose adaptado do oficial (imagem única ghcr.io/toeverything/affine:stable,
-- servidor + job de migração + Postgres + Redis embutidos).
-- Removido container_name e name: para evitar colisão entre instâncias; dados em volumes nomeados.
INSERT INTO templates (name, description, category, icon, definition, compose_yaml, featured, verified, tags)
VALUES (
  'AFFiNE',
  'Open-source Notion & Miro alternative — docs, whiteboards and databases in a local-first workspace',
  'cms',
  'affine',
  '{"services":[{"name":"affine","image":"ghcr.io/toeverything/affine:stable","port":3010,"versions":["ghcr.io/toeverything/affine:stable"]}]}',
  'services:
  affine:
    image: ghcr.io/toeverything/affine:stable
    ports:
      - "3010:3010"
    depends_on:
      redis:
        condition: service_healthy
      postgres:
        condition: service_healthy
      affine_migration:
        condition: service_completed_successfully
    volumes:
      - affine-storage:/root/.affine/storage
    environment:
      - REDIS_SERVER_HOST=redis
      - DATABASE_URL=postgresql://affine@postgres:5432/affine
      - AFFINE_INDEXER_ENABLED=false
    restart: unless-stopped

  affine_migration:
    image: ghcr.io/toeverything/affine:stable
    command: ["sh", "-c", "node ./scripts/self-host-predeploy.js"]
    volumes:
      - affine-storage:/root/.affine/storage
    environment:
      - REDIS_SERVER_HOST=redis
      - DATABASE_URL=postgresql://affine@postgres:5432/affine
      - AFFINE_INDEXER_ENABLED=false
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  postgres:
    image: pgvector/pgvector:pg16
    volumes:
      - affine-pg:/var/lib/postgresql/data
    environment:
      POSTGRES_USER: affine
      POSTGRES_DB: affine
      POSTGRES_INITDB_ARGS: "--data-checksums"
      POSTGRES_HOST_AUTH_METHOD: trust
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "affine", "-d", "affine"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  affine-storage:
  affine-pg:
',
  'true', true, ARRAY['affine','notion','docs','whiteboard','notes']::text[]
);
