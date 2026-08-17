# 10 — Persistência

> **Status:** Estratégia de dados.
> **RFC relacionada:** [`RFC-0004`](rfc/RFC-0004-persistence.md)
> **Princípios atendidos:** P2, P6, P7, P8, P10.

---

## 1. Propósito

Definir a estratégia completa de persistência, que varia por edição do produto:

| Edição | Persistência | Modelo |
|--------|--------------|--------|
| **Community** | SQLite (WAL) | um arquivo, zero processo |
| **Business** | PostgreSQL | serviço gerenciado (via runtime OCI), multi-tenant maior |
| **Enterprise** | PostgreSQL HA (Patroni) | alta disponibilidade, failover automático |

**Princípio norteador:** o custo de persistência deve ser **proporcional ao valor** — um
self-hoster de 1 VPS não deve pagar o custo de um cluster Postgres. A mesma arquitetura de dados
(portas de repositório) serve os três backends.

## 2. Justificativa da estratégia por edição

### 2.1 SQLite (Community)

**Vantagens:**
1. **Zero processo, zero setup** — não existe "serviço de banco" para manter (P2, P4).
2. **Zero rede** — latência mínima; sem auth, sem portas expostas.
3. **SSD mínimo** — arquivo + WAL; fácil backup (`VACUUM INTO`).
4. **Custo de manutenção nulo** — sem tuning de servidor.
5. **Confiabilidade** — WAL + `PRAGMA` de durabilidade; recuperação trivial.

**Limitações:**
- Escrita concorrente: 1 escritor por vez. Mitigação: fila de escrita no `core`, batch commits,
  transações curtas, WAL para leitura concorrente.
- Escala: ótimo até dezenas de milhares de recursos; não é banco de analytics.
- `fsync` por transação: mitigado com `synchronous=NORMAL` + WAL (crash-safe no nível de
  transação commitada).

**Decisões de uso:**
- `PRAGMA journal_mode=WAL`
- `PRAGMA synchronous=NORMAL`
- `PRAGMA cache_size` limitado (poucos MB)
- `PRAGMA auto_vacuum=INCREMENTAL` + `VACUUM` agendado por GC
- Backup: `VACUUM INTO` (snapshot consistente) ou backup contínuo com hot WAL

### 2.2 PostgreSQL (Business)

**Por que quando:** multi-tenant com muitos usuários e escrita concorrente real, extensões
(PostGIS etc.), replicação, e quando o time já opera PG. É o padrão da indústria para o papel
"banco de verdade".

**Como Aether evita o custo:** PG roda como **workload gerenciado pelo próprio runtime OCI**
(container Podman/unit systemd) — não é "container de suporte" eterno; é opcional (só na edição
Business), e o usuário pode conectar um PG externo existente.

**Trade-offs:** +1 serviço (RAM ~150–300 MB, SSD ~1 GB); justificado na edição certa.

### 2.3 PostgreSQL HA (Enterprise)

**Modelo:** Patroni (leader + replicas + etcd/consul para eleição) — mas **sem container etcd
fixo**: Aether usa o event log/distributed lock do próprio core para a eleição ou um etcd
gerenciado. Failover automático; leitura em réplicas; zero-downtime para a plataforma.

## 3. Camadas de dados (aplicáveis a qualquer backend)

```
+---------------------------+
| Repositórios (portas)     |  ← domínio usa apenas estas
|  AppRepo, DeployRepo, ... |
+---------------------------+
| Implementações por backend|
|  sqlite/*, postgres/*     |
+---------------------------+
|  State store (CRUD)        |
|  Event log (append-only)   |
|  Snapshot store (projeção) |
|  Secrets (criptografado)   |
|  Audit (append-only)       |
+---------------------------+
```

### 3.1 State store

Entidades de domínio (apps, deployments, orgs, servers, etc.) como tabelas relacionais
tradicionais. Migrações gerenciadas por **ferramenta de migração embutida** (idempotente,
transacional).

### 3.2 Event log (append-only)

Toda mudança relevante vira evento. Tabela `events(id, aggregate_type, aggregate_id,
sequence, type, payload_json, timestamp)` com:
- Índice por (aggregate_type, aggregate_id, sequence) — replay por agregado.
- Retenção/compactação por política (GC).

### 3.3 Snapshot store

Projeções materializadas para leitura rápida (ex.: dashboard de deployments). Snapshot criado
a cada N eventos ou T minutos; recovery = último snapshot + replay.

### 3.4 Secrets

- Tabela `secrets(id, ref, ciphertext_blob, kdf_params)` — valor nunca em claro.
- Criptografia AES-256-GCM; chave de dados (DEK) criptografada por chave mestra (KEK).
- Ver [`16-seguranca.md`](16-seguranca.md).

### 3.5 Audit

- Tabela append-only `audit_events(user, action, resource, ip, ts, metadata)`.
- Imutável na prática (sem UPDATE/DELETE na API).

## 4. Migrações de schema

- Migrações **versionadas, transacionais**; cada migração roda em transação única.
- Em SQLite: `PRAGMA user_version` (ou tabela `schema_migrations`).
- Atualizações de versão = aplicar migrations pendentes de forma idempotente + event log de
  `schema.migrated`.
- Rollback de versão = migração reversa explícita (mantida por versão).

## 5. Concorrência e consistência

- **Aggregate-level locking**: uma transação por aggregate (evita deadlocks).
- **Idempotência**: todos os handlers de evento são idempotentes (upsert por aggregate+sequence).
- **Outbox pattern**: comando + eventos são persistidos na **mesma transação**; um dispatcher
  publica eventos após commit (garante at-least-once + idempotência de consumo).

## 6. Backup e restore de dados

- SQLite: `VACUUM INTO '<path>'` para snapshot; restauração = copiar arquivo + WAL replay.
- PostgreSQL: `pg_dump`/`pg_basebackup`.
- Event log + secrets + certs incluídos no backup de estado.
- Detalhes: RFC-0014.

## 7. Decisões de design de schema (principais)

| Decisão | Justificativa |
|---------|---------------|
| IDs UUIDv7 | ordenáveis por tempo, sem seq races, globais |
| Timestamps UTC com timezone | consistência |
| JSON para payloads de evento/config | flexibilidade sem JOINs pesados |
| Chaves FK explícitas | integridade |
| Índices mínimos (perfis de leitura conhecidos) | menos SSD em índice |
| WAL / auto_vacuum no SQLite | escrita + contenção de SSD |

## 8. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| SQLite contenção em escrita | fila de escrita + batch + transações curtas |
| SQLite crescimento infinito | auto_vacuum + VACUUM agendado + arquivamento de event log |
| Corrupção (falha de disco) | WAL, backups frequentes, verificação de integridade (`PRAGMA integrity_check`) |
| PG HA complexidade | começa em fase 5; abstração de repositório já preparada |
| Migrações quebradas | teste de migração em CI (banco sintético), versionamento rigoroso |

## 9. Decisões

- **Decisão:** SQLite padrão em Community (zero custo), PG para Business, PG HA para Enterprise.
- **Decisão:** event log e outbox na mesma transação do comando.
- **Decisão:** migrações versionadas e transacionais, com rollback por versão.
- **Decisão:** secrets criptografados por padrão, em qualquer backend.

## 10. Referências

- RFC-0004 (Persistence).
- Event Bus: [`12-event-bus.md`](12-event-bus.md).
- Backup: RFC-0014.
