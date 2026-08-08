# Fase 0 — Workspace de Spikes

> **Status:** Em execução.
> **Objetivo:** validar as hipóteses técnicas mais arriscadas antes de escrever produto
> (ver [`../docs/18-roadmap.md`](../docs/18-roadmap.md) §2).

---

## Board de status

| Spike | Pergunta | Ambiente de execução | Status | Saída |
|-------|----------|----------------------|--------|-------|
| [`sql-01-wal`](sql-01-wal/) | SQLite WAL sob carga: contenção de escrita | macOS + **Linux (ext4, VM)** | ✅ Concluído | [`report.md`](sql-01-wal/report.md) |
| [`sql-02-eventlog`](sql-02-eventlog/) | Outbox + event sourcing: throughput | macOS + **Linux (ext4, VM)** | ✅ Concluído | [`report.md`](sql-02-eventlog/report.md) |
| [`lang-go-rust`](lang-go-rust/) | Linguagem final (RAM, binário, startup) | macOS (rustc 1.97; go 1.26) | ✅ Concluído | [`report.md`](lang-go-rust/report.md) — **Go** |
| [`runtime-driver`](runtime-driver/) | Execution Engine: interface + driver | macOS (Docker como stand-in de semântica OCI) | ✅ Concluído | [`report.md`](runtime-driver/report.md) — 18/18 |
| [`traefik-dynamic`](traefik-dynamic/) | Config dinâmica via API em memória | macOS (binário Traefik 3.7.10) | ✅ Concluído | [`report.md`](traefik-dynamic/report.md) |
| [`podman-linux`](podman-linux/) | Podman rootless + Quadlet + Buildah | ✅ **Linux real (VM Debian 13 arm64)** | ✅ Concluído | [`results.md`](podman-linux/results.md) |
| [`acme-pebble`](acme-pebble/) | Certificates vs servidor ACME real | ✅ **Linux real (VM + Pebble)** | ✅ Concluído | [`results.md`](acme-pebble/results.md) |

## Benchmark idle em Linux (VM)

| Métrica | Resultado | Meta (`docs/03`) |
|---------|-----------|------------------|
| RAM (core `aether serve`) | **14 MB RSS** | < 120 MB ✓ |
| SSD (estado) | **284 KB** (+ binário 16 MB) | < 300 MB ✓ |
| CPU idle (60 s) | **0.1% us / 0.1% sy (99.8% idle)** | ≈ 0 ✓ |
| Processos Aether | **1** | ≤ 6 ✓ |

## SQLite re-bench em ext4 (Linux)

| Cenário | macOS | Linux (ext4) |
|---------|-------|--------------|
| WAL 8 writers (ops/s) | 25.5k | **42.9k** |
| Outbox ev/s | 79.9k | **158.1k** |
| Stalls | 0 | **0** |
| Snapshot 1M ev | 71 ms | 952 ms (disco VM menor) |

## Regras do workspace

1. Cada spike é autossuficiente (`run.sh` + `report.md`).
2. `report.md` segue: **Hipótese → Método → Resultado → Análise → Conclusão → Recomendação de ADR**.
3. Resultados que mudam decisão de RFC geram edição da RFC correspondente.
4. Spikes Linux-only (podman/buildah) entregam **harness executável** para rodar no alvo de
   referência (Debian 12, cgroup v2), pois macOS não suporta rootless OCI nativo.

## Hipóteses de F0 a validar

| # | Hipótese (da arquitetura) | Onde vive a decisão | Status |
|---|---------------------------|---------------------|--------|
| H1 | SQLite WAL suporta a contenção de escrita do core (fila+batch+transações curtas) | RFC-0004 | ✅ Confirmada (zero stalls) |
| H2 | Outbox + event sourcing em SQLite tem throughput suficiente para o core | RFC-0005 | ✅ Confirmada (~80k ev/s) |
| H3 | Linguagem compilada (Go/Rust) atende RAM idle < 120 MB | ADR-001 / RFC-0000 | ✅ Confirmada → **Go** (5 MB) |
| H4 | A porta `RuntimeDriver` é suficiente para expressar podman E docker | RFC-0006 | ✅ Confirmada (18/18) |
| H5 | Config dinâmica do proxy em memória (API) evita I/O por deploy | RFC-0002 | ✅ Confirmada (zero file-write) |
| H6 | Podman rootless + Quadlet + Buildah atende o modelo de execução | RFC-0001 | ⏳ Harness Linux pronto |

## Relatório consolidado

Ver [`F0-RELATORIO.md`](F0-RELATORIO.md).

## Como rodar

```bash
# todos os spikes que rodam neste host
./run-all.sh

# um spike específico
cd sql-01-wal && ./run.sh
```
