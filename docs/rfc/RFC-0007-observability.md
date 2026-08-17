# RFC-0007 — Observabilidade

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P2, P5, P8
- **Dependências:** RFC-0005 (Event Bus)

---

## 1. Objetivo

Definir o sistema de observabilidade: logs, métricas, tracing, alertas, timeline e audit —
com **desperdício zero** (coleta só com subscriber).

## 2. Escopo

**Dentro:** logs (rotação/retenção/streaming), métricas (agregação/expostção), tracing leve,
alertas por evento, timeline (derivada do bus), audit (append-only).
**Fora:** dashboards de terceiros (são consumidores via export opcional); infra observability do host.

## 3. Responsabilidades

- Coletar logs do core/agent/proxy/apps; rotação; streaming (SSE).
- Coletar métricas sob demanda (cgroup v2, /proc, stats sob demanda).
- Expor `/metrics` (Prometheus) só com subscriber.
- Tracing leve e amostrado; export OTLP opcional (plugin).
- Avaliar alertas em eventos + reconciliação de baixa frequência.
- Timeline por recurso (derivada do bus).
- Audit log append-only.

## 4. Arquitetura

```
                ┌─────────────────────────────────────┐
Logs ──────────►│ Log Manager (rotação, retenção, SSE)│
Metrics ───────►│ Metric Aggregator (janelas mem)     │
Tracing ───────►│ Tracer (sampling, OTLP opcional)    │
Events ────────►│ Alert Engine (regras em eventos)    │
Events ────────►│ Timeline Builder                    │
Audit ─────────►│ Audit Sink (append-only)            │
                └─────────────────────────────────────┘
```

## 5. Fluxos

### 5.1 Logs de app

```
conmon → arquivo /var/log/aether/apps/<app>/out.log
rotação por tamanho/tempo → zstd
stream: socket unix → SSE (UI/CLI)
```

### 5.2 Métricas

```
usuario abre dashboard → subscribe metrics para app X
agent coleta stats (podman stats sob demanda) → agrega janelas
sem subscriber → sem coleta
```

### 5.3 Alerta

```
evento service.crashed → AlertEngine avalia regras
≥3 em 10 min → alert.fired → notifica (e-mail/Slack/...)
health ok → alert.resolved
```

## 6. Interfaces

```go
type LogSink interface { Write(ctx, entry LogEntry) error }
type MetricSink interface { Record(ctx, name string, value float64, labels map[string]string) error }
type AlertRule struct {
    ID string; Condition string; Window time.Duration; Threshold int
    Actions []string
}
type AlertEngine interface {
    Evaluate(ctx, e Event) error
    Reconcile(ctx) error  // baixa frequência, só com regras ativas
}
type AuditSink interface { Record(ctx, a AuditEvent) error }
```

## 7. Eventos

Emitidos: `alert.fired`, `alert.resolved`, `log.rotated`, `metrics.subscribed`,
`metrics.unsubscribed`.
Consumidos: todos os eventos de domínio (para timeline/alertas).

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Ver logs de app | streaming via SSE em < 1 s |
| App crashando | alerta após 3 crashes |
| Dashboard CPU | métrica sob demanda |
| Quem apagou X? | audit |
| Timeline de deploy | lista de eventos |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Métricas sob demanda | CPU ~0 | picos não capturados sem subscriber |
| Alertas em eventos | reativo | precisar de reconcile |
| Sem store longo | zero SSD | sem histórico profundo (export resolve) |

## 10. Decisões

- **D-001:** logs estruturados JSON; rotação 10 MB/diária; retenção 14d ou 100 MB.
- **D-002:** métricas sob demanda; `/metrics` com subscriber.
- **D-003:** alertas por evento + reconcile 60 s (regras ativas).
- **D-004:** timeline/audit derivados do bus.
- **D-005:** tracing amostrado; OTLP via plugin (default off).

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Falha de dados sem subscriber | média | documentar; "ativar monitoramento" explícito |
| Alertas perdem estado | baixa | reconcile |
| SSD cresce por logs | baixa | rotação/compressão/rretenção |

## 12. Alternativas descartadas

- **Prometheus como dependência**: descartado (custo fixo; formato é export, não requisito).
- **Loki/EFK**: descartado (serviços externos; overkill v1).
- **Alertmanager**: descartado (integração externa; regras no core bastam).
