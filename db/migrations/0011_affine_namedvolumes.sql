-- AFFiNE: usa named volumes (sem bind-mount com path da VM). Config seedada via
-- volume nomeado compartilhado aether-affine-config.
UPDATE templates
SET compose_yaml = 'services:
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
      - aether-affine-config:/root/.affine/config
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
      - aether-affine-config:/root/.affine/config
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
    restart: unless-stopped

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
    restart: unless-stopped

volumes:
  affine-storage:
  affine-pg:
'
WHERE name = 'AFFiNE';
