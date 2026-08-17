# RFC-0017 — API e CLI

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P6, P10
- **Dependências:** —

---

## 1. Objetivo

Definir a API REST (v1) e a CLI (`aether`) como interfaces programáticas de primeira classe —
a UI é apenas um cliente.

## 2. Escopo

**Dentro:** design REST (OpenAPI), autenticação (JWT/API keys), idempotência, streams (SSE/WS),
formato declarativo (`aether.yml`), CLI (comandos, saída JSON), webhooks de saída, rate limiting.
**Fora:** UI (consome API); lógica de domínio (fica nos módulos).

## 3. Responsabilidades

- Servir recursos REST com contrato OpenAPI.
- Autenticar/autorizar (RBAC enforced no core).
- Garantir idempotência em mutações.
- Prover streams (eventos, logs, terminal).
- Suportar import/export declarativo.
- CLI: instalação/operação local + cliente remoto.

## 4. Arquitetura

```
UI (SPA) ──► API REST/SSE/WS ──► Core (use cases)
CLI ───────► API REST (remoto)
CLI ───────► host ops (install/update/uninstall)
Webhooks (out) ──► HTTP(s)
```

## 5. Fluxos

### 5.1 Operação idempotente (deploy)

```
1. POST /api/v1/apps/:id/deploy + Idempotency-Key
2. valida RBAC
3. cria Deployment → 202 + Location: /deployments/:id
4. cliente assina SSE de eventos do deployment
5. terminal: deployment.ready
```

### 5.2 Stream de logs

```
1. GET /api/v1/logs?stream=app:<id>  (SSE)
2. core abre socket unix do log
3. encaminha eventos ao cliente; fecha ao sair
```

## 6. Interfaces

```go
// Exemplo de handlers
type API interface {
    AppAPI    // CRUD apps, deploy, rollback, logs
    DeploymentAPI
    ProjectAPI
    ServerAPI
    BackupAPI
    CertAPI
    DomainAPI
    TemplateAPI
    AuthAPI  // login, tokens, mfa
    KeyAPI   // api keys
    AuditAPI
    StreamAPI // SSE/WS
}
```

## 7. Eventos

Emitidos (via stream): todos os eventos relevantes (`deployment.*`, `service.*`, `backup.*`).
Consumidos: da UI/CLI (assinatura de streams).

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Deploy via CI | `202` + status assíncrono |
| Script de backup | API key com escopo |
| Terminal | WS autenticado |
| Migrar plataforma | export/import `aether.yml` |
| CLI em SSH | comandos completos |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| REST + SSE + WS | simples, padronizado | SSE não é bidirecional (usar WS p/ terminal) |
| Idempotency-Key | segurança em retry | cliente precisa cooperar |
| `aether.yml` declarativo | migração/import | formato precisa de versionamento |

## 10. Decisões

- **D-001:** API REST v1 + OpenAPI 3.1; streams SSE + WS.
- **D-002:** idempotência obrigatória em mutações longas.
- **D-003:** API keys com escopos; RBAC no core.
- **D-004:** CLI com `--json` para automação.
- **D-005:** `aether.yml` como formato de import/export/migração.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Breaking change | média | versionamento + changelog |
| Abuso | baixa | rate limiting |
| Import divergente | baixa | validação de schema + dry-run |

## 12. Alternativas descartadas

- **GraphQL**: descartado (mais complexo; REST cobre o modelo).
- **gRPC público**: descartado (ecossistema de clientes menor; gRPC é interno core↔agent).
- **Sem CLI (só UI)**: descartado (automação de primeira classe).
