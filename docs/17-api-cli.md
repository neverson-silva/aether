# 17 — API e CLI

> **Status:** Interface programática.
> **RFC relacionada:** [`RFC-0017`](rfc/RFC-0017-api-cli.md)
> **Princípios atendidos:** P6 (automação de primeira classe), P10.

---

## 1. Propósito

API e CLI de **primeira classe**: a UI é apenas um cliente. Tudo que a UI faz, API e CLI fazem.
Automação (CI/CD, scripts, integrações) depende exclusivamente de API/CLI.

## 2. API REST (v1)

### 2.1 Princípios

- **REST** com JSON; **OpenAPI 3.1** como contrato (spec versionada).
- **Idempotência**: mutações com `Idempotency-Key` header; operações longas → `202` +
  `Location` do recurso de progresso (Deployment).
- **Versionamento**: `/api/v1/...`; breaking changes só em versão nova.
- **Paginação**: cursor-based para listas grandes.
- **Concorrência**: ETag/If-Match para updates.

### 2.2 Autenticação

- `Authorization: Bearer <token>` — JWT (sessão) ou API key (`sk_...`).
- API keys com escopos; RBAC enforced no core.

### 2.3 Recursos principais

```
/users, /orgs, /teams, /roles
/projects, /projects/{id}/environments
/apps, /apps/{id}/deployments, /apps/{id}/rollback
/deployments/{id}
/servers, /servers/{id}/agents
/domains, /certificates
/databases, /backups, /restores
/templates, /marketplace
/cron-jobs, /workers
/logs?stream=..., /metrics
/api-keys
/audit-logs
/plugins, /plugins/{id}/enable
```

### 2.4 Streams

- **SSE** `/api/v1/stream/events` — timeline, status de operações.
- **WebSocket** `/api/v1/ws/logs` e `/api/v1/ws/terminal` — logs ao vivo e terminal.

### 2.5 Errors

- Problem Details (RFC 9457): `{type, title, status, detail, instance}`.
- Códigos estáveis; corpo com validação (RFC 7807 compat).

## 3. CLI (`aether`)

### 3.1 Papéis

1. **Instalação/operação local** (host): `install`, `update`, `rollback`, `uninstall`, `status`.
2. **Cliente remoto**: autentica com API; todos os recursos.
3. **Scripting/CI**: saída JSON estável + exit codes documentados.

### 3.2 Comandos (v1)

```
aether login | logout | whoami
aether apps list|create|deploy|logs|rollback|rm
aether projects ...
aether servers add|list
aether backups create|restore|list
aether cron jobs
aether api-keys
aether update | rollback | uninstall | status
aether help
```

### 3.3 Formato de saída

- Default: tabela (TTY) / JSON (`--json`) para automação.
- `--output=yaml` para configs declarativas (import/export).

## 4. Configuração declarativa (import/export)

Formato YAML canônico (`aether.yml`) para descrever projects/apps/domains/backups:

```yaml
project: my-app
environments: [prod, staging]
apps:
  api:
    source: { git: { repo: git@github.com:me/api.git, branch: main } }
    build: { dockerfile: Dockerfile }
    domains: [{ domain: api.example.com, tls: letsencrypt }]
    env: { NODE_ENV: production }
    resources: { cpu: 250m, mem: 512Mi }
  db:
    image: postgres:16
    volumes: [{ name: pgdata }]
```

- **Export**: `aether export` gera o YAML do estado atual.
- **Import**: `aether import aether.yml` reconstrói (reconciliação idempotente).
- Serve como **caminho de migração** entre plataformas (Coolify/Dokploy exports adaptados).

## 5. Webhooks de saída (da plataforma)

- Webhooks para eventos (`deployment.ready`, `backup.failed`, `alert.fired`).
- Configurável por org/evento; HMAC assinado; retry com backoff.

## 6. Rate limiting e uso

- Limites por API key/org (configuráveis).
- `Retry-After`; respostas `429` com Problem Details.

## 7. Versionamento e compatibilidade

- API v1 estável durante v1 (sem breaking em minor).
- Deprecações anunciadas com 2 releases de antecedência.
- CLI verifica versão da API (`/api/v1/version`).

## 8. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Breaking change | versionamento por URL + changelog |
| Abuso de rate limit | limites + throttling |
| Terminal via WS inseguro | auth forte, RBAC, audit |
| Import/export divergente | validação de schema + dry-run |

## 9. Decisões

- **Decisão:** API REST + SSE/WS; UI é cliente.
- **Decisão:** idempotência obrigatória em mutações longas.
- **Decisão:** CLI com modo JSON para automação.
- **Decisão:** formato declarativo `aether.yml` para import/export/migração.

## 10. Referências

- RFC-0017 (API & CLI).
- Event Bus (streams): [`12-event-bus.md`](12-event-bus.md).
- RBAC: [`16-seguranca.md`](16-seguranca.md) §9.
