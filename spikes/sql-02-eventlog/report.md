# SPIKE-SQL-02 — Outbox + Event Sourcing em SQLite: throughput

> **Status:** Concluído ✓
> **Data:** 2026-08-02
> **Host:** macOS arm64, 10 CPUs, 24 GB RAM, sqlite 3.51.0
> **Hipótese (H2):** o pipeline de event sourcing do core (append → outbox → publish →
> projeção/checkpoint) tem throughput suficiente, e snapshot+replay (recovery) é barato.

---

## 1. Método

Schema de referência do core (RFC-0004/RFC-0005):

- `events` (append-only, PK `(aggregate_type, aggregate_id, sequence)`, coluna `published`).
- `projections` (projeção materializada por aggregate).
- `consumer_checkpoint` (idempotência de consumo).

Cenários:
- **A) append puro** — baseline de escrita.
- **B) outbox completo** — insert + marca `published=1` na mesma transação (caminho real).
- **C) consumer** — aplica projeção (upsert) + checkpoint por evento.
- **D) replay idempotente** — evento já processado → checa checkpoint → skip.
- **E) snapshot + recovery** — materializar última versão por aggregate em 1M eventos + scan.

## 2. Resultados

| Cenário | Carga | Throughput |
|---------|-------|-----------|
| A) append puro | 200k ev | **78.581 ev/s** |
| B) outbox completo | 200k ev | **79.875 ev/s** |
| C) projeção + checkpoint | 100k ev | **120.122 ev/s** |
| D) replay idempotente (skip) | 200k ev | **196.829 ev/s** |
| E) snapshot (1M ev) | 1M | **71 ms** (snapshot), **7 ms** (scan count) |

## 3. Análise

1. **Outbox não custa nada**: B ≈ A (79,9k vs 78,6k ev/s). Marcar `published` na mesma
   transação não adiciona custo mensurável. O padrão outbox da RFC-0005 é viável sem overhead.
2. **Projeção é mais barata que append**: C (120k ev/s) supera o append porque o upsert em
   `projections` (500 aggregates) tem muito menos linhas que o append de eventos. Projeções
   "leves por handler" não são gargalo.
3. **Replay idempotente é trivial**: D (197k checks/s) confirma que re-executar handlers em
   recovery (skip via checkpoint) é barato — recuperação por replay é segura e rápida.
4. **Snapshot de 1M eventos = 71ms**: recovery com snapshot + replay pós-snapshot fica em
   **milissegundos**. Mesmo um self-hoster com milhões de eventos acumulados reinicia o core
   instantaneamente.
5. **Comparação com a demanda real**: um plano de controle self-host gera da ordem de 1–100
   eventos/s em operação normal (deploys, webhooks, cron). Estamos 3 ordens de magnitude acima.
   Nenhum throttling é necessário em v1.

## 4. Conclusão

**H2 CONFIRMADA.** O event sourcing com outbox em SQLite é viável, rápido e com recovery
barato. Não há justificativa de performance para broker externo (Kafka/NATS) no core em v1 —
o gargalo nunca estará no event bus.

## 5. Recomendações de ADR (para RFC-0004/RFC-0005)

- Manter **outbox na mesma transação** (custo zero medido).
- **Snapshot a cada N eventos (ex.: 10k) ou 5 min** — suficiente; custo de 1M eventos é 71ms.
- Checkpoint de consumo com `INSERT OR IGNORE` (chave natural) — idempotência barata.
- Índice por `ts` e PK `(aggregate_type, aggregate_id, sequence)` confirmados.
- Reavaliar em Linux NVMe antes de fechar RFCs (disco alvo).

## 6. Rerun

```bash
cd spikes/sql-02-eventlog && python3 bench.py
```
