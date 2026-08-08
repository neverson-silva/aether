# F0 — Relatório Consolidado de Spikes

> **Status:** 5/6 hipóteses validadas; 1 aguardando execução Linux.
> **Data:** 2026-08-02
> **Objetivo:** fechar as decisões da Fase 0 (spikes) e registrar recomendações de ADR para
> as RFCs. Nenhuma mudança de RFC é feita silenciosamente — cada recomendação referencia a
> RFC-alvo.

---

## 1. Resumo de hipóteses

| Hipótese | Resultado | Números-chave | Report |
|----------|-----------|---------------|--------|
| H1 SQLite WAL contenção | ✅ **Confirmada** | zero stalls até 8 writers; ~25k ops/s; batch ×4 | [`sql-01-wal/report.md`](sql-01-wal/report.md) |
| H2 Outbox/event sourcing | ✅ **Confirmada** | ~80k ev/s outbox; snapshot 1M ev = 71 ms | [`sql-02-eventlog/report.md`](sql-02-eventlog/report.md) |
| H3 Linguagem | ✅ **Go** | Go 5,0 MB / Rust 2,4 MB RSS idle; ambos ≪ 120 MB | [`lang-go-rust/report.md`](lang-go-rust/report.md) |
| H4 RuntimeDriver | ✅ **Confirmada** | 18/18 passos contra engine real | [`runtime-driver/report.md`](runtime-driver/report.md) |
| H5 Traefik config em memória | ✅ **Confirmada** | rota ao vivo sem file-write e sem restart | [`traefik-dynamic/report.md`](traefik-dynamic/report.md) |
| H6 Podman rootless/Quadlet/Buildah | ⏳ Harness pronto | — (exige Linux) | [`podman-linux/`](podman-linux/) |

## 2. Decisões de arquitetura fechadas em F0

### D-ADR-001: Linguagem = **Go** (core e agent)

- Ambas cumprem o orçamento (H3). Decidiu **ecossistema OCI**: podman/buildah/skopeo/
  containers/* são Go → embutir `containers/image`/`containers/storage` no futuro com paridade.
- Goroutines multiplexadas (~GOMAXPROCS threads OS) são ideais para o core event-driven.
- Build limpo 0,22 s (8× mais rápido que Rust) — menor fricção de CI.
- Binário estático CGO=0, ~2 MB.
- **Registro**: atualizar ADR-001 em [`docs/05-arquitetura-geral.md`](../docs/05-arquitetura-geral.md)
  e [`docs/18-roadmap.md`](../docs/18-roadmap.md) (feito).

### D-PERSIST-01: SQLite `synchronous=NORMAL` + WAL + write queue com batch

- NORMAL é ~2,3× mais rápido que FULL; WAL dá durabilidade de transação commitada.
- Core: dispatcher único serializa escritas e faz batch (~8 tx ou 10 ms) — valida o modelo
  da RFC-0004.
- **Registro**: RFC-0004 §2.1/§7.

### D-EVENTS-01: outbox na mesma transação; snapshot a cada 10k eventos ou 5 min

- Outbox sem custo medido (A≈B). Snapshot de 1M eventos = 71 ms → recovery em ms.
- **Registro**: RFC-0005 §10; RFC-0004 §3.

### D-DRIVER-01: driver executa CLI OCI via subprocesso (sem SDK de vendor)

- Validado contra Docker real com 18 passos. Sem dependência de bibliotecas de vendor no core.
- **Registro**: RFC-0006 (adicionar nota de implementação).

### D-NET-01: config dinâmica via provider HTTP; pollInterval 5 s; reconcile via rawdata

- Validado com Traefik 3.7.10: rota adicionada ao vivo em memória, zero file-write.
- **Registro**: RFC-0002 §4/§5.

## 3. Ajustes de RFCs decorrentes (pendentes de execução)

| RFC | Ajuste proposto |
|-----|-----------------|
| RFC-0004 | `synchronous=NORMAL` default; write queue + batch commit; snapshot 10k/5min |
| RFC-0005 | registrar custo zero do outbox; parâmetros de snapshot |
| RFC-0006 | Pull retorna digest; LogStream.Close; nota "subprocess CLI" |
| RFC-0002 | provider HTTP + pollInterval; reconcile via rawdata |
| RFC-0007 | `podman stats --no-stream` (sob demanda) validado no driver |

> Edições serão aplicadas quando H6 (Linux) fechar, em lote, mantendo o ciclo de vida das RFCs.

## 4. Pendências para fechar F0

1. **H6 (podman-linux)**: executar `spikes/podman-linux/harness.sh` em Debian 12 e preencher
   `results.md` — fecha RFC-0001.
2. **Re-bench em Linux NVMe**: sql-01/sql-02 usam APFS/macOS; repetir no alvo Linux antes de
   fechar RFC-0004/RFC-0005 (limite de fsync é o fator).
3. **Benchmark comparativo Coolify/Dokploy/Aether** ([`docs/02`](../docs/02-analise-concorrentes.md) §8):
   executar em VM de referência (4 vCPU/8 GB/100 GB).
4. **Protótipo de `containers/image` embutido** (RegistryDriver): spike de Go com
   `github.com/containers/image` (skopeo como lib) — opcional antes da F1.

## 5. Impacto nas metas de engenharia ([`docs/03`](../docs/03-metas-engenharia.md))

- RAM idle: estimativa revisada para **< 60 MB** (Go core ~5 MB no micro-bench; proxy ~30 MB;
  agente dormido ~5 MB) — folga maior que os 120 MB originais.
- SSD: nenhuma imagem de suporte; binários Go ~2–5 MB — folga sobre os 300 MB.
- Processos: modelo validado (3 processos + worker efêmero) mantém ≤ 6.

## 6. Conclusão

F0 validou as **5 hipóteses mais arriscadas** do núcleo (persistência, eventos, linguagem,
runtime driver, networking). O risco remanescente é exclusivamente o **H6 (ambiente OCI rootless
em Linux)**, para o qual o harness está pronto. Com H6 fechado, a Fase 1 (MVP) pode começar com
as RFCs-0001..0006 ajustadas.
