# 11 — Cache

> **Status:** Estratégia de cache.
> **Princípios atendidos:** P2 (nada cresce infinitamente), P5, P7.

---

## 1. Propósito

Definir a estratégia completa de cache: camadas, políticas (TTL, LRU, eviction), GC e
snapshots. **Nada pode crescer infinitamente** — todo cache tem limite físico, política de
eviction e GC.

## 2. Onde há cache

| Cache | Local | Conteúdo |
|-------|-------|----------|
| `mem-lru` | core (RAM) | projeções quentes, config de UI, resultado de chamadas externas |
| `disk-object` | disco (dir) | blobs pequenos (templates, ícones, artifacts de marketplace) |
| `image-layer` | podman storage | camadas de imagem (deduplicação nativa) |
| `build-cache` | disco (dir de build) | camadas de build (Buildah) |
| `config-proxy` | memória (provider) | dynamic config do proxy |
| `db-buffer` | SQLite/PG buffers | pages de banco |

## 3. Cache em memória (core) — `mem-lru`

### 3.1 Estrutura

- Implementação: LRU com **límite duplo**: `max_entries` (ex.: 10 000) e `max_bytes`
  (ex.: 64 MB).
- TTL por classe de dado:
  - Config estática: 60 s
  - Resultados de APIs externas (git, cloud): 30–300 s (por classe)
  - Projeções quentes: até invalidação por evento
  - Token/credencial: o mínimo seguro

### 3.2 Eviction e invalidação

- **Eviction LRU** quando estoura qualquer limite.
- **Invalidação por evento**: ao processar `app.updated`, remove chaves de `app:<id>` do cache.
  (Eventos são o mecanismo primário de invalidação — P5.)
- **Explicit invalidation** por chave/prefixo.

### 3.3 Snapshot para disco

- A cada N minutos (default 5) ou em shutdown gracioso, o `mem-lru` gera um snapshot compacto
  (`cache/snap.db` — SQLite ou arquivo binário) com apenas chaves não-expiráveis ou de vida longa.
- Na inicialização: carrega snapshot (uma leitura) — não re-hidrata tudo.
- Snapshot com `fsync` controlado; corrompido → descarta e reconstrói.

### 3.4 Garantia de limite

- Invariante: `len(entries) <= max_entries` e `bytes <= max_bytes` após cada op.
- Métrica exposta: `aether_cache_size`, `aether_cache_hits`, `aether_cache_misses`.
- GC verifica desvio e força eviction.

## 4. Cache em disco — `disk-object`

- Objetos pequenos com TTL longo (templates, imagens de marketplace).
- Hash de conteúdo para deduplicação.
- Quota (ex.: 200 MB) + LRU em índice de metadados.
- Cleanup por GC diário.

## 5. Cache de camadas de imagem (podman storage)

- Deduplicação nativa (content-addressable): imagens que compartilham camadas não duplicam.
- GC de imagens: manter referenciadas + últimas N não-referenciadas (default 5).
- GC de camadas órfãs: `podman system prune` agendado em janela de baixa atividade.

## 6. Cache de build (Buildah)

- Pasta dedicada com quota (default 10 GB, configurável).
- LRU por último uso; camadas base assinadas reutilizadas.
- Opções por build: `cache-enabled` (default true), `no-cache` por app.
- `build.finished` → trigger de avaliação de GC.

## 7. Cache de config do proxy

- Mantido **em memória no provider** (Traefik dynamic config).
- Tamanho mínimo (config é pequena). Reconciliado no boot.

## 8. GC de cache (central)

O módulo `gc` (mesmo módulo do [`04`](04-analise-consumo-recursos.md) §12) executa:

| Alvo | Trigger | Ação |
|------|---------|------|
| `mem-lru` | a cada op (limites) | eviction LRU |
| `mem-lru` | snapshot | persistir + truncar |
| `disk-object` | diário (janela) | remover expirados/LRU além da quota |
| imagem/camadas | janela | prune |
| build-cache | `build.finished` + janela | podar LRU além da quota |

GC nunca roda em janelas de deploy/backup (agendado para idle).

## 9. Métricas de cache

- `aether_cache_*` (size, entries, hits, misses, evictions, snapshot_size).
- Relatório no dashboard Observability; alerta se eviction rate > limiar (sinal de cache
  subdimensionado).

## 10. Decisões

- **Decisão:** nenhum serviço de cache externo (sem Redis); caches dentro do `core`.
- **Decisão:** limite duplo (entradas + bytes) + TTL + invalidação por evento.
- **Decisão:** snapshot em disco com recuperação rápida; nada infinito.
- **Decisão:** GC centralizado com janelas de idle.

## 11. Referências

- Análise de consumo: [`04-analise-consumo-recursos.md`](04-analise-consumo-recursos.md) §9–12.
- Event Bus (invalidação por evento): [`12-event-bus.md`](12-event-bus.md).
