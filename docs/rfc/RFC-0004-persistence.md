# RFC-0004 — Persistência

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P2, P6, P7, P8, P10
- **Dependências:** —

---

## 1. Objetivo

Definir a estratégia de persistência por edição (SQLite / PostgreSQL / PostgreSQL HA), as
camadas de dados (state, event log, snapshot, secrets, audit) e o modelo de migrações e
concorrência.

## 2. Escopo

**Dentro:** estratégia por edição; camadas de dados; migrações; concorrência; backup/restore
de banco (estado); decisões de schema.
**Fora:** backup de volumes de app (RFC-0014); runtime OCI (RFC-0001).

## 3. Responsabilidades

- Prover repositórios (portas) para o domínio.
- Implementar SQLite (Community) e PostgreSQL (Business/Enterprise).
- Gerir event log + outbox.
- Gerir snapshots e compactação.
- Gerir secrets cifrados.
- Migrações transacionais + rollback por versão.

## 4. Arquitetura

```
Domínio → Repositórios (portas)
   │
   ├── sqlite/  (implementação)
   │     state.db (WAL)
   │     events.db (append-only)
   │     secrets (cifrado)
   │     audit.db (append-only)
   └── postgres/ (implementação)
         schema public
```

## 5. Fluxos

### 5.1 Escrita com outbox (SQLite)

```
1. begin tx
2. UPDATE state ...
3. INSERT INTO events(outbox) ...
4. commit  (um fsync)
5. publisher lê outbox → publica
6. delete outbox rows
```

### 5.2 Snapshot + compactação

```
1. a cada N eventos/T min → CREATE snapshot (projeção completa)
2. marca eventos <= snapshot como materializados
3. GC: remove eventos materializados antigos (política)
```

### 5.3 Migração

```
1. detectar versão (user_version / schema_migrations)
2. aplicar migrações pendentes (cada uma transacional)
3. registrar schema.migrated
4. rollback de versão → migração reversa correspondente
```

## 6. Interfaces

```go
type Repo interface { /* repositórios de domínio: Apps, Deployments, ... */ }
type EventStore interface {
    Append(ctx, aggregateType, aggregateID string, events []Event) ([]Sequence, error)
    Load(ctx, aggregateType, aggregateID string, from Sequence) ([]Event, error)
    Outbox(ctx) ([]OutboxEntry, error)
    Compact(ctx, policy) error
}
type SnapshotStore interface {
    Take(ctx, name string, data []byte) error
    Load(ctx, name string) ([]byte, error)
}
type SecretStore interface {
    Put(ctx, ref string, ciphertext []byte) error
    Get(ctx, ref string) ([]byte, error)
    Delete(ctx, ref string) error
}
```

## 7. Eventos

Emitidos: `schema.migrated`, `eventlog.compacted`, `snapshot.taken`.
Consumidos: (GC) — triggers internos.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Community 1 VPS | SQLite WAL, zero processo |
| Business multi-tenant | PG com concorrência real |
| Enterprise HA | Patroni + failover |
| Crash recovery | snapshot + replay |
| Update de versão | migrações transacionais + rollback |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| SQLite default | zero custo/setup | 1 escritor; limites de escala |
| PG por edição | escalabilidade | +serviço, +RAM |
| Outbox same-tx | consistência | tabela extra |
| JSON payloads | flexibilidade | sem FK em payload |

## 10. Decisões

- **D-001:** SQLite Community; PG Business; PG HA Enterprise.
- **D-002:** IDs UUIDv7; timestamps UTC.
- **D-003:** outbox na mesma transação do comando.
- **D-004:** `PRAGMA` WAL/NORMAL/auto_vacuum no SQLite.
- **D-005:** migrações versionadas, transacionais, com reversa.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| SQLite escrita concorrente | média | fila de escrita + batch |
| SQLite crescimento | baixa | auto_vacuum + compactação |
| PG HA complexidade | baixa (fase 5) | abstração pronta; começa tarde |
| Corrupção | baixa | WAL + backups + integrity_check |

## 12. Alternativas descartadas

- **Postgres em Community**: descartado (custo fixo desnecessário para o perfil).
- **Redis para estado**: descartado (estado efêmero; viola P7).
- **Kafka/NATS para fila/eventos**: descartado (processo externo; overkill em v1).
