# 12 — Event Bus

> **Status:** Fundação da arquitetura orientada a eventos.
> **RFC relacionada:** [`RFC-0005`](rfc/RFC-0005-event-bus.md)
> **Princípios atendidos:** P5 (eventos como fonte primária de verdade), P2, P10.

> **Implementação atual:** o barramento realtime (Redis pub/sub + Streams) e o hub WebSocket
> (`seq`/`replay`, eventos efêmeros) estão implementados para entrega de eventos ao frontend e
> fila durável de deploys. O Postgres continua a **fonte de verdade** dos estados; o event
> sourcing completo (outbox na mesma transação, LSM global, snapshots) é visão futura.

---

## 1. Propósito

O Event Bus é o sistema de comunicação assíncrona central da plataforma. Ele persiste eventos
como **fonte primária de verdade** (event sourcing) e os distribui para componentes inscritos.
**Polling é proibido sempre que possível** — o estado deriva de eventos, não de consultas
periódicas.

## 2. Por que eventos como fonte primária de verdade

| Problema do polling | Solução por eventos |
|---------------------|---------------------|
| CPU desperdiçada consultando estado imutável | Componente só acorda quando há evento |
| Latência entre mudança e efeito | Efeito imediato no subscriber |
| Estado espalhado em N lugares | Estado derivado de uma sequência de eventos |
| Debug difícil | Timeline completa de qualquer recurso |
| Recovery complexo | Replay de eventos + idempotência |
| Integrações frágeis (webhooks, crons) | Um evento, N consumidores |

## 3. Modelo de eventos

### 3.1 Estrutura do evento

```json
{
  "id": "evt_01HZ...",
  "aggregate_type": "deployment",
  "aggregate_id": "dep_01HZ...",
  "sequence": 42,
  "type": "deployment.ready",
  "payload": { "...": "..." },
  "metadata": { "actor": "user_01", "ip": "...", "request_id": "..." },
  "timestamp": "2026-08-02T10:00:00Z"
}
```

### 3.2 Requisitos

- **Ordem por aggregate**: sequence monotônica por (aggregate_type, aggregate_id).
- **Ordem global**: LSM (log-structured merge) por timestamp + id.
- **Durabilidade**: evento persistido na **mesma transação** do comando (outbox).
- **At-least-once**: consumidores devem ser idempotentes.
- **Retenção**: política configurável; compactação via snapshot.

## 4. Pipeline de escrita (Outbox)

```
[Comando] → Transaction[State change + INSERT eventos(outbox)]
→ Commit
→ Publisher: lê outbox commitado → publica no bus em processo
→ (multi-node) → distribui para agentes/outros nodes via gRPC
→ Consumidores locais: executam handlers (idempotentes)
→ (checkpoint de consumo por grupo de consumidor, se necessário)
```

- Sem outbox externo: o "outbox" são as tabelas de evento no mesmo banco.
- Publisher usa NOTIFY-like (SQLite hook/trigger) ou o próprio dispatcher do core para
  distribuir no processo.

## 5. Topologia de transporte

| Cenário | Transporte |
|---------|-----------|
| Instância única (dev/teste) | `AETHER_RUNTIME_BACKEND=memory` — pub/sub em processo + event log em memória |
| Produção / multi-instância (default) | `AETHER_RUNTIME_BACKEND=redis` — Redis **pub/sub** (`notify:org:<org>`) + **Streams** (event log `ev:org:<org>`, fila de deploy `q:deployments:*`) |
| Cliente (browser) | WebSocket único `/api/v1/ws/realtime` (hub por org), com `seq`/`replay` e eventos efêmeros |

Em produção o Redis é o barramento realtime entre instâncias da API e a fila durável de deploys
(consumer groups). O Postgres continua sendo a **fonte de verdade** dos estados; o event log Redis
é o canal de entrega realtime + trilha de auditoria recente (MAXLEN 5000 por org). O `install.sh`
e o `dev.sh` configuram `AETHER_RUNTIME_BACKEND=redis` por padrão.

## 6. Sagas

Operações de longo curso (deploy, backup, restore, migração de schema) são **sagas**:

```
deployment.created → saga[Build → Schedule → Health → Promote]
  cada passo é um handler de evento que emite o próximo evento
  falha → evento de compensação (rollback) → events de rollback
```

Sagas são idempotentes e rastreáveis pela timeline.

## 7. Timers (o "polling legítimo")

Nem tudo é evento externo; alguns fluxos precisam de tempo:

- **Retries**: timer com backoff exponencial (job falhou → agendar retry).
- **Crons**: agendamento determinístico (cron jobs do usuário).
- **Renovação de certs**: timer (D-30) com jitter.
- **Backup**: janela agendada.

Todos os timers são **determinísticos e limitados** (não há "polling contínuo"). Timers são
persistidos (não apenas em memória) para sobreviver a restart — no mesmo banco (tabela
`scheduled_events`).

## 8. Timeline e audit a partir de eventos

- A timeline de um recurso = eventos filtrados por aggregate_id.
- Audit = eventos com metadata de ator + ações administrativas.
- Nada é "logado" em paralelo ao que já é evento.

## 9. Eventos por domínio (catálogo resumido)

Ver [`06-dominios-sistema.md`](06-dominios-sistema.md) §4 e RFC-0005 para o catálogo completo.
Aqui, exemplos de categorias:

| Categoria | Exemplos |
|-----------|----------|
| Identity | `user.created`, `role.changed`, `apikey.created` |
| Applications | `app.created`, `app.updated`, `envvar.changed` |
| Deployments | `deployment.created`, `deployment.ready`, `deployment.failed` |
| Runtime | `service.started`, `service.crashed`, `service.scaled` |
| Build | `build.started`, `build.finished`, `build.failed` |
| Networking | `domain.added`, `route.updated`, `proxy.reloaded` |
| Certificates | `cert.issued`, `cert.renewal_failed`, `cert.revoked` |
| Backups | `backup.started`, `backup.finished`, `backup.failed` |
| Infrastructure | `server.heartbeat`, `agent.upgraded`, `cluster.formed` |
| Observability | `alert.fired`, `alert.resolved` |

## 10. Idempotência

Todos os handlers:
- Registram (aggregate, sequence) processados (tabela `consumer_checkpoint`).
- Se já processado → skip.
- Upserts são por chave natural.

## 11. Recovery e projeções

- Snapshot do estado a cada N eventos/T minutos.
- Recovery: carregar snapshot + replay de eventos pós-snapshot.
- Projeções leves (dashboard, contadores) são materializadas no banco e atualizadas por
  handlers — nunca por query pesada on-the-fly.

## 12. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Event log cresce | retenção + compactação (snapshot apaga eventos materializados) |
| Ordem fora de sequência (multi-node) | sequência por aggregate + reconexão com buffering |
| Consumidor morto (dead letter) | DLQ em tabela + retry com backoff + alerta |
| Replay pesado | snapshot frequente + replay em lotes |
| Bus de memória perde eventos em crash | outbox no banco garante durabilidade; publisher replays não-publicados |

## 13. Decisões

- **Decisão:** eventos são fonte primária de verdade (event sourcing com projeções).
- **Decisão:** outbox na mesma transação; transporte em memória (single-node) e gRPC (multi-node).
- **Decisão:** sem fila/mensageria externa obrigatória (sem Kafka/NATS/Redis).
- **Decisão:** polling proibido; timers determinísticos e limitados.

## 14. Referências

- RFC-0005 (Event Bus).
- Persistência (outbox/event log): [`10-persistencia.md`](10-persistencia.md) §3.
- Cache (invalidação por evento): [`11-cache.md`](11-cache.md).
