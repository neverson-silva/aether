# RFC-0021 — PostgreSQL como banco oficial

- **Status:** Implementado
- **Data:** 2026-08-04
- **Dependências:** RFC-0005 (Event Bus)

## Objetivo

Substituir completamente SQLite por PostgreSQL 15+ como único banco da plataforma, com bootstrap
totalmente automático e qualidade de produção (concorrência, observabilidade, HA/replicação futura).

## Bootstrap automático

Na primeira inicialização a aplicação:

1. Conecta ao PostgreSQL (retry com backoff exponencial — `DATABASE_RETRY_ATTEMPTS`/`DATABASE_RETRY_DELAY`).
2. Valida versão mínima (15+ via `server_version_num`).
3. Cria o banco quando inexistente (`CREATE DATABASE` via conexão de manutenção).
4. Garante o schema configurado (`DATABASE_SCHEMA`, default `public`).
5. Executa migrations pendentes sob `pg_advisory_lock` (uma instância por vez; demais aguardam).
6. Seeds idempotentes (templates do marketplace; admin inicial via registro na UI).

A API só sobe depois do banco pronto.

## Driver e camada de acesso

- Driver: `github.com/jackc/pgx/v5` via `database/sql` (`pgx/v5/stdlib`) — mantém a arquitetura `*sql.DB`.
- `db.SQL`/`db.Tx`: wrappers que convertem placeholders `?` → `$N` automaticamente — toda a camada
  de repositórios existente permanece intacta.
- Pool configurável (`DATABASE_POOL_MIN`/`MAX`), timeouts de conexão/idle/statement.
- Migrations idempotentes, versionadas em `schema_migrations`, transacionais por versão.

## Configuração

| Variável | Default | Descrição |
|---|---|---|
| `DATABASE_URL` | — | URL completa (sobrepõe as demais) |
| `DATABASE_HOST` / `DATABASE_PORT` | 127.0.0.1 / 5432 | Endpoint |
| `DATABASE_NAME` / `DATABASE_USER` / `DATABASE_PASSWORD` | aether / postgres | Credenciais |
| `DATABASE_SSL_MODE` | prefer | disable/require/verify-full |
| `DATABASE_SCHEMA` | public | Schema (multi-tenant por schema) |
| `DATABASE_POOL_MIN` / `DATABASE_POOL_MAX` | 2 / 20 | Pool |
| `DATABASE_CONNECTION_TIMEOUT` / `DATABASE_IDLE_TIMEOUT` | 10 / 300 | Timeouts (s) |
| `DATABASE_STATEMENT_TIMEOUT` / `DATABASE_QUERY_TIMEOUT` | 0 | Timeouts (s) |
| `DATABASE_APPLICATION_NAME` | aether | Identificação no PG |
| `DATABASE_LOGGING` | false | Log de queries |
| `DATABASE_MIGRATE_ON_START` | true | Migra no boot |
| `DATABASE_SEED_ON_FIRST_START` | true | Seeds idempotentes |
| `DATABASE_RETRY_ATTEMPTS` / `DATABASE_RETRY_DELAY` | 10 / 2 | Retry com backoff |

## Backup/restore

Backup de estado agora é `pg_dump -Fc`; restore via `pg_restore --clean --if-exists` (mesmo banco).

## Compatibilidade

PostgreSQL 15/16/17+ — local, RDS, Cloud SQL, Neon, Supabase, Crunchy, Timescale. Sem recursos
depreciados; `gen_random_uuid()` (nativo 13+), `JSONB` preparado para dados semi-estruturados.

## Testes

Suite roda contra PostgreSQL de teste (schema isolado por teste, `DROP SCHEMA ... CASCADE` no cleanup).
Cobertura nova: bootstrap, migrations idempotentes, **migração concorrente com advisory lock**,
retry de conexão, healthcheck com pool/latência/versão.
