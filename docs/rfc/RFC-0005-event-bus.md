# RFC-0005 — Event Bus

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P5 (eventos como fonte primária de verdade), P2, P10
- **Dependências:** RFC-0004 (Persistência)

---

## 1. Objetivo

Definir o **Event Bus**: comunicação assíncrona central, com eventos persistidos como fonte
primária de verdade (event sourcing), zero polling, sagas, timers determinísticos e suporte a
multi-node (gRPC) em fases futuras.

## 2. Escopo

**Dentro:** modelo de eventos, outbox, pub/sub em processo, transporte multi-node (gRPC),
sagas, timers persistidos, idempotência, recuperação/projeções, retenção.
**Fora:** UI; lógica de negócio específica de cada domínio.

## 3. Responsabilidades

- Publicar eventos com durabilidade (outbox).
- Entregar eventos a consumidores locais (barramento em memória).
- Distribuir eventos entre nós (core↔agent) quando multi-server.
- Persistir `scheduled_events` (timers) e re-agendar após restart.
- Manter checkpoints de consumo (idempotência).
- Compactar event log por snapshot.

## 4. Arquitetura

```
[Comando] → Tx(state + outbox) → Commit
              │
        Publisher (outbox → bus)
              │
   ┌──────────┴─────────────┐
   │  In-process bus        │   consumers locais (handlers)
   │  (zero custo)          │
   └──────────┬─────────────┘
              │
       [gRPC streaming] (multi-node, fase 3+)
              │
         Agent (remote) → executa
```

## 5. Fluxos

### 5.1 Publicação

```
1. comando chega (handler do domínio)
2. abre tx; persiste mudança de estado + eventos(outbox)
3. commit
4. publisher: lê outbox commitado → publica no bus
5. consumidores locais executam (handlers idempotentes)
6. remove linhas de outbox publicadas
```

### 5.2 Consumo idempotente

```
handler(event):
  if checkpoint.contains(aggregate, seq): return
  aplicar lógica (upsert por chave natural)
  checkpoint.record(aggregate, seq)
```

### 5.3 Timer

```
1. registrar scheduled_event (em banco)
2. quando devido → publicar evento
3. falha → retry com backoff (backoff table)
```

### 5.4 Recovery

```
1. boot: load último snapshot
2. replay eventos pós-snapshot (ordenados por seq/aggregate)
3. reprocessar outbox não-publicado
4. re-agendar timers pendentes
```

## 6. Interfaces

```go
type EventBus interface {
    Publish(ctx, events []Event) error
    Subscribe(pattern string, h Handler) Subscription
    Schedule(ctx, at time.Time, event Event, opts ScheduleOpts) error
    Checkpoint(ctx, consumer string) error
}

type Handler func(ctx context.Context, e Event) error

type Event struct {
    ID string; AggregateType string; AggregateID string
    Sequence int64; Type string; Payload json.RawMessage
    Meta map[string]string; Timestamp time.Time
}
```

## 7. Eventos

Emitidos: `event.published`, `event.consumed_failed` (DLQ), `scheduler.fired`.
Consumidos: todos os eventos de domínio (catálogo em `06-dominios-sistema.md`).

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Deploy (saga) | passos orquestrados por eventos; rollback por compensação |
| Auto-deploy por webhook | webhook → evento → pipeline |
| Timeline | agregação por aggregate_id (zero custo) |
| Alertas | regra avalia evento (não polling) |
| Multi-node | evento distribuído para agente executar |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Event sourcing | verdade única, recovery, auditoria | complexidade de projeção/compactação |
| Outbox same-tx | consistência | overhead em tx |
| Sem broker externo | zero custo, sem processo | reimplementar primitivas (DLQ, retry) |
| Barramento em memória | velocidade | reinício perde fila em memória (outbox persiste) |

## 10. Decisões

- **D-001:** eventos = fonte primária de verdade.
- **D-002:** outbox na mesma transação.
- **D-003:** transporte em memória (single) + gRPC (multi).
- **D-004:** sem broker externo obrigatório.
- **D-005:** timers persistidos; polling proibido.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Event log cresce | média | retenção + compactação |
| Ordem entre nós | baixa | sequência por aggregate + buffering |
| Consumer falho | média | DLQ + backoff + alerta |
| Replay pesado | baixa | snapshots frequentes |

## 12. Alternativas descartadas

- **NATS/Kafka**: processo externo; overkill em v1; custo fixo.
- **Redis pub/sub**: efêmero; viola durabilidade; P7.
- **Sem event sourcing (só CRUD)**: descartado (perde timeline/audit/recovery).
