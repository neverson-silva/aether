# RFC-0011 — Pipeline de Deployments

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P5, P6
- **Dependências:** RFC-0001, RFC-0005

---

## 1. Objetivo

Definir o pipeline de deployment: build → schedule → health → promote, rollback, preview
deployments, e a coordenação via saga de eventos.

## 2. Escopo

**Dentro:** modelo Deployment, saga, build, schedule (units), health check, promote, rollback,
preview (PR/branch), triggers (manual, webhook, auto-deploy).
**Fora:** UI; runtime (RFC-0001); networking (RFC-0002).

## 3. Responsabilidades

- Orquestrar a saga de deploy com eventos e compensações.
- Executar build (via Execution Engine) quando necessário.
- Escrever units (Quadlet) via Runtime.
- Aplicar health checks e decidir promote/rollback.
- Manter histórico de deployments para rollback e timeline.
- Preview deployments (ambientes temporários).

## 4. Arquitetura

```
Deployments (domínio)
  DeploymentSaga
   ├── BuildStep      → Execution Engine (build)
   ├── ScheduleStep   → Runtime (units)
   ├── HealthStep     → HealthCheck
   ├── PromoteStep    → Networking (rota) + certs
   └── Compensation   → rollback
```

## 5. Fluxos

### 5.1 Deploy (imagem)

```
1. trigger (manual/webhook/git) → deployment.created
2. BuildStep (se source; senão skip) → build.finished | build.failed
3. ScheduleStep → unit new (canary slot) → service.started
4. HealthStep → se ok → PromoteStep
   │                  ├─ rota aponta para novo
   │                  ├─ unit antiga parada (após grace)
   │                  └─ deployment.ready
   └─ se health falhou → rollback automático (compensação) → deployment.failed + alerta
```

### 5.2 Rollback

```
1. POST /apps/:id/rollback?target=dep_X
2. saga: unit anterior → health → promote
3. deployment.rolledback
```

### 5.3 Preview (PR)

```
1. webhook PR opened → cria PreviewDeployment (branch build)
2. rota temporária <preview-id>.app.com (TLS via cert engine)
3. PR closed → remove preview + rota + cert
```

## 6. Interfaces

```go
type DeploymentService interface {
    Start(ctx, appID, opts DeployOpts) (*Deployment, error)
    Rollback(ctx, deploymentID, target string) error
    Cancel(ctx, deploymentID) error
    Status(ctx, deploymentID) (*DeploymentStatus, error)
}

type HealthChecker interface {
    Check(ctx, endpoint string, expect *Expect) error
    Watch(ctx, spec) (<-chan CheckResult, error)
}
```

## 7. Eventos

Emitidos: `deployment.created`, `deployment.building`, `deployment.scheduled`,
`deployment.health_checking`, `deployment.ready`, `deployment.failed`,
`deployment.rolledback`, `deployment.cancelled`.
Consumidos: `build.finished`, `service.started`, `source.updated`, `webhook.received`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Deploy com build | build+deploy < 2 min (app ref) |
| Deploy por imagem | < 10 s |
| Crash pós-deploy | rollback automático |
| Rollback manual | < 30 s |
| PR → preview | rota temporária |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Saga com compensação | robusto | complexidade |
| Canary slot | zero downtime | usa recursos temporários |
| Health check default off | custo zero | precisa ligar por app |

## 10. Decisões

- **D-001:** deploy como saga de eventos; compensação = rollback.
- **D-002:** canary slot (zero downtime) padrão.
- **D-003:** health check por app (default off); obrigatório para auto-rollback.
- **D-004:** rollback restaura unit + rota + cert refs anteriores.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Build falha | média | logs + retry |
| Health falso-negativo | baixa | janela + retries |
| Rollback inconsistente | baixa | idempotência por deployment |
| Concorrência de deploys | média | fila por app; estado por deployment |

## 12. Alternativas descartadas

- **Blue-green completo (2 stacks)**: descartado (overhead em v1; canary slot cobre).
- **Deploy destrutivo (parar antes de subir)**: descartado (downtime).
- **Sem health check (assume ok)**: descartado (pode promover quebra).
