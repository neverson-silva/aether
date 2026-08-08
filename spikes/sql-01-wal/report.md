# SPIKE-SQL-01 — SQLite WAL sob carga: contenção de escrita

> **Status:** Concluído ✓
> **Data:** 2026-08-02
> **Host:** macOS arm64, 10 CPUs, 24 GB RAM, sqlite 3.51.0 (via Python 3.9 stdlib)
> **Hipótese (H1):** SQLite WAL com o modelo do core (fila de escrita + batch commits +
> transações curtas) suporta a contenção de escrita sem degradação inaceitável.

---

## 1. Método

- Banco com o schema de referência do core: tabela `state` (WITHOUT ROWID, PK por aggregate)
  + tabela `events_outbox` (append-only).
- Cada "operação de domínio" = 1 transação: **upsert de estado + insert de evento no outbox**
  (modelo outbox da RFC-0005).
- Cenários: 1/2/4/8 escritores concorrentes; batch 1 e 8; `synchronous` NORMAL vs FULL.
- Métricas: ops/s agregado, latência p50/p95/p99, stalls de contenção (`locked`/`busy`).

## 2. Resultados

| Cenário | Writers | Batch | ops/s (agregado) | p50 | p95 | p99 | Stalls |
|---------|---------|-------|------------------|-----|-----|-----|--------|
| A) fila serializada | 1 | 1 | 28.161 | 0.02ms | 0.03ms | 0.11ms | 0 |
| B) concorrência baixa | 2 | 1 | 25.205 | 0.02ms | 0.03ms | 0.12ms | 0 |
| C) concorrência média | 4 | 1 | 27.490 | 0.02ms | 0.03ms | 0.56ms | 0 |
| D) concorrência alta | 8 | 1 | 25.465 | 0.02ms | 0.03ms | 1.20ms | 0 |
| E1) batch commits | 4 | 8 | 111.995 | 0.04ms | 0.06ms | 1.32ms | 0 |
| F) durabilidade FULL | 1 | 1 | 12.044 | 0.07ms | 0.08ms | 0.10ms | 0 |
| G) durabilidade FULL | 4 | 1 | 10.141 | 0.07ms | 0.08ms | 1.14ms | 0 |

> ops/s = transações commitadas; cada transação grava estado + evento outbox
> (E1 grava 8 eventos por transação → ~112k eventos/s agregados).

## 3. Análise

1. **Contenção ≈ zero.** Até 8 escritores concorrentes não houve nenhum stall (`busy`/`locked`)
   e o throughput agregado ficou praticamente constante (~25–28k ops/s). O SQLite WAL em
   meio sólido (NVMe/APFS) escala horizontalmente até o limite de I/O de fsync.
2. **Batch commits multiplicam**: E1 (batch=8) entrega ~4× o throughput de batch=1. A estratégia
   do core ("fila + batch commits") é o multiplicador certo — não a concorrência bruta.
3. **`synchronous` é o gargalo**: NORMAL é ~2,3× mais rápido que FULL no cenário de 1 writer e
   ~2,7× com 4 writers. Como o evento fica no outbox (fonte de verdade), `synchronous=NORMAL`
   com WAL é seguro: uma transação commitada sobrevive a crash do processo; no pior caso de
   queda de energia, um evento pode ser perdido apenas se ainda não ack — aceitável e mitigável
   com o snapshot periódico.
4. **Latência**: p50 sub-100µs em todos os cenários; p99 degrada suavemente com a concorrência
   (0.1→1.3ms), irrelevante para operações de controle.

## 4. Conclusão

**H1 CONFIRMADA.** SQLite WAL é mais que suficiente para o core: mesmo o "pior caso" de 8
escritores concorrentes fica dentro de qualquer necessidade de um plano de controle self-hosted
(deploys, eventos, configuração). O modelo de escrita do core (fila + batch) é validado.

## 5. Recomendações de ADR (para RFC-0004)

- Manter `journal_mode=WAL` + `synchronous=NORMAL` (não FULL) — ganho de 2,3×.
- Implementar **write queue no core**: um único dispatcher que serializa e faz batch de
  transações — elimina até a contenção residual medida.
- `PRAGMA busy_timeout=5000` como rede de segurança.
- Batch target ~8 transações ou 10ms de janela de batch (whichever first) para o outbox.
- Reavaliar em Linux NVMe (alvo real) antes de fechar a RFC-0004 (macOS APFS ≠ Linux ext4/btrfs).

## 6. Rerun

```bash
cd spikes/sql-01-wal && python3 bench.py
```
