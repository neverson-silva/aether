# 02 — Análise dos Concorrentes: Engenharia Reversa de Coolify e Dokploy

> **Status:** Análise estática e de engenharia reversa baseada no conhecimento público de
> arquitetura, código aberto e comportamento observado dos projetos.
> **Objetivo:** Identificar as decisões arquiteturais que causam alto consumo de recursos e
> definir como Aether evita cada uma delas.

---

## 1. Resumo executivo

Coolify e Dokploy resolvem o mesmo problema com arquiteturas radicalmente diferentes, e ambas
têm o mesmo problema econômico: **a plataforma compete com o usuário pelos recursos do servidor**.

| Dimensão | Coolify | Dokploy | Aether (objetivo) |
|----------|---------|---------|-------------------|
| Linguagem núcleo | PHP (Laravel) | TypeScript (Node/NestJS) | Linguagem compilada estática (Go ou Rust) |
| Containers internos | Muitos (Postgres, Redis, Proxy, Monitor, Traefik, Meilisearch, etc.) | Muitos (Postgres, Traefik, Watchtower, minio, etc.) | Quase nenhum; processos host + 1 proxy |
| Banco | PostgreSQL em container | PostgreSQL em container | **PostgreSQL (decisão final — ver RFC-0021)** |
| Daemon de execução | Docker Engine + Traefik | Docker + Traefik + Watchtower | Podman + Quadlet (systemd) + proxy nativo |
| Orquestração de containers internos | docker-compose próprio | docker-compose próprio | systemd units / Quadlet |
| Build | Docker build in-cluster | Docker build in-cluster | Buildah rootless, build externo opcional |
| Processos residentes | 10–30+ | 10–25+ | ≤ 6 |
| RAM idle | 400 MB – 1.5 GB+ | 300 MB – 1 GB+ | < 120 MB (alvo) |
| SSD instalado | 3–10+ GB (imagens + camadas + builds + banco) | 3–8+ GB | < 300 MB (alvo) |
| Atualização | Docker pull + compose up | Docker pull + compose up | Troca de binário + migração transacional |
| Polling | Sim (workers do Laravel, jobs, watchers) | Sim (agendamentos NestJS, watchers) | Eventos; zero polling |
| Estado | Banco + Redis | Banco | Banco + log de eventos (event-sourcing) |

---

## 2. Coolify — Arquitetura observada

### 2.1 Stack e componentes

- **Núcleo:** PHP 8.x / Laravel (framework web). Roda como container ou no host.
- **Banco:** PostgreSQL 15+ (container oficial `coolify/db`), com volume persistente.
- **Cache/queue:** Redis (container `coolify/redis`) para cache, filas e filas de jobs.
- **Proxy:** Traefik (container) configurado dinamicamente via provider file, controlado pelo
  app Laravel.
- **Build runner:** build de imagens dentro do próprio servidor usando o Docker Engine (buildkit).
- **Monitor:** serviço de monitoramento (`coolify/monitor`) baseado em Prometheus para coletar
  métricas do host e de containers.
- **Search:** Meilisearch opcional (deployments/search).
- **Cron/queue workers:** `php artisan schedule:run`, `php artisan queue:work` etc.
- **Agents:** para multi-server, usa um container agente que se conecta ao painel.

### 2.2 Fluxo de deploy (resumo)

```
[Git push / Webhook] → [App Laravel recebe] → [Cria job de deploy na fila]
→ [Queue worker (php artisan queue:work) executa job]
→ [git clone/busca no diretório de build]
→ [docker compose build / docker build]
→ [docker compose up -d / docker run]
→ [Atualiza banco + traefik config + envia evento SSE/websocket à UI]
```

### 2.3 Consumo de recursos — causas raiz

| Item | Causa | Observação |
|------|-------|-----------|
| RAM | PostgreSQL + Redis + Traefik + Monitor + Laravel + workers | Cada container paga 50–300 MB; Postgres ~100–200 MB em idle; Redis ~5–20 MB; Traefik ~30–60 MB; monitor ~100+ MB |
| RAM | PHP-FPM/php workers: múltiplos processos PHP residentes | Vários workers `queue:work`, schedule runners |
| SSD | Imagens de todos os containers de suporte (postgres, redis, traefik, monitor, meilisearch) | ~2–6 GB de imagens |
| SSD | Banco com WAL + logs + snapshots + Meilisearch | Growth contínuo sem GC rígido |
| SSD | Build cache e imagens intermediárias sem GC agressivo | `docker system prune` agendado, mas defaults conservadores |
| CPU | Polling: schedule Laravel a cada minuto, health checks, monitor Prometheus scraping | Trabalho contínuo mesmo ocioso |
| CPU/IO | Build in-cluster concorre com produção | Concorrência direta de CPU/IO |
| Processos | docker daemon + containerd + postgres + redis + traefik + monitor + queue workers + php-fpm | Fácil 15–30 processos/containers |

### 2.4 Pontos fortes a preservar (design de produto)

- UX de onboarding excelente; instalação "one-liner" via script.
- Modelo de templates e marketplace rico.
- Suporte amplo a VPS e provedores (Hetzner, AWS, etc.).
- Documentação extensa; comunidade grande.
- Preview deployments, SSR/static classification.

### 2.5 Fraquezas arquiteturais a evitar

- Estado central em Postgres+Redis com jobs em fila → a atualização é um "compose up" frágil;
  filas podem re-executar jobs antigos.
- Muitos serviços "fixos" rodam mesmo quando não usados (ex.: monitor, search).
- Polling frequente para health checks e jobs agendados.
- Docker Engine como alicerce: daemon privilegiado, overlay2 storage driver, build cache pesado.
- Atualizações são operações Docker; rollback de versão da plataforma é complexo.

---

## 3. Dokploy — Arquitetura observada

### 3.1 Stack e componentes

- **Núcleo:** TypeScript / Node.js com NestJS (monorepo, backend e frontend separados).
- **Banco:** PostgreSQL (container `dokploy/db`) com Prisma ORM.
- **Proxy:** Traefik (container) dinamicamente configurado (provider file/dinâmico).
- **Deploy:** Docker Engine. Para builds, executa `docker build` (com cache) via remote API.
- **Watchtower:** atualização automática de containers (incluindo da própria plataforma).
- **S3/MinIO:** armazenamento de backups (container `minio` opcional), e suporte a S3 externo.
- **Agendamentos:** NestJS `@nestjs/schedule` (cron) para backups, checks, health checks.
- **CI (Dokploy v2 "dokploy CI"):** runner próprio de build em TypeScript, com registro de
  builders, em desenvolvimento ativo.
- **Migrations:** Prisma migrate.
- **Multi-server:** agent do Dokploy em servidores remotos (comunicação via Docker socket/tunnel).

### 3.2 Fluxo de deploy (resumo)

```
[Webhook/Git] → [Backend NestJS] → [Cria deployment]
→ [docker build com cache via Docker Engine]
→ [docker compose / docker run]
→ [Atualiza banco, gera configuração do Traefik]
→ [WebSocket/SSE atualiza UI]
```

### 3.3 Consumo de recursos — causas raiz

| Item | Causa | Observação |
|------|-------|-----------|
| RAM | Node.js runtime (200–400 MB baseline por processo) | NestJS é pesado em memória comparado a PHP ou binários estáticos |
| RAM | Postgres + Traefik + Watchtower + MinIO + Node backend | Multiplica o baseline |
| RAM | Prisma + múltiplos worker threads, processos de build | Builds Node consomem muito |
| SSD | Imagens de suporte (db, traefik, watchtower, minio) | ~3–6 GB |
| SSD | Cache de build Docker, images layer cache | Sem GC rígido por padrão |
| SSD | Backups no MinIO local (volume) | Backup duplica dados localmente |
| CPU | Cron de health checks e monitoramento, watchtower polling de registry | Trabalho periódico contínuo |
| Processos | Node backend + postgres + traefik + watchtower + docker daemon + agents | 10–20+ |

### 3.4 Pontos fortes a preservar

- UX moderna, "batteries included", onboarding guiado.
- Recursos nativos de banco de dados gerenciado e backups com S3.
- Multi-server razoavelmente maduro.
- Interface muito polida; alto foco em usabilidade.

### 3.5 Fraquezas arquiteturais a evitar

- Node.js como runtime do plano de controle é caro em RAM e CPU.
- `Watchtower` é um processo extra continuamente fazendo polling de registries.
- Docker Engine como alicerce (mesma crítica do Coolify).
- Atualizações via compose + watchtower: pouco determinísticas.
- Cron fixos com agendamentos constantes mesmo sem uso.

---

## 4. Decisões arquiteturais que provocam alto consumo — mapa geral

Consolidando as duas análises, as decisões que causam consumo elevado são:

| # | Decisão dos concorrentes | Efeito | Estratégia Aether |
|---|--------------------------|--------|-------------------|
| D1 | Rodar o plano de controle dentro de containers (Postgres, Redis, painel) | +RAM, +SSD, +processos, +atualização complexa | **Adotado: Postgres obrigatório** (RFC-0021); Redis opcional (fila em banco) |
| D2 | Docker Engine + containerd + dockerd daemon | Daemon privilegiado, storage overlay, cache pesado, processo grande residente | Podman rootless (sem daemon, sem containerd central), Quadlet → systemd units |
| D3 | Redis para filas/cache | +container, +RAM, +SSD | Fila em banco (Postgres outbox); cache em memória do processo único |
| D4 | Traefik como container único e central | Processo extra; configuração dinâmica via provider file causa escrita em disco a cada deploy | Proxy nativo gerenciado pelo Networking Engine; ainda Traefik, mas como processo host/unit com config em memória |
| D5 | Monitor Prometheus + exporters de suporte | +containers, +RAM, +CPU, +SSD | Coleta de métricas leve embutida no agente; sem Prometheus como dependência obrigatória em v1 |
| D6 | Queue workers PHP/Node residentes | Múltiplos processos sempre ativos | Workers efêmeros sob demanda; processo único que despacha goroutines/threads leves |
| D7 | Build in-cluster com buildkit | Concorre com produção; cache grande | Build rootless com Buildah; fila de builds; GC rigoroso; cache mínimo e assinado |
| D8 | Agendamentos/crons fixos (health check polling, prune) | CPU contínua mesmo ociosa | Cron dirigido por eventos + política de retenção; health checks sob demanda ou por serviço |
| D9 | Watchtower polling de registries | Processo + polling contínuo | Atualizações dirigidas por eventos/hooks, sem agente de polling |
| D10 | MinIO/backup local | Duplicação de dados em disco | Backups em volume compressível; alvos remotos opcionais; zero containers extras em v1 |
| D11 | Meilisearch/search index | +SSD, +RAM | Sem busca pesada; index em memória |
| D12 | Migração da plataforma como "compose up" | Downtime, risco, imagens novas cada update | Binário único + migração transacional + zero-downtime |
| D13 | Estado em Postgres+Redis com filas | Atualização/recovery complexo | Estado derivado de event log persistente; recovery é replay + idempotência |

---

## 5. Como Aether evita cada decisão — matriz de mitigação

### D1 — Plano de controle em containers → **processos host**

**Por que:** cada container de suporte adiciona: imagem (~100 MB–2 GB), camadas, processo,
RAM residente, ciclo de vida para gerenciar, e complexidade de atualização. Um processo nativo
(compilado, estático) tem overhead mínimo e é trivial de atualizar (troca de binário).

**Mitigação Aether:**
- Plano de controle = **1 binário** + **1 agente** (processos nativos).
- Banco padrão: **PostgreSQL 15+ (self-bootstrapping)** — decisão final (RFC-0021); SQLite removido.
- Postgres (quando habilitado) roda como serviço gerenciado pelo runtime OCI *apenas na
  edição Business*, fora do caminho crítico de v1.
- Nenhum Redis. Nenhum cache externo. Nenhum container de suporte em v1.

### D2 — Docker Engine → **Podman rootless + crun + Quadlet**

**Por que:** `dockerd` é um daemon privilegiado (root), com `containerd` interno, storage driver
`overlay2`, e mantém estado grande em `/var/lib/docker`. Podman é **daemonless**: cada operação
é um comando; não existe processo residente. Com **Quadlet** convertemos containers em units
systemd, e o **systemd** cuida do ciclo de vida com política de restart, de forma declarativa e
barata. **crun** (C em vez de Go) reduz RAM por container e tempo de start.

**Mitigação Aether:**
- Sem daemon central de containers → menos processos, menos RAM, menos SSD.
- Workloads de usuário = units systemd (Quadlet) → o sistema operacional é quem orquestra.
- Runtime rootless → sem daemon privilegiado, menor superfície de ataque.

### D3 — Redis para filas/cache → **fila persistente + cache em memória do processo**

**Por que:** Redis resolve filas/cache a custo de um serviço extra. Em volumes pequenos/médios,
o custo (RAM, SSD RDB/AOF, processo, monitoramento) não se justifica.

**Mitigação Aether:**
- Fila de trabalho: tabelas em SQLite (`pending_jobs`) processadas por um dispatcher único
  com **retry exponencial** — nada de polling; `NOTIFY`-like via trigger/read com backoff
  determinístico ou `syscall` de notificação quando disponível.
- Cache: mapa em memória LRU com TTL e GC, com snapshot para disco em intervalos controlados
  (ver [`11-cache.md`](11-cache.md)).
- Fases futuras (cluster/high-load): plugar Redis via plugin `cache.redis` — a abstração de
  cache já existe, a implementação é plugável.

### D4 — Traefik central em container → **proxy como processo gerenciado pelo Networking Engine**

**Por que:** Traefik é ótimo como **proxy dinâmico** (config em memória, service discovery),
mas rodá-lo como container com provider file (escrever `traefik.yml`/`dynamic.yml` em disco a
cada deploy) gera I/O e latência. Além disso, fixar a arquitetura no Traefik impede migração.

**Mitigação Aether:**
- O **Networking Engine** é a abstração; **Traefik é um Provider** (implementação padrão).
- Configuração é aplicada **via API em memória** (dynamic config HTTP provider) — sem escrita
  de arquivos a cada deploy.
- Em versões futuras: Providers Caddy, NGINX, HAProxy, Envoy sob a mesma interface.
- Proxy roda como unit systemd rootless, não como container de suporte.

### D5 — Monitor Prometheus → **observabilidade embutida e leve**

**Por que:** Prometheus + node-exporter + cadvisor = +3 containers/processos + RAM + SSD + CPU.
Para a maioria dos usuários self-hosted, o custo não se justifica.

**Mitigação Aether:**
- O agente coleta métricas do host (via `/proc`, `cgroup v2`, `podman stats` sob demanda)
  com custo desprezível e **somente quando há subscriber** (sem subscriber → sem coleta).
- Métricas por container obtidas sob demanda do runtime; agregadas e expostas via `/metrics`
  no formato Prometheus (quem quiser scrape usa; não somos dependentes de Prometheus).
- Alertas avaliados no evento (não por polling).

### D6 — Queue workers residentes → **dispatcher + workers efêmeros**

**Por que:** N workers sempre rodando = N × RAM + CPU constante.

**Mitigação Aether:**
- Processo único (`core`) com goroutines: despacha trabalho, e apenas **durante execução de
  jobs** há consumo real; entre jobs, o processo dorme (sem CPU).
- Em cargas altas, permite escalar workers efêmeros (executáveis) que sobem/descem sob demanda,
  com idle timeout.

### D7 — Build in-cluster com buildkit → **Buildah rootless, fila, GC rigoroso**

**Por que:** buildkit mantém cache grande e paraleliza em threads que competem com produção.

**Mitigação Aether:**
- Build via **Buildah rootless** em container próprio de build efêmero.
- Fila de builds: um por servidor por padrão (controle de concorrência de CPU/IO).
- Build cache: `cache` em pasta dedicada com **limite de tamanho + GC com política de retenção**
  (LRU por uso). Cache de camadas assinado para reuso seguro.
- Opção de **build externo** (CI/CD do usuário pusha a imagem) para quem quiser zerar o custo
  de build local.

### D8 — Crons fixos de health check → **avaliação por evento**

**Por que:** polling de health checks em N serviços = N requisições a cada intervalo, para sempre.

**Mitigação Aether:**
- Health checks: executados **por demanda** (deploy, promoção, alerta) ou com intervalo
  configurável **por serviço** (default: desligado a menos que o usuário ligue; ou "on-touch").
- Sem subscriber de métricas → sem coletor ativo.

### D9 — Watchtower → **updates dirigidos por evento**

**Por que:** Watchtower faz polling contínuo de registries e reinicia containers.

**Mitigação Aether:**
- Atualização de imagem de usuário: acionada por **webhook/evento** (registry webhook, Git push,
  comando) e executada com **política de saúde** (canary restart + rollback automático se health
  falhar).
- Nunca há processo de polling de registry.

### D10 — MinIO/backup local → **backup remoto/compressível, sem container extra**

**Mitigação Aether:**
- Backup nativo via `pg_dump`/SQLite `VACUUM INTO` + compressão (zstd) e upload opcional para
  S3/object storage (driver plugável).
- Sem container MinIO em v1. Volumes de app copiados com `tar` incremental ou snapshots de
  filesystem quando disponíveis.

### D11 — Meilisearch → **SQLite FTS5**

**Mitigação Aether:** busca full-text embutida no SQLite (FTS5) para catálogo de templates/apps.
Sem serviço de search externo.

### D12 — Atualização "compose up" → **binário + migração transacional**

**Mitigação Aether:** ver [`15-installer.md`](15-installer.md). Atualização = substituir binário,
rodar migrações transacionais, recarregar units. Rollback = restaurar binário anterior + migração
reversa. Zero-downtime.

### D13 — Estado Postgres+Redis+filas → **event log + estado derivado**

**Mitigação Aether:** ver [`12-event-bus.md`](12-event-bus.md). Todos os eventos são persistidos
em append-only log; o estado atual é projeção (snapshot periódico + replay). Recovery é replay +
idempotência. Migração de esquema não depende de filas vivas.

---

## 6. Diferenças de arquitetura que garantem a vantagem competitiva

| Eixo | Coolify/Dokploy | Aether |
|------|-----------------|--------|
| Modelo mental | "Gerenciador de Docker" com abstrações em cima | Sistema Operacional para Aplicações (runtime escondido) |
| Custos fixos | Pago por serviços de suporte sempre ativos | Pago somente pelo que existe (0 processo ocioso) |
| Atualização | Docker pull + compose up (vulnerável a imagem remota) | Binário reproduzível + migração transacional local |
| Segurança | Docker Engine root é superfície crítica | Podman rootless, sem daemon, units systemd isoladas |
| Recovery | Depende de Redis/Postgres/filas coerentes | Replay de event log, idempotente |
| Escalabilidade | Monólito centralizado; agents frágeis | Módulos + plugin system + multi-servidor por eventos |
| Migração saindo | Dados em Docker volumes/Postgres proprietários | Imports para Coolify/Dokploy (formato declarativo YAML) |

---

## 7. Aprendizados de UX a incorporar (não é só recursos)

1. **Onboarding guiado** (Dokploy é referência de fluidez).
2. **One-liner de instalação** com verificação de requisitos e mensagens claras.
3. **Detecção automática de ambiente** no instalador.
4. **Catálogo visual de apps** com deploy em 1 clique.
5. **Preview deployments** como experiência nativa (não apêndice).
6. **Monitoramento de qualidade de vida**: logs em tempo real na UI, streaming por SSE.
7. **Migrations transparentes**: o usuário não deve perceber o schema mudando.

---

## 8. Como validar a análise na prática (fase de benchmark)

Antes da implementação, montar **benchmark de referência**:

```
Cenário de benchmark: VPS limpa 4 vCPU / 8 GB / 100 GB SSD
1. Instalar Coolify → medir RAM/SSD/CPU/processos em idle
2. Instalar Dokploy → idem
3. Instalar Aether → idem
4. Deploy de 3 aplicações de referência em cada plataforma
5. Medir: tempo de deploy, RAM pico, SSD após builds, rollback
```

Relatório comparativo é aceite de arquitetura para as metas de
[`03-metas-engenharia.md`](03-metas-engenharia.md).

---

## 9. Conclusão

Coolify e Dokploy são produtos excelentes de UX, mas ambos pagam custos fixos altos por
arquitetura de "muitos serviços de suporte sempre ativos, baseados em Docker Engine, com
polling". Aether ataca exatamente esses custos: **planos de controle nativos, runtime OCI
daemonless (Podman rootless + systemd), evento-driven, sem serviços de suporte fixos** — o que
permite rodar a plataforma e as aplicações em hardware que os concorrentes não conseguem
atender com conforto. A paridade funcional em v1 garante que migrar seja barato; a diferença
de custo fixo garante que ficar seja vantajoso.
