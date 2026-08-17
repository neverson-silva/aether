# RFC-0013 — Databases Gerenciadas

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P2, P6
- **Dependências:** RFC-0001 (Execution Engine)

---

## 1. Objetivo

Definir o provisionamento de **databases gerenciadas** (PostgreSQL, MySQL/MariaDB, Redis,
MongoDB, etc.) como workloads OCI gerenciados pela plataforma, com backup integrado.

## 2. Escopo

**Dentro:** catálogo de DBs, provisionamento via Execution Engine, users/databases, exposição
a apps, backup/restore por dump, quotas.
**Fora:** tuning fino de cada engine (config avançada fica exposta via env/conf, mas sem
"cluster magic" em v1).

## 3. Responsabilidades

- Provisionar um banco como serviço (unit systemd + volume).
- Criar database + usuário com senha (secrets cifrados).
- Expor para apps (rede privada / DNS interno).
- Agendar backups (dump específico) — ver RFC-0014.
- Retenção de backups por política.
- Restore.

## 4. Arquitetura

```
Databases (domínio)
   DBEngine (porta)
    ├── PostgreSQL
    ├── MySQL/MariaDB
    ├── Redis
    └── MongoDB (v2+)
        │
  provisiona via Execution Engine (ContainerSpec + VolumeSpec)
```

## 5. Fluxos

### 5.1 Provisionar

```
1. criar Database(engine, version, resources, volume)
2. ExecutionEngine.Run (imagem oficial, rootless, limits)
3. inicia engine; espera ready (health)
4. cria database + user; senha gerada → secret
5. expõe no DNS interno (app.<org>.<host>.aether.internal)
6. database.created + secret.cert ref
```

### 5.2 Backup

```
1. policy (horário, retenção) → scheduled_event
2. dump (pg_dump/mysqldump/redis-cli save) → compress (zstd)
3. upload para blob store (local/S3)
4. backup.finished | backup.failed
```

## 6. Interfaces

```go
type DBEngine interface {
    Name() string
    Versions() []string
    InitContainerSpec(req DatabaseRequest) (ContainerSpec, error)
    CreateDatabase(ctx, handle, req) (DBHandle, error)
    Dump(ctx, handle, dest io.Writer) error
    Restore(ctx, handle, src io.Reader) error
    Health(ctx, handle) error
}
```

## 7. Eventos

Emitidos: `database.created`, `database.deleted`, `database.ready`, `database.failed`,
`database.backup_scheduled`, `database.backup_finished`, `database.restored`.
Consumidos: `backup.tick` (do scheduler), `app.attached`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Adicionar Postgres a um app | banco pronto + creds secret |
| Backup diário | dump + retenção |
| Restore | banco no estado do backup |
| App migra de plataforma | mesmo formato de DSN |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Engine via runtime OCI | sem imagem de suporte | atualização de imagem é do usuário |
| Dump nativo | compatível | dump pode ser grande |
| Secrets para creds | segurança | gestão de rotação |

## 10. Decisões

- **D-001:** catálogo v1: PostgreSQL, MariaDB/MySQL, Redis. MongoDB v2+.
- **D-002:** creds via secret store (cifrado); senha gerada.
- **D-003:** backup por dump nativo + zstd + blob store.
- **D-004:** exposição via DNS interno da plataforma (não publicar porta por default).

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Imagem oficial muda API | baixa | pin de versão |
| Dump grande | baixa | compressão + streaming |
| DB parado após crash | baixa | systemd restart + health |

## 12. Alternativas descartadas

- **DB como container de suporte da plataforma**: descartado (P4; são workloads do usuário).
- **Multi-engine "cluster" em v1**: descartado (complexidade; fase enterprise).
- **Backup via filesystem copy**: descartado (inconsistente; dump é correto).
