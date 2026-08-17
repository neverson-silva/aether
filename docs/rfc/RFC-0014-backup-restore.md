# RFC-0014 — Backup e Restore

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P8, P2, P6
- **Dependências:** RFC-0004 (Persistência)

---

## 1. Objetivo

Definir backup e restore do **estado da plataforma** e dos **dados das aplicações** (volumes,
bancos), com agendamento, retenção e destinos remotos.

## 2. Escopo

**Dentro:** backup de estado (SQLite VACUUM INTO / pg_dump), backup de volumes de app,
dump de databases, compressão, blob store (local/S3/SSH), agendamento por eventos, retenção,
restore (validação e reconciliação).
**Fora:** replica síncrona (é HA, fase 5); backup de terceiros (são destinos).

## 3. Responsabilidades

- Backup agendado de estado e de dados.
- Restore íntegro e validado.
- Retenção (default 7) com GC.
- Destinos: local (diretório), S3-compatível (plugin), SSH.
- Janelas de baixa atividade (não conflitar com deploy).

## 4. Arquitetura

```
Backups (domínio)
  BackupScheduler (eventos + timer)
   ├── StateBackup (SQLite VACUUM INTO / pg_dump)
   ├── VolumeBackup (tar incremental / snapshot)
   └── DbBackup (dump engine-specific — RFC-0013)
        │
   Compressor (zstd) → ObjectStore (blob)
        │
   Retention/GC → poda
```

## 5. Fluxos

### 5.1 Backup agendado

```
1. policy configurada → scheduled_event
2. trigger → backup.started
3. coleta estado + dados (em janela de baixa atividade)
4. comprime → upload (local/S3/SSH)
5. registra backup.finished (manifesto: lista de blobs, versões)
6. GC retenção
```

### 5.2 Restore

```
1. escolher backup (manifesto)
2. validate (checksum, versões)
3. para estado: restaurar SQLite/PG → restart core (replay event log)
4. para volumes: extrair tar → montar volumes
5. para dbs: dump restore
6. reconciliação idempotente (units, certs, rotas)
7. restore.finished
```

## 6. Interfaces

```go
type BackupService interface {
    Create(ctx, spec BackupSpec) (*Backup, error)
    Restore(ctx, backupID string, opts RestoreOpts) (*Restore, error)
    List(ctx, filter) ([]Backup, error)
    Validate(ctx, backupID) error
}

type ObjectStore interface {
    Put(ctx, key string, r io.Reader, size int64) error
    Get(ctx, key string) (io.ReadCloser, error)
    Delete(ctx, key string) error
    List(ctx, prefix string) ([]ObjectInfo, error)
}
```

## 7. Eventos

Emitidos: `backup.scheduled`, `backup.started`, `backup.finished`, `backup.failed`,
`restore.started`, `restore.finished`, `restore.failed`, `backup.retention_purged`.
Consumidos: `app.deleted`, `database.deleted`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Backup diário do painel | snapshot íntegro em S3 |
| Restaurar após perda | plataforma volta + apps reconciliados |
| Backup de Postgres | dump + zstd + retenção 7 |
| Migrar para outro host | restore em novo host |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Manifesto de backup | restauração precisa | formato proprietário (versionado) |
| Compressão zstd | economia de SSD/banda | CPU de compressão (baixa) |
| Blob store plugável | flexibilidade | +1 integração |
| Janelas de baixa atividade | não conflita com produção | backup pode atrasar |

## 10. Decisões

- **D-001:** estado via SQLite VACUUM INTO / pg_dump.
- **D-002:** volumes via tar incremental; dbs via dump nativo.
- **D-003:** zstd para compressão; checksums.
- **D-004:** destinos: local, S3-compatível, SSH (plugin).
- **D-005:** retenção default 7; GC.
- **D-006:** janelas de baixa atividade; nunca conflitar com deploy.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Backup parcial | baixa | manifestos + checksums |
| Restore inconsistente | baixa | validação + reconciliação |
| Destino remoto indisponível | média | retry + alerta |
| Disco cheio | baixa | quotas + GC |

## 12. Alternativas descartadas

- **Backup contínuo por WAL**: descartado em v1 (complexidade; adiado fase 5).
- **MinIO local obrigatório**: descartado (P4; blob externo).
- **Copiar arquivos crus sem dump**: descartado (inconsistente para DBs).
