# 04 — Análise de Consumo de Recursos e Estratégias de Minimização

> **Status:** Análise profunda. Base técnica das metas de [`03-metas-engenharia.md`](03-metas-engenharia.md).
> **Objetivo:** decompor cada fonte de consumo (SSD, RAM, CPU, processos, containers, imagens,
> volumes, logs, cache, builds, GC, snapshots) e definir a estratégia de minimização de cada uma.

---

## 1. Modelo de custo da plataforma

O custo total da plataforma em um servidor é:

```
C_total = C_runtime_fixo + C_operacional + C_estado + C_logs + C_cache + C_builds + C_backup
```

| Componente | Descrição | Onde Aether ataca |
|-----------|-----------|-------------------|
| `C_runtime_fixo` | Processos e bibliotecas sempre residentes | Mínimos processos; binário único; sem daemon de containers |
| `C_operacional` | Trabalho periódico (polling, crons, watchers) | Evento-driven; timers com backoff; sem watchers |
| `C_estado` | Banco de dados e estado persistido | SQLite com WAL e vacuum; event log append-only |
| `C_logs` | Logs do sistema e de apps | Rotação, retenção, estruturas compactas |
| `C_cache` | Caches em memória e em disco | LRU com limite; TTL; snapshot controlado |
| `C_builds` | Imagens, camadas, build cache | GC rigoroso; cache assinado limitado; build externo opcional |
| `C_backup` | Duplicação de dados | Compressão, diffs, alvos remotos |

---

## 2. Uso de SSD

### 2.1 Fontes de consumo de SSD

1. **Imagens de suporte da plataforma** (concorrentes): Postgres, Redis, Traefik, monitor,
   meilisearch, watchtower, minio → 2–8 GB.
2. **Imagens de aplicação do usuário**: base + camadas → por app (não é "desperdício", mas
   pode ser otimizado com deduplicação de camadas e podman share storage).
3. **Build cache**: camadas intermediárias de builds → dezenas de GB se não podados.
4. **Image cache / layer cache**: overlay diff caches, `vfs`? (podman storage dirs).
5. **Banco**: SQLite (arquivo), Postgres (catálogo + WAL + índices + tabelas temporárias).
6. **Logs**: arquivos de log do sistema, journald, logs de containers.
7. **Event log**: append-only de eventos (controlável por retenção + compactação).
8. **Snapshots/backups locais**: duplicação.
9. **Volumes de app**: dados de usuário (não é custo da plataforma, mas deve ter políticas).

### 2.2 Estratégias Aether de minimização de SSD

| Fonte | Estratégia |
|-------|-----------|
| Imagens de suporte | **Zero imagens de suporte.** Proxy, agentes e core são binários nativos. |
| Imagens de app | **Camada única por app** (squash final) com base compartilhada; podman storage compartilha camadas entre apps (deduplicação nativa via conteúdo endereçado). |
| Build cache | **GC por política**: limite de tamanho (ex.: 10 GB default configurável), eviction LRU por último uso, e **cache de camadas assinado** só para camadas base de imagens populares. Build em pasta dedicada com `no-cache` opcional. |
| Image cache | Podman prune agendado por evento de baixa atividade (janela noturna) com limite; nunca em janela de pico. |
| Banco | SQLite WAL com `VACUUM` incremental e `auto_vacuum`; event log compactado periodicamente (checkpoint em snapshot); Postgres (Business) com `pg_repack`-like opcional. |
| Logs | Rotação por tamanho e tempo; compressão zstd; retenção configurável por serviço (default: 100 MB/serviço ou 14 dias). Journald com limites rígidos. |
| Event log | Retenção por política (ex.: 90 dias para eventos não-críticos); compactação de snapshot elimina eventos antigos já materializados. |
| Snapshots/backups | Não manter backups locais por padrão; backup remoto compressível; snapshot em janela de baixa atividade com retenção limitada. |
| Volumes | `tmpfs` quando não persistente; quotas de volume por app (default com limite); documentar `nocopy`/`rsync` incremental. |

### 2.3 Estimativa de SSD de Aether (instalação limpa)

```
Binário core                  ~40 MB
Binário agent                 ~30 MB
Binário proxy (Traefik)       ~60 MB
SQLite (state + schema)       ~8 MB
Event log (0 eventos)         ~0.1 MB
Plugins essenciais            ~20 MB
Diretórios runtime/units      ~5 MB
------------------------------------------------
Total limpo                   ~165 MB   (< meta de 300 MB)
```

Após uso: imagens de apps dominam (necessário), controladas por políticas.

---

## 3. Uso de RAM

### 3.1 Fontes de consumo de RAM

1. **Processos sempre residentes**: dockerd, containerd, PHP-FPM, Node, Postgres, Redis,
   Traefik, monitor, workers.
2. **Runtime do plano de controle** (Node/PHP têm baselines altos).
3. **Caches em memória** (Redis, buffer pools).
4. **Threads/stacks**: cada thread Go ~2 MB de stack inicial (ajustável com `GOMAXPROCS`).
5. **Overhead de containers**: runc/crun + conmon por container.
6. **Garbage collector** e heap: GC de linguagem; tuning.

### 3.2 Estratégias Aether de minimização de RAM

| Fonte | Estratégia |
|-------|-----------|
| Daemons | Sem dockerd, sem containerd central, sem PHP/Node. Podman é daemonless. |
| Processos de suporte | Zero serviços de suporte em v1. |
| Linguagem | Binário compilado (Go/Rust): heap pequeno, sem VM, GC controlado (Go) ou sem GC (Rust). |
| Threads | `GOMAXPROCS` alinhado ao número de CPUs; workers efêmeros; idle timeout. |
| Cache em memória | LRU com `max_entries` e `max_bytes`; TTL; snapshot para disco e liberação. |
| SQLite | buffer pool limitado; `PRAGMA cache_size` configurado; WAL com `checkpoint`. |
| Container overhead | crun (C, ~menos RAM por container); conmon em modo leve; evitar containers de suporte. |
| Builds | Controle de concorrência de builds (2 default); builds em cgroup separado com limite de RAM. |
| Streams/logs | Buffers circulares com limite; streaming sem acumular (sockets). |

### 3.3 Estimativa de RAM (idle)

```
core (processo principal)      ~25 MB RSS  (Go, com caches pequenos)
agent (multi-server)           ~8 MB RSS   (dormindo; acorda por evento)
proxy (Traefik)                ~30 MB RSS
SQLite buffers                 ~8 MB
Misc/stack/caches              ~25 MB
------------------------------------------------
Total idle                     ~96 MB     (< meta de 120 MB)
```

---

## 4. Uso de CPU

### 4.1 Fontes de consumo de CPU

1. **Polling**: health checks, watchers de registries, crons fixos (schedule Laravel, NestJS).
2. **Coleta de métricas**: Prometheus scraping contínuo.
3. **GC da linguagem** (Node/PHP).
4. **Builds in-cluster** concorrendo com produção.
5. **Log ingestion** pesada (parsing contínuo).
6. **Serialização/desserialização** de configs em disco a cada deploy (Traefik file provider).

### 4.2 Estratégias Aether de minimização de CPU

| Fonte | Estratégia |
|-------|-----------|
| Polling | **Zero polling por padrão.** Tudo evento-driven. Timers com backoff determinístico e "poll só quando subscriber existe". |
| Health checks | Sob demanda (deploy/promoção) ou por serviço com intervalo configurável (default off). |
| Métricas | Coleta sob demanda; agregado em memória; exposto em `/metrics` sem scraping interno. |
| Proxy config | Dynamic config **em memória via API** — sem escrita de arquivo a cada deploy. |
| Builds | Concorrência limitada; priorização; builds deslocados para janelas ou servidores de build dedicados (fase 3+). |
| Logs | Parsing minimalista; formatos estruturados; streaming sem reparse. |
| GC linguagem | Go com `GOGC` tunado; evitar alocações em hot path. |

---

## 5. Número de processos

### 5.1 Processos desejados

| Processo | Resident | Descrição |
|----------|----------|-----------|
| `aether-core` | sim | Plano de controle: API, event bus, scheduler, dispatch, cache, cert engine, git, notifications |
| `aether-agent` | sim | Executor no servidor (local ou remoto): runtime OCI driver, logs, metrics, units systemd |
| `aether-proxy` | sim | Provider de proxy (Traefik) como processo host |
| `conmon` (por app) | por app | Supervisor de container (leve) |
| workers efêmeros | sob demanda | builds, backups, restores, git ops |
| `systemd` (core OS) | sim | Orquestra units dos apps |

Total residente de Aether: **3** (`core`, `agent`, `proxy`). Com workers efêmeros em uso:
+1 por operação em curso. Sempre ≤ 6 em repouso total.

### 5.2 Como garantimos "poucos processos"

- Um binário `core` embute: API, scheduler, dispatcher, event log, cache, certificate engine,
  git client, notification, marketplace. Nada disso é processo separado.
- O `agent` é um processo único que despacha operações de runtime em goroutines; quando o
  servidor é o local, o agent pode rodar no mesmo processo `core` (modo embedded) para reduzir
  a 2 processos.
- Proxy é o único "serviço externo" necessário.

---

## 6. Número de containers (plataforma)

| Container da plataforma | Existe? | Nota |
|-------------------------|---------|------|
| Banco (SQLite) | não | arquivo |
| Cache (Redis) | não | memória do core |
| Proxy | não | processo host |
| Monitor | não | agent |
| Search | não | FTS5 |
| Build runner | só durante build | efêmero |
| Apps do usuário | sim | units systemd (Quadlet) |
| DBs gerenciadas | sim (fase 2+) | via runtime OCI |

**Regra:** a plataforma só cria containers para o que é *trabalho* do usuário; nenhum container
existe para *servir* a plataforma em v1.

---

## 7. Número de imagens e camadas

- Imagens de suporte: **0** (v1).
- Imagens de app: controladas pelo GC de imagens (prune por uso/tempo).
- Camadas: deduplicadas pelo storage do podman (content-addressable); squash final reduz o
  número de camadas por imagem.
- Image store: diretório dedicado com quota (ex.: 20 GB default) e GC por LRU.

## 8. Logs

- Estrutura: JSON estruturado, compacto (campos mínimos).
- Rotação: por tamanho (ex.: 10 MB) e por tempo; compressão zstd; retenção default
  (14 dias ou 100 MB por serviço — configurável).
- Journald: limite de tamanho (`SystemMaxUse`).
- Streams ao vivo: via sockets unix, sem acumular em memória.
- Logs de app: diretório sob `/var/log/aether/apps/<app>/` com `symlink` para fácil acesso;
  nunca duplicados.

## 9. Cache

- **Em memória (core)**: LRU com `max_bytes` e `max_entries`; TTL por classe de dado;
  eviction explícita; snapshot para disco a cada X minutos com `fsync` controlado; na inicialização
  recarrega snapshot (leitura única).
- **Em disco**: SQLite (buffer), event log, imagens (GC), build cache (GC).
- **Nada cresce infinitamente**: todo cache tem limite físico + política de eviction + GC.
- Detalhes: [`11-cache.md`](11-cache.md).

## 10. Build Cache

- Diretório dedicado com quota (ex.: 10 GB).
- Eviction: LRU por último uso; camadas base assinadas e reutilizadas; `--squash` final.
- Não manter camadas de builds antigos após N builds ou X dias.
- Opção: `--no-cache` por app (parâmetro de deployment).

## 11. Image Cache / Layer Cache

- `podman` storage compartilha camadas entre imagens (deduplicação automática).
- GC de imagens: `podman image prune` agendado por evento de "low activity" (timer com
  janela noturna e limite de tamanho), não em horário de pico.
- Manter apenas: imagens referenciadas por apps + últimas N imagens não referenciadas.

## 12. Garbage Collection

Política central de GC (um módulo `gc` no core):

| Alvo | Trigger | Política |
|------|---------|----------|
| Imagens de app não referenciadas | evento `app.removed` + timer diário | manter últimas N (default 5) |
| Build cache | evento `build.finished` + limite | LRU por tamanho (quota 10 GB) |
| Event log | checkpoint de snapshot | retenção por idade/tipo |
| Logs | rotação | por tamanho/tempo |
| Cache em memória | TTL + eviction | LRU contínuo |
| Volumes órfãos | evento de remoção | `podman volume prune` com confirmação |
| Snapshots/backups | retenção | manter N (default 7) |

GC nunca roda durante janelas de deploy/backup.

## 13. Snapshots

- **Snapshot do estado (core)**: gerado a cada N eventos ou T minutos (default 5 min) com
  `fsync`; em crash, recovery = último snapshot + replay de eventos desde o snapshot.
- **Snapshot de banco (app)**: `VACUUM INTO`/`pg_dump` para backup; não usado em runtime.
- **Snapshot de volume**: opcional (fase 3+) com `tar` incremental ou snapshot de filesystem.
- Snapshot nunca bloqueia produção (copy-on-write / streaming).

## 14. Indicadores de eficiência (o que NÃO medir errado)

- **RAM medida com "free"** inclui page cache do kernel (que é *bom*). Medir RSS real dos
  processos da plataforma + RSS dos containers de suporte.
- **SSD** deve incluir imagens de suporte (que zeramos) e build cache (que limitamos).
- **CPU** deve ser medida em janela de idle (60 s) e durante operações de pico (deploy, backup)
  separadamente.

## 15. Caso de uso: servidor de 512 MB

Cenário meta: VPS 1 vCPU / 512 MB RAM / 20 GB SSD.

```
Custo fixo Aether:  ~96 MB RAM, ~165 MB SSD, ~0% CPU, 3 processos
Restante:           ~416 MB RAM para apps
Apps viáveis:       proxy + 2–3 apps pequenas (ex.: 1 Postgres pequeno, 1 Node, 1 Nginx)
```

Comparação com concorrentes: Coolify/Dokploy consumiriam 400 MB–1.5 GB → inviável em 512 MB.
Este é o caso de uso que diferencia o produto e deve ser verificado em CI.

## 16. Conclusão

Cada byte que economizamos é um byte para o usuário. A estratégia de minimização é sistêmica:
não é "otimizar um processo", é **eliminar categorias inteiras de consumo** — zero imagens de
suporte, zero daemons, zero polling, zero serviços opcionais por padrão — e aplicar **limites
físicos + GC** em tudo que ainda cresce (imagens, logs, caches, event log, backups).
