# Aether Platform — Documento Consolidado (canônico)

Este documento é a fonte única de leitura da Aether Platform. Ele consolida produto, princípios,
arquitetura, domínios, runtime, networking, persistência, eventos, segurança, operação, API,
CLI, plugins, Services, RFCs e roadmap em um único texto autocontido.

Os documentos especializados originais (`docs/00-*.md`…`18-roadmap.md`, `docs/rfc/*`,
`docs/spec/*`) permanecem no repositório como histórico/referência detalhada, mas nada aqui
depende deles. Quando houver conflito entre este documento e um documento antigo, **este
documento prevalece** e deve refletir a implementação atual.

---

## 1. Visão executiva

Aether é uma plataforma PaaS self-hosted para executar applications, workers, jobs, databases e
Services em infraestrutura própria. O produto prioriza OCI/Podman, baixo consumo de recursos,
rootless, segurança por padrão, recovery após falhas e deploy orientado a eventos.

- **PostgreSQL é a fonte de verdade** dos estados funcionais (deployments, apps, projects,
  environments, usuários, configurações, schedules, histórico, auditoria, monitoramento).
- **NATS transporta eventos, jobs, sinais e estado efêmero**:
  - **Core NATS** entrega realtime efêmero (sinais, notificações live, presença, broadcast).
  - **JetStream** entrega jobs duráveis, deployments, backups, restores, snapshots, cron,
    retries, DLQ, event log, replay, scheduler e KV (locks, estado).
- Hierarquia do produto: **Organization → Project → Environment → Service**. Um Service pode
  ser Application, Database, Worker, Cron ou Compose.

### 1.1 Modelo mental: "Sistema Operacional para Aplicações"

Um sistema operacional provê ao programa: abstração de hardware, gerenciamento de processos,
gerenciamento de recursos, sistema de arquivos, redes, segurança e serviços comuns. Aether
traduz esse modelo:

| Conceito de SO | Correspondente Aether |
|----------------|----------------------|
| Hardware | Servidores físicos/VMs/nós |
| Processos | Aplicações e seus deployments |
| Recursos | CPU/RAM/SSD/redes por ambiente |
| Sistema de arquivos | Volumes, armazenamento persistente |
| Redes | Domains, HTTPS, proxy, service discovery |
| Segurança | RBAC, secrets, isolamento, TLS |
| Serviços comuns | Logs, metrics, backups, observabilidade |

O usuário opera **conceitos de alto nível** (Applications, Projects, Deployments, Domains,
Servers, Organizations, Templates, Environments). **Containers são apenas uma implementação**
desse modelo — um detalhe do runtime, intercambiável e oculto.

**Consequência arquitetural inegociável**: o runtime nunca deve ser conhecido pelas camadas
superiores. Nenhuma parte do domínio de aplicação pode importar, referenciar ou depender de
Podman, Docker, containerd ou Kubernetes. Toda interação com o mundo de containers acontece
exclusivamente através da interface do Execution Engine (Runtime Driver).

### 1.2 Proposta central: eficiência de recursos

Toda infraestrutura consumida pela plataforma **reduz a capacidade disponível para os containers
do usuário**. Se a plataforma consome 1 GB de RAM e 5 GB de SSD em idle, o usuário perde
exatamente isso de capacidade produtiva. "Aether nunca deve competir por recursos com as
aplicações do usuário" é o critério de design para praticamente todas as decisões.

**Metáfora de aceite** (hardware 4 vCPU / 8 GB RAM / 100 GB SSD):
- Aether instalada e ociosa deve ser quase imperceptível: `top` não deve mostrar processos
  Aether entre os 10 primeiros consumidores de CPU.
- O custo fixo da plataforma deve caber na "casca" do sistema: containers leves de sistema,
  um processo supervisor, um processo de API, agents dormidos.

**Metas verificadas em CI** (benchmark de referência: VPS Debian 12, 4 vCPU, 8 GB, NVMe):

| Métrica | Meta v1 | Meta Fase 5 (Enterprise) |
|---------|---------|--------------------------|
| RAM total (plano de controle) | **< 120 MB** | < 256 MB (+ audit/HA) |
| SSD total (instalação limpa) | **< 300 MB** | < 500 MB |
| CPU idle (média 60 s) | **≈ 0%** (evento-driver) | < 1% |
| Processos residentes | **≤ 6** | ≤ 12 (inclui agentes) |
| Containers de suporte | **0** | 0 (continua) |
| Imagens de suporte | **0** | 0 |
| Threads do processo principal | ≤ 12 | ≤ 32 |

Justificativa: binário estático Go/Rust com heap pequeno + banco com buffers internos pequenos
+ cache LRU limitado + nenhum daemon; SSD = binário (~30–80 MB) + banco (< 10 MB) + runtime
directories; CPU ≈ 0 porque nenhum polling e todo trabalho é acionado por evento.

### 1.3 Metas de operação e capacidade

| Operação | Meta v1 | Notas |
|----------|---------|-------|
| Instalação limpa (script → pronto) | **< 2 min** | sem downloads de imagem de plataforma |
| Primeira inicialização | **< 5 s** | banco pequeno, migrações rápidas |
| `update` (sem downtime) | **< 60 s** | troca de binário + migração transacional |
| Rollback de versão da plataforma | **< 30 s** | binário anterior + migração reversa |
| Deploy de app com imagem pronta | **< 10 s** | pull + start + health check |
| Deploy de app com build local | **< 2 min** (app de referência) | build rootless local |
| Rollback de deployment | **< 30 s** | deployment anterior |
| Restart de app após crash | **< 5 s** | policy de restart |
| Reinício do host (recovery) | **< 90 s** até apps prontas | podman/systemd em boot |

Capacidade: ≥ 200 apps pequenos por nó; deployments simultâneos configuráveis (default baixo);
≥ 30 databases gerenciados; backup agendado não conflita com deploys (política de janelas +
locks). Qualidade: uptime ≥ 99,9% (um nó); zero-downtime em updates obrigatório; build de app
nunca bloqueia outro build (concorrência limitada); plataforma nunca faz OOM-kill em app do
usuário; monitoring nunca excede 1% de CPU extra (sem subscriber = sem coleta).

---

## 2. Princípios arquiteturais (inegociáveis)

| # | Princípio | Consequência prática |
|---|-----------|----------------------|
| P1 | OCI First, nunca Docker First | Abstração Execution Engine; drivers (Podman hoje; Docker/containerd/K8s futuros) |
| P2 | Zero desperdício estrutural | Nada roda sem uso; todo cache com limite físico; zero polling |
| P3 | Mínimos processos residentes | Work assíncrono em workers efêmeros; não em processos ociosos |
| P4 | Poucos containers, minimalistas | Sem containers "bônus" de suporte; processo host onde resolver |
| P5 | Eventos como fonte primária | Estado deriva de eventos; comunicação assíncrona persistente |
| P6 | Degradação e simplicidade operacional | Instalação 1 comando; update atômico e reversível; zero manutenção manual |
| P7 | Menos, mas composto | Bibliotecas pequenas e maduras; sem imagens desnecessárias |
| P8 | Os dados são do usuário | Sem telemetria obrigatória; backup/restauração são direitos |
| P9 | Segurança por padrão | Rootless; least privilege; secrets cifrados; auditoria |
| P10 | Crescimento sem reescrita | 1 servidor → N servidores/HA sem mudanças estruturais |

**P5 em detalhe**: proibir polling sempre que possível; o estado do sistema é derivado de uma
sequência de eventos; componentes se comunicam por um Event Bus assíncrono e persistente; tudo
o que pode ser reativo, é reativo. **P2 em detalhe**: nenhum serviço roda se não estiver em uso;
nenhuma rotina roda se não houver trabalho; nenhum cache cresce sem limite; nenhum log é
persistido sem política de retenção; toda operação periódica é acionada por evento ou agendada
com política rigorosa — nunca por polling cego.

### 2.1 O que NÃO é Aether

| Não é | Por quê |
|-------|---------|
| Um Docker Manager | Gerenciar containers é detalhe de implementação; o domínio opera aplicações |
| Um painel administrativo | Há API/CLI de primeira classe; a UI é um cliente |
| Uma PaaS gerenciada | É self-hosted; o usuário controla tudo |
| Um Kubernetes distro | K8s pode ser um driver futuro, nunca o modelo mental da plataforma |
| Um CI/CD | Build e deploy são primitivas; CI completo fica fora do escopo v1 |

### 2.2 Promessas ao usuário

1. Você nunca vai competir com a plataforma por RAM, CPU ou SSD.
2. Migração do Coolify ou do Dokploy em horas, não semanas.
3. Instalação em hardware que os concorrentes consideram insuficiente (ex.: VPS 512 MB–1 GB).
4. Suas aplicações continuam rodando mesmo se a plataforma for atualizada, reiniciada ou o
   painel estiver indisponível — os workloads são geridos pelo runtime OCI de forma declarativa.
5. Nada é obrigatório: cada provider, integração e módulo é carregado sob demanda.

### 2.3 Aplicação na engenharia

- Cada RFC cita explicitamente qual(is) princípio(s) atende.
- Decisão que aumente consumo em idle precisa de justificativa escrita e aprovação registrada.
- Metas de recursos são checadas em CI (benchmarks) a cada release candidato; regressão > 10%
  bloqueia merge exceto com justificativa aprovada.

---

## 3. Processos (executáveis)

### 3.1 aether-api

Atende HTTP e WebSocket, autentica, autoriza, executa queries, valida requests, persiste
comandos rápidos (commits de estado), cria jobs, publica eventos e grava o outbox. **NÃO
executa** deploys, backups, restores, cron contínuo ou coleta contínua de métricas. Tudo de
longa duração é encaminhado como job durável para o worker (ex.: `POST /deployments` →
valida → transação Postgres → cria deployment + outbox → commit → publica job → HTTP 202).

### 3.2 aether-worker

Executa deployments (build CNB/Dockerfile/pull), backups, restores, snapshots, cron, scheduler,
recovery de jobs interrompidos, outbox dispatcher, cleanup, watchers de runtime e provisionamento
de domínios. Consome JetStream durable consumers (pull) com ACK/NAK, retry backoff e DLQ.
Expõe `/health`, `/ready` e `/metrics` (porta 8081). Readiness só é verdade após validar os
consumers e com monitor contínuo do barramento.

### 3.3 aether-monitoring

Processo isolado com loop próprio (~2s): coleta host stats, container stats, `podman stats`,
`podman system df`, métricas de CPU/memória/rede/disco, classificação de ownership
(aether/user/system por labels), agregação e snapshots de monitoring — publica via Core NATS
(live) e persiste no Postgres com batching (agregado a cada tick; recursos a cada 5 ticks;
purge semanal). Porta de health 8082. A morte do monitoring não afeta API/worker/deploys; um
deploy pesado não atrasa o loop de monitoring.

---

## 4. Arquitetura de código

### 4.1 Hexagonal por feature (módulos)

Cada bounded context vive em `api/internal/modules/<feature>/` com até 4 pacotes:

| Pacote | Responsabilidade |
|--------|------------------|
| `domain` | Models, máquinas de estado, erros sentinela (`ErrNotFound/ErrConflict/ErrValidation/ErrForbidden`), interfaces de store |
| `application` | Casos de uso, validação + defaults, orquestração de stores |
| `http` | Handlers Gin + DTOs + helpers de rota |
| `infra` | Store Postgres (pgx + sqlc), Podman, adapters externos, criptografia |

O domínio **não conhece** Gin, pgx, Podman ou NATS. O `bootstrap` (`api/internal/platform/bootstrap`)
conecta configuração → pools → runtime → stores → services → handlers → router → http.Server.

### 4.2 Plataforma (packages transversais)

- `platform/api` — router Gin, middlewares (RequestID, RequestLogger, CORS, Timeout com exceção
  de `/api/v1/ws/`, auth, rate limit de login), `/health` + `/ready` + `/metrics`.
- `platform/config` — leitura de env (`AETHER_*`, `DATABASE_*`) com defaults; novos knobs
  entram aqui, nunca `os.Getenv` espalhado.
- `platform/database` — open/pool/EnsureDatabase/Migrate (migrations idempotentes).
- `platform/database/adapter` — drivers de databases gerenciadas (postgres, mysql, mariadb,
  mongo, mssql, oracle, redis) para introspect/catalog do Studio.
- `platform/druntime` — runtime de mensageria:
  - `adapter/nats` — implementação completa em Core NATS + JetStream (queue, events, pubsub,
    state KV, locks KV, presence KV TTL, cache local, rate limit local, scheduler).
  - `adapter/memory` — implementação em processo para testes.
  - `queue`, `events`, `pubsub`, `state`, `locks`, `presence`, `ratelimit`, `cache`,
    `scheduler` — contratos (interfaces).
- `platform/messaging` — `Envelope` (ID/Type/SchemaVersion/CorrelationID/CausationID/OrgID/
  ResourceID/CreatedAt/Payload) e catálogo central de subjects.
- `platform/outbox` — tabela `outbox_events`, claim (FOR UPDATE SKIP LOCKED), retry backoff,
  dispatcher com publicação direta de jobs `.queued` no JetStream.
- `platform/worker` — deploy worker (build/run/health/notify/recovery) + watcher.
- `platform/health` e `platform/observability` — health server por processo + métricas
  (jobs, reconciliação, publish errors, coleta de monitoring).
- `platform/infrastructure/pg/gen` — gerados por sqlc (`api/db/queries/*.sql` → gen). Nunca
  editar gerados à mão; editar o SQL e regenerar.

### 4.3 Configuração (env principais)

`AETHER_API_ADDR`, `AETHER_STATE`, `AETHER_RUNTIME_BACKEND` (nats|memory),
`AETHER_NATS_URL/NAME/USER/PASSWORD`, `AETHER_WORKER_HEALTH_ADDR`, `AETHER_MONITORING_HEALTH_ADDR`,
`DATABASE_HOST/PORT/NAME/USER/PASSWORD/SSL_MODE`, `AETHER_CNB_BUILDER`,
`AETHER_INGRESS_NETWORK`, `AETHER_FREE_DOMAIN_BASE/PROVIDER`, `AETHER_PUBLIC_URL`,
`AETHER_COOKIE_SECURE`, `GOOGLE_OAUTH_*`, `DATABASE_MIGRATE_ON_START`. Credenciais de
NATS geradas pelo installer em `~/.aether/keys/nats.auth`.

---

## 5. Deployments

### 5.1 Fluxo completo

```
request → deployment "queued" no PostgreSQL (fonte de verdade)
       → outbox (mesma transação: CreateDeploymentAndOutbox)
       → dispatcher publica job durável no JetStream (aether.jobs.deployments)
       → worker consome → "building" → "starting" → "health_checking" → "ready" | "failed"
       → Postgres atualizado + event log (deploy.*) + realtime + notification
```

### 5.2 Máquina de estados

- Transições válidas: `queued → building|failed|cancelled`; `building → starting|failed|cancelled`;
  `starting → health_checking|failed`; `health_checking → ready|failed`.
- Estados terminais: `ready`, `failed`, `rolled_back`, `cancelled`.
- Rollback: cria novo deployment a partir do último `ready`.
- **Deployments do mesmo Service são serializados** (lock por app `w.inFlight` + guard de
  status no Postgres); deployments independentes rodam em paralelo (concorrência do deploy
  worker = 4).

### 5.3 Build (estratégia)

1. **Dockerfile fornecido** pelo usuário → `podman build -f`.
2. **CNB (default)**: `pack build <img> -p <src> -B 127.0.0.1:5000/builder:node-spa`
   `--docker-host=inherit` `--pull-policy=never` `--platform linux/<arch>`; grupos
   `aether/spa-static` (SPAs/SSG) e `aether/node-server` (apps Node com servidor).
3. **comandos customizados** (`install_command`/`build_command`/`start_command`) → Dockerfile
   gerado (node build + nginx serve) quando configurado.
4. **Nada detectado** → erro claro com orientação (nunca fallback silencioso).

Ao sucesso: container `aether-<depID8>-<number>` na rede ingress, alias `app-<appID8>`,
health check (se habilitado) contra `http://<hostPort><path>` antes de `ready`; container
antigo removido após sucesso (libera a porta pública).

### 5.4 Recovery de deployments interrompidos

No startup do worker: deployments em `building/starting/health_checking` com `started_at`
antigo → `failed` (evita recriar containers duplicados); jobs `queued` sem mensagem no
JetStream → re-enfileirados. Mensagem no JetStream com estado terminal → ACK.

---

## 6. Runtime e build

### 6.1 Execution Engine

O Execution Engine é a única porta de entrada para o mundo de containers. A regra absoluta:
nenhuma outra parte da plataforma chama podman/docker/containerd; tudo passa pelo engine.

```
Execution Engine  (semântica neutra: run/stop/build/pull/push/logs/stats/exec/network/volume)
        │
        ├── Runtime Driver  (interface abstrata)
        │       ├── PodmanDriver     ← implementação padrão (v1)
        │       ├── DockerDriver     ← futura
        │       ├── ContainerdDriver ← futura
        │       └── KubernetesDriver ← futura
        │
        └── Build Driver
                └── Buildah/CNB (buildpacks) — v1
```

```go
type RuntimeDriver interface {
    Pull(ctx, ImageRef, auth) (ImageDigest, error)
    Push(ctx, ImageRef, auth) error
    ExistsImage(ctx, ref) (bool, error)
    PruneImages(ctx, opts) error
    Run(ctx, ContainerSpec) (ContainerHandle, error)
    Start/Stop/Restart(ctx, id) error
    Remove(ctx, id, force) error
    Exec(ctx, id, args, stdin) (ExecResult, error)
    Logs(ctx, id, since, follow) (LogStream, error)
    Stats(ctx, id) (ContainerStats, error)
    List(ctx, selector) ([]ContainerInfo, error)
    NetworkCreate/Delete/Inspect(ctx, name) error
    ContainerConnectToNetwork(ctx, id, net) error
    VolumeCreate/Delete/Inspect(ctx, name, opts) error
    Build(ctx, BuildSpec) (ImageDigest, error)
    Info(ctx) (RuntimeInfo, error)
    GC(ctx, policy) (GCReport, error)
}
```

### 6.2 Por que Podman + Buildah + Skopeo + Quadlet + conmon + crun

- **Podman**: engine daemonless, rootless; zero daemon central (menos RAM/processos); storage
  content-addressable (dedup de camadas); CLI/API compatível com Docker; gerencia quadlets e
  pods; base OCI real. Trade-offs: redes rootless mais complexas (mitigado com
  slirp4netns/pasta e rede host/socket quando apropriado), portas < 1024 exigem
  ambient capabilities ou redirecionamento, alguns recursos avançados (GPU) menos maduros.
- **Buildah**: build de imagens OCI sem daemon e sem container em execução; cache controlável
  (`--squash`, `--cache-from`); compatível com Dockerfile; exige fila/concorrência própria —
  exatamente o controle desejado.
- **Skopeo**: inspecionar/copiar imagens sem pull completo; verificação de assinatura; útil
  para registry local/mirrors e supply chain.
- **Quadlet**: converte declarações de containers/pods/volumes/network em units systemd;
  systemd dá restart policy, resource limits (CPUWeight/MemoryMax), socket activation, boot
  ordering; isolamento por user instance; recovery nativo pós-reboot; modelo declarativo.
- **conmon**: supervisor minimalista (stdio, PID, status, logs, terminal).
- **crun**: runtime low-level em C; menos RAM e start mais rápido que runc; substituível por
  qualquer OCI runtime via config.

### 6.3 Modelo de execução via units

`Application spec → UnitSpec → arquivos .container/.pod/.volume/.network → systemctl --user
enable/start → podman run (crun) + conmon`. Unidades por serviço: `app-<id>-main`, worker,
db, cron timer; recursos com limites injetados; rollback = restaurar unit anterior + restart.

### 6.4 Builds CNB (SmartBuild) — detalhe

- Builder local: `127.0.0.1:5000/builder:node-spa`, construído por
  `infra/buildpacks/builders/build-builder.sh` (**podman build**, nunca `pack builder create`
  — "duplicate paths" no unpack com podman), publicado no registry local `127.0.0.1:5000`.
- Buildpacks **100% aether** (sem Paketo): `aether/spa-static` e `aether/node-server`
  (+ php-server, ruby-server, dotnet-server, go-server, rust-server, jvm-server).
- Builder: FROM `ubuntu:24.04`, stack `io.buildpacks.stacks.aether` com run image
  `docker.io/library/ubuntu:24.04`, lifecycle 0.21.x + buildpacks em `/cnb/buildpacks`,
  `order.toml` e labels de metadata.
- Daemon de build: `--docker-host` apontando para o socket do podman (VM forward no Mac via
  gvproxy; socket montado no container).
- O instalador garante builder + registry + run image (`podman pull ubuntu:24.04`) e possui
  stamp por sha256 para rebuild incremental.
- Log de diagnóstico por deploy: "cnb: detected framework=... type=... build=... output=...".


## 7. Networking Engine

### 7.1 Propósito e abstração do proxy

O Networking Engine administra toda a exposição de aplicações à rede: domínios, HTTPS,
TLS/termination, balanceamento, middlewares e service discovery. Ele **abstrai o proxy** —
Traefik é apenas um Provider (implementação padrão). No futuro: Caddy, NGINX, HAProxy, Envoy
sob a mesma interface.

Razões para abstrair: antilock-in (providers mudam de ecossistema a cada ciclo); semântica
única (o domínio fala "domínio", "rota", "middleware"); migração = trocar driver, não
reescrever a plataforma; custo/recursos controlados.

```go
type ProxyProvider interface {
    Name() string
    Apply(ctx, ProxyConfig) error        // config dinâmica em memória
    GetStatus(ctx) (ProxyStatus, error)  // versão, backends, certs carregados
    Reload(ctx) error
    Certificates() ([]ProxyCert, error)  // somente refs/estado
}
```

`ProxyConfig` é neutra: lista de rotas, middlewares, TLS refs e load balancers.

### 7.2 Traefik como provider padrão — por quê

| Critério | Traefik | Caddy | NGINX | Envoy |
|----------|---------|-------|-------|-------|
| Config dinâmica em memória (API) | excelente | boa (JSON API) | média (reload) | excelente (xDS) |
| Service discovery nativo | excelente | boa | requer lua/código | excelente |
| Middlewares prontos | muito bons | razoáveis | bons | bons |
| HTTP/3 | suporte | suporte | parcial | suporte |
| RAM | ~30–60 MB | ~15–30 MB | ~10–25 MB | ~50–100 MB |
| Cert manager integrado | sim (mas usamos o nosso) | sim (Autocert) | via certbot | via mTLS/CA |
| Modelo declarativo | ótimo | ótimo | declarativo | declarativo (xDS) |

**Decisão**: Traefik em v1 — melhor equilíbrio entre dinâmica em memória, middlewares, HTTP/3 e
maturidade. Caddy candidato para v2 se a pressão de RAM exigir; NGINX/Envoy para casos
específicos.

### 7.3 Configuração sem escrita em disco

Certificates emite/mantém certs; Networking constrói `ProxyConfig` em memória; o proxy aplica
via API HTTP de dynamic config — **sem arquivo em disco a cada deploy**. Reconciliação: se o
proxy cair, no boot o estado desejado é reaplicado.

### 7.4 Fluxos principais

**Adicionar domínio a uma app**:

```
[UI/API] → domain.add → Applications.domain.attached
→ Networking: cria Route(domain, target)
→ Certificates: solicita/renova cert (HTTP-01 ou DNS-01)
→ ProxyConfig atualizada → apply (API Traefik)
→ events: domain.added, cert.issued, proxy.reloaded
```

**Terminação TLS**: o proxy termina TLS (443) e entrega HTTP ao app na rede interna; refs de
cert vêm do Certificate Engine (nunca o proxy controla emissão).

**Service discovery**: cada app/serviço se registra como backend ao iniciar
(`service.registered`); rota dinâmica por hostname/path sem reiniciar.

### 7.5 Domínios, HTTPS e desafios ACME

| Desafio | Uso | Provider |
|---------|-----|----------|
| HTTP-01 | domínios com apontamento pronto | Let's Encrypt (default) |
| DNS-01 | wildcard, domínios sem porta 80 | Let's Encrypt + Cloudflare DNS, Route53, etc. (via plugin DNS) |

- Wildcard: só via DNS-01. Renovação: D-30 com jitter (evitar thundering herd); falhas geram
  evento `cert.renewal_failed`.

### 7.6 Middlewares (v1)

| Middleware | Descrição |
|-----------|-----------|
| Rate Limit | limite por IP/client; configurável por rota |
| Forward Auth | delegar autenticação a serviço externo (ex.: authelia) |
| Headers | injetar/remover cabeçalhos; security headers |
| Rewrite | reescrita de paths |
| Redirect | redirects (http→https, apex→www) |
| Auth basic | proteção simples por rota |

### 7.7 Load balancing, HTTP/3 e rede interna

- LB: round-robin entre réplicas; affinity por IP (sticky) opcional; health check upstream
  marca backend indisponível.
- HTTP/3 (H3+QUIC) por app, opt-in, fallback transparente para HTTP/2/1.1.
- Redes podman por app; service discovery por nome de rede; apps em rede privada não expõem
  portas públicas (somente o proxy fala com elas, ou porta publicada quando o usuário pedir).

### 7.8 Dados de domínio

```
Domain { id, name, tls {provider, challenge}, wildcard bool }
Route { domainRef, appRef, port, path, middlewareRefs[], lb {...} }
Middleware { type, config }
ProxyConfig { routes[], middlewares[], certs[], tlsMinVersion }
```

Domínio real no produto: `modules/domains` (application + provision + worker), free domains
sob `AETHER_FREE_DOMAIN_BASE` (ex.: `*.apps.aether.local`, nip.io by default) e domínios
custom com ACME; configuração dinâmica do Traefik escrita na shared volume `aether-traefik`.
Em dev, o container Traefik (`aether-traefik`) é o entrypoint 80/443; domínios free usam
DNS nip.io/sslip.io/traefik.me.

---

## 8. Certificate Engine

### 8.1 Soberania

O Certificate Engine é o componente **único e soberano** para certificados TLS: emissão,
renovação, revogação, histórico, auditoria e alertas. Let's Encrypt e outros ACME são apenas
providers. **O proxy nunca controla certificados** — apenas consome refs. Isso garante troca de
proxy sem quebrar a cadeia de certs e centraliza a política de renovação auditável.

Por que não delegar ao proxy: soberania da renovação (Traefik/autocert não expõem histórico/
auditoria); consistência multi-proxy; wildcard via DNS-01 exige integração com providers de DNS;
auditoria/compliance (quem emitiu, quando, qual domínio, qual provider); alertas proativos;
controle de custo (um único processo de renovação com jitter).

### 8.2 Arquitetura

```
Certificate Engine (Core)
   ├── Policy Manager   (regras: providers, contas, renovação, retenção)
   ├── Account Manager  (contas ACME)
   ├── Challenge Runner (HTTP-01 via proxy; DNS-01 via plugin DNS)
   ├── Store            (certs + chaves criptografadas; histórico)
   └── Notifier         (eventos de emissão/renovação/falha)
   └── Provider (ACME)
         ├── Let's Encrypt (default)
         ├── ZeroSSL (futuro)
         └── Private CA (futuro, enterprise)
```

```go
type AcmeProvider interface {
    Register(ctx, email) (Account, error)
    OrderCertificate(ctx, Order) (OrderResult, error)
    CompleteChallenge(ctx, Challenge, proof) error
    GetCertificate(ctx, Order) (CertBundle, error)
    Revoke(ctx, cert, reason) error
}

type DnsProvider interface {
    UpsertTxtRecord(ctx, name, value, ttl) error
    DeleteTxtRecord(ctx, name, value) error
}
```

### 8.3 Fluxos

**Emissão HTTP-01**:

```
domain.added → CertEngine.ensure(domain) → Policy → Account ACME → Order
→ Challenge http-01 (token) → proxy serve challenge → ACME valida → Order finaliza
→ CertEngine baixa cert → criptografa e persiste → refs no proxy (config em memória)
→ events: cert.issued
```

**Emissão DNS-01 / wildcard**:

```
wildcard domain.added → Order wildcard → Challenge dns-01
→ DnsProvider.UpsertTxtRecord (plugin) → aguarda propagação (poll curto com timeout)
→ ACME valida → cert → persiste → refs no proxy → cleanup DeleteTxtRecord
```

**Renovação**: D-30 com jitter por domínio; retries com backoff; após N falhas → evento
`cert.renewal_failed` + alerta; nunca deixar expirar (renew agressivo perto do vencimento).

**Revogação**: manual (UI/API) ou automática (chave comprometida); notifica ACME e remove refs.

### 8.4 Storage, política e riscos

- Chaves privadas criptografadas (AES-256-GCM; chave derivada da master key) em
  `/var/lib/aether/keys/certs/<domain>/privkey.pem.enc`; cert público legível; histórico em
  `cert_events`; permissões 0600; o core injeta refs no proxy (in-memory por default).

| Parâmetro | Default | Nota |
|-----------|---------|------|
| Renew window | D-30 | configurável |
| Retry backoff | 1h, 2h, 4h... máx 24h | |
| Max failures antes de alerta | 3 | |
| Retenção de histórico | 1 ano | audit |
| Manter certs revogados | 30 dias | diagnóstico |

Riscos: rate limit de ACME (jitter + SAN + DNS-01), renovação silenciosa (alerta + renew
agressivo), chave comprometida (revogação + rotação), DNS provider indisponível (retry +
fallback HTTP-01 quando possível).

---

## 9. Persistência, outbox e recovery

### 9.1 PostgreSQL (único banco, fonte de verdade)

Persiste: estado desejado e observado, configurações, deployments, jobs, histórico, auditoria,
notifications, outbox, schedules, cron jobs, snapshots, monitoring e secrets cifrados.
Migrações: `api/db/migrations/0NNN_*.sql` (ex.: 0026_outbox), aplicadas no start
(`DATABASE_MIGRATE_ON_START`), idempotentes e transacionais; geradas via sqlc
(`api/db/queries/*.sql` → `platform/infrastructure/pg/gen`).

Tabelas centrais (seleção): `users, orgs, members, api_keys, audit_logs, projects,
environments, apps, app_env, env_variables, deployments, domains, previews, cron_jobs, workers,
app_policies, databases, backup_configurations, backup_jobs, restore_jobs, snapshots,
snapshot_schedules, alerts*, notification*, s3_destinations, oidc_providers, branding,
webhooks, compose_apps, gitops, servers, clusters, pipeline*, mirrors, volumes, monitoring_*,
host_stats, outbox_events, variable_audit, scm_*`.

### 9.2 Outbox (transactional outbox)

O outbox grava a intenção de publicação **junto** da mudança funcional quando atomicidade é
necessária (ex.: `deployment.queued` em `CreateDeploymentAndOutbox`):

```sql
CREATE TABLE outbox_events (
    id UUID PRIMARY KEY, topic TEXT NOT NULL, event_type TEXT NOT NULL,
    aggregate_type TEXT NOT NULL, aggregate_id TEXT NOT NULL, payload BYTEA NOT NULL,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(), attempts INTEGER NOT NULL DEFAULT 0,
    published_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX outbox_events_pending_idx ON outbox_events (available_at, created_at)
    WHERE published_at IS NULL;
```

Depois do commit:

```
dispatcher: claim (UPDATE ... FOR UPDATE SKIP LOCKED, attempts+1)
→ publica no JetStream (MsgID determinístico = event.ID)
→ somente após confirmação marca published_at
→ falha → retry com backoff (5s, 15s, 30s, 1m, 5m)
→ payload inválido = erro permanente (não retry eterno)
```

- O dispatcher converte eventos terminados em `.queued` em **jobs duráveis** direto na Queue
  JetStream (`deployments`, `backups`, `restore.cancel`, `backup.schedule`...), usando o
  `queue.Job` do payload — não depende de subscriber live.
- Workflows que usam outbox: deployment (transação única), backup manual/scheduled, cancel,
  restore. Fallback para enqueue direto só quando o outbox não está configurado (testes).

### 9.3 Recovery (idempotente, PostgreSQL-first)

Recovery reconcilia PostgreSQL e JetStream após crash, restart do NATS, perda de ACK,
redelivery ou ausência de mensagem:

- Backup jobs antigos em `preparing/running/uploading/verifying/cancelling` → `queued` +
  re-enfileirados com ID determinístico (`RecoverInterruptedBackupJobs`,
  `RecoverInterruptedRestoreJobs` — com `error_code/error_message=''` pois são NOT NULL).
- Deployments antigos em `building/starting/health_checking` → `failed`
  (`RecoverInterruptedDeployments`), sem recriar containers duplicados.
- `queued` sem mensagem no JetStream → re-enfileirado no startup (deploy: drain loop + recovery;
  backup/restore: recovery direto com query UNION por `org_id`).
- Mensagem no JetStream com estado terminal no Postgres → ACK (idempotência).
- Locks por recurso (KV JetStream, CAS/revision + TTL) impedem execução dupla entre instâncias.
- Regra: operações repetíveis voltam a `queued`; operações não repetíveis viram `failed`.

### 9.4 Restore assíncrono

A API cria `RestoreJob` em `queued` e responde **202 Accepted** (com `id`, `backup_id`,
`target_database_id`, `status`); o worker executa download, restore, verificação e transições
persistidas (`preparing → downloading → restoring → verifying → completed|failed`).
**Nenhum restore pesado ocorre em handler HTTP.** Falha de enqueue persiste
`RESTORE_ENQUEUE_FAILED`; falha por lock ocupado persiste `RESTORE_ALREADY_RUNNING`.
A rota legada (`/databases/:dbID/restore`) delega ao fluxo assíncrono e retorna 202 quando o
backend assíncrono está configurado (503 caso contrário — nunca executa restore síncrono).

---

## 10. Cache e estado efêmero

Cache **nunca** é fonte de verdade: limitações físicas (entradas + bytes), TTL real,
invalidação por evento e rebuild a partir do PostgreSQL.

| Cache | Local | Conteúdo |
|-------|-------|----------|
| `mem-lru` | core (RAM) — adapter NATS local (`NewCache`) | projeções quentes, config de UI, resultados externos, catálogo do Studio |
| `disk-object` | disco | blobs pequenos (templates, ícones) |
| `image-layer` | podman storage | camadas de imagem (dedup content-addressable) |
| `build-cache` | disco | camadas de build (quota 10 GB default + LRU) |
| `config-proxy` | memória do provider | dynamic config |

### 10.1 LRU em memória

- Implementação: LRU com **limite duplo** (`max_entries` ex.: 10.000; `max_bytes` ex.: 64 MB).
- TTL por classe: config estática 60 s; APIs externas 30–300 s; projeções quentes até
  invalidação por evento; tokens/credenciais com o mínimo seguro.
- Eviction LRU ao estourar qualquer limite; invalidação por evento (`app.updated` → remove
  `app:<id>`); invalidação explícita por chave/prefixo.
- Métricas: `aether_cache_size`, `aether_cache_hits`, `aether_cache_misses`,
  `aether_cache_errors` (expostas em `/metrics` e `/api/v1/runtime/metrics`).

### 10.2 Snapshot em disco

A cada N min (default 5) ou em shutdown gracioso, o LRU gera snapshot compacto com apenas
chaves não-expiráveis/vida longa; no boot carrega uma única leitura; corrompido → descarta e
reconstrói; `fsync` controlado.

### 10.3 Estado efêmero (KV JetStream)

- **State KV** (`AETHER_STATE`): `Set/Get/Del/Changes` com envelope
  `{value, expires_at}`; TTL real: valores expirados não são retornados e são removidos;
  `Changes` publica via Core NATS `aether.state.<key>`.
- **Locks KV** (`AETHER_LOCKS`): `Acquire` com `Create` CAS (revision), owner token por
  `revision:owner`, TTL no valor; `Renew` e `Release` verificam owner + revision; stale lock
  (expirado) é reutilizado via `Update` com revision.
- **Presence KV** (`AETHER_STATE`): `presence.<scope>_<member>` com `expires_at`; heartbeat
  renova; `Members/Count` ignoram e limpam expirados.
- **Rate limiting**: local por instância (token bucket em memória) — suficiente para instância
  única; decisão documentada (distribuído global seria via KV em multi-instância).
- **Cache distribuído**: com o produto atual em instância única, o cache é local; em cluster,
  compartilhar via KV com TTL/invalidação explícitas; nunca armazenar grandes blobs no KV.


## 11. Event bus e realtime

### 11.1 Transporte e subjects

| Camada | Subjects | Uso |
|--------|----------|-----|
| Core NATS | `notify:org:<org>` , `aether.live.<topic>` | realtime efêmero, notificações, broadcast |
| JetStream | `aether.events.<topic>` (event log), `aether.jobs.<stream>` (jobs), `aether.dlq.<stream>` (DLQ) | durável, replay, retry, schedule |

O namespace segue as permissões `publish: "aether.>"` / `subscribe: "aether.>"`. O catálogo
central de subjects vive em `api/internal/platform/messaging/subjects.go`:

```go
const (
    JobsPrefix   = "aether.jobs."
    EventsPrefix = "aether.events."
    LivePrefix   = "aether.live."
    DLQPrefix    = "aether.dlq."
    StatePrefix  = "aether.state."
    MonitoringPrefix  = "aether.monitoring."
    NotifyPrefix     = "notify:"
    MonitoringSnapshot = MonitoringPrefix + "snapshot"
)
func Jobs(stream) / Events(topic) / Live(topic) / DLQ(stream) / State(key) / NotifyOrg(orgID)
```

### 11.2 Envelope padrão

```go
type Envelope struct {
    ID            string    `json:"id"`
    Type          string    `json:"type"`
    SchemaVersion int       `json:"schema_version"`
    CorrelationID string    `json:"correlation_id,omitempty"`
    CausationID   string    `json:"causation_id,omitempty"`
    OrgID         string    `json:"org_id,omitempty"`
    ResourceID    string    `json:"resource_id,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
    Payload       []byte    `json:"payload"`
}
```

Envelope é o formato único para jobs e eventos; o decode aceita também payload legacy
(sem envelope) para compatibilidade.

### 11.3 Catálogo de eventos por domínio (real)

| Categoria | Eventos |
|-----------|---------|
| Deployments | `deploy.queued`, `deploy.building`, `deploy.starting`, `deploy.health_checking`, `deploy.ready`, `deploy.failed`, `deploy.rolled_back`, `deploy.cancelled`, `deploy.build.log` (efêmero) |
| Backups | `backup.queued`, `backup.preparing`, `backup.running`, `backup.uploading`, `backup.verifying`, `backup.completed`, `backup.failed`, `backup.cancelling`, `backup.cancelled` |
| Restores | `restore.queued/.../completed/failed` |
| App | `app.state` (efêmero), notifications |
| Monitoramento | snapshot live via `aether.monitoring.snapshot` |

Eventos não-ephemeral: gravados no event log JetStream (`aether.events.org.<org>`) com seq por
org, depois publicados no Core (`notify:org:<org>`). Efêmeros: apenas Core.

### 11.4 Fluxo realtime

```
worker persiste status → Realtime.NotifyDeploy → event log JetStream (seq por org)
→ Core NATS (notify:org) → API WebSocket hub (/api/v1/ws/realtime)
→ frontend invalida queries → toast + notification
```

- Protocolo WS: um socket por sessão; primeiro message do cliente define escopo/seq; hub
  autoriza por `org` (ou `app:<id>`/`deployment:<id>` via `Authorize` + RBAC), envia replay
  (`.queued` history + `seq` do client), depois eventos incrementais; efêmeros com seq=0 não
  voltam no replay.
- Heartbeat: servidor envia ping; cliente responde `{"op":"ping"}` a cada ~25s; read timeout
  do hub 45s; desconexão limpa presence.
- O middleware `Timeout` **nunca** é aplicado a `/api/v1/ws/**`.
- Pressence: `Join/Heartbeat` por escopo com TTL 60s (`presence.*` no KV); `Count/Members`
  expostos na API.
- Frontend reconecta com backoff; `localStorage` guarda o último `seq` por escopo.

**Polling proibido** para dados entregues por WS (deploys, app states, notifications).
Exceções aceitas: net-q 15s, presence 30s/10s, host-stats 2s, log-follow SSE, notificações
fallback 30s (offline). Dúvida → consultar o usuário antes de introduzir polling.

### 11.5 ACK / NAK / DLQ (semântica de consumo)

| Resultado do handler | Ação | Detalhe |
|----------------------|------|---------|
| Sucesso | `Ack()` | termina a mensagem |
| Falha recuperável | `NakWithDelay(delay)` | retry com backoff pelo attempt |
| Falha permanente | `Ack()`/`Term()` | persiste `failed`; payload inválido → DLQ |
| Inválido estrutural (JSON/UUID/org) | `Term()` + DLQ | `queue.PermanentError` |

- DLQ (`aether.dlq.<stream>`) carrega `dlqMessage`: job decodificado, `original_subject`,
  `attempts`, `failed_at`, `error`, `payload`, `correlation_id`, `causation_id`, `resource_id`.
- `MaxDeliver = 5`; `BackOff = [5s, 15s, 30s, 1m, 5m]`; `NakWithDelay(retryDelay(job.Attempt))`.
- Jobs longos: `queue.StartProgress` chama `InProgress()` a cada 5 minutos; `Close()` do
  consumer NAK pendências (redelivery imediata).
- Erros de Ack/Nack são logados via métrica `aether_jobs_*` (nunca descartados silenciosamente).

### 11.6 Retry policy (padrão único)

`attempt 1 → 5s | 2 → 15s | 3 → 30s | 4 → 1m | 5 → 5m`. Diferenciados:
`PermanentError` (DLQ/Term), recuperável (NakWithDelay), `context.Canceled` (shutdown grace).

---

## 12. Jobs e scheduler

### 12.1 Consumidores duráveis (pull) por stream

| Stream (`aether.jobs.<stream>`) | Consumer group | Concorrência | Trabalho |
|--------------------------------|----------------|-------------|----------|
| deployments | `workers` | 4 | deploy/cancel/rollback/recovery |
| backups | `backup-workers` | 2 | backup, backup.schedule, backup.cancel, restore |
| snapshots | `snapshot-workers` | 2 | snapshot.create |
| cron | `cron-workers` | 2 | cron.execute |

Todos com `AckExplicitPolicy`, `AckWait` (30 min), `MaxDeliver` (5), `BackOff`,
`MaxAckPending` (16), `DeliverPolicy: DeliverAll`, `ReplayPolicy: Instant`, DLQ e métricas.

- Concorrência: loops concorrentes por worker (`Concurrency` por tipo), com guards:
  - backups/restores serializados por database via locks `db:<id>:backup`/`db:<id>:restore`;
  - deployments serializados por app (`inFlight` + status guard no Postgres);
  - uma fila congestionada não bloqueia as demais (streams separados).
- Jobs rejeitados por tipo desconhecido `Ack` (permanente); payloads inválidos Nack → DLQ no
  limite (ver exceção de idempotência: `Term` direto em inválido estrutural).

### 12.2 Scheduler nativo JetStream

- Stream `AETHER_JOBS` com `AllowMsgSchedules: true` (NATS ≥ 2.14.0).
- Primitivas: `ScheduleAt`/`ScheduleJobAt` (`WithScheduleAt` + `WithScheduleTarget`),
  `ScheduleJobEvery` (`@every <dur>`), `ScheduleJobCron` (6-field + timezone via
  `WithScheduleTimeZone`).
- **Postgres guarda schedules** (fonte de verdade): `backup_configurations` (type/minute/at/
  day/start/cron/timezone + `next_run_at`), `cron_jobs` (`schedule`, `next_run`),
  `snapshot_schedules` (`cron`, `next_run`).
- **Reconciliação** (no worker, com liderança via lock `leader`): lista schedules ativos do
  Postgres → persiste `next_run` calculado → (re)cria no JetStream (cron: `ScheduleJobCron`;
  one-shot: `ScheduleAt`) → purga obsoletos do KV (`scheduler.recurring.*`).
  Se o JetStream for apagado, o Postgres reconstrói tudo.
- Chaves determinísticas: `backups:<id>`, `cron:<id>`, `snapshots:<id>` (recurring subjects
  hashados: `aether.jobs.schedules.recurring.<sha256>`).
- Updates não duplicam: purge do subject antigo + `MsgID determinístico` por chave+run.
- Execução do schedule: mensagem `backup.schedule`/`cron.execute`/`snapshot.create` é
  consumida pelo worker, que cria o job real (`enqueue` via outbox) e avança `next_run`
  (fallback: quando o scheduler não é recurrente, o próprio handler reagenda o próximo run).

### 12.3 Horizontal scaling

Durable consumers distribuem trabalho entre instâncias do worker (1..N sem mudar código);
`MaxAckPending` limita puxadas; `AckWait` + `InProgress` evitam redelivery acidental durante
jobs longos; locks por recurso garantem exclusividade cross-instance.

---

## 13. Domínios (bounded contexts)

### 13.1 Identity

- Agregados: `User`, `Organization`, `Team`, `ApiKey`, `Session`, `RoleBinding`.
- Responses: autenticação (password Argon2id; OIDC/SSO; MFA em fase 2+), autorização (RBAC por
  org; escopos de API key), sessões/JWT com rotação, convites.
- Eventos: `user.created`, `org.created`, `team.invited`, `user.role.changed`,
  `apikey.created`, `session.revoked`. Fronteira: só ele decide quem pode agir.
- Concretização: `modules/auth` (register/login/me/members/keys/TOTP/audit, SSO settings),
  sessões em tabela; org criada com slug único; rol default Owner para o fundador.

### 13.2 Projects/Environments

- Agregados: `Project`, `Environment`. Agrupam services e variáveis; scope de RBAC.
- Eventos: `project.created`, `environment.added/deleted`.
- Concretização: `modules/apps` (projects/environments/apps/env vars) + `modules/variables`
  (variáveis por projeto/ambiente, com audit/export/import e resolver de effective vars).

### 13.3 Applications

- Modelo: fonte (imagem, git, upload, compose), build (build_type, dockerfile, comandos,
  root/install/build/start, dist folder), porta/recursos (cpu, mem, storage), env vars/secrets,
  health checks, domains, previews, image retention.
- Eventos: `app.created/updated/removed`, `envvar.changed`, `healthcheck.updated`.
- Concretização: `modules/apps` + `modules/specs` (análise de fonte, planner, compare).

### 13.4 Deployments

- Pipeline: saga build → schedule → health → promote; rollback sob demanda; previews por
  PR/branch; webhooks de Git/registry viram comandos; timeline/estado completo persistido.
- Eventos: todos os `deploy.*`. Concretização: `modules/deployments` + `platform/worker`.

### 13.5 Runtime / Execution Engine

- Runtime representa estado desejado: `RuntimeHandle`, `ImageRef`, `ContainerSpec`,
  `ComposeSpec`, `DriverSpec`; não conhece Podman (usa a porta `RuntimePort`).
- Concretização: `platform/worker/runtime.go` (podman CLI via os/exec; container info/stats/
  logs/exec/health), `modules/snapshots/infra/podman.go` (snapshot de volume com tar da VM),
  `modules/databases/application/lifecycle.go` (provision de DBs por engine).

### 13.6 Build

- Fila, concorrência limitada, janelas; cache/quota/GC; registro local. Eventos:
  `build.started/finished/failed`, `build.cache_pruned`. Concretização: `worker.go` (CNB/
  Dockerfile/custom), `planner`, `builders/*` (builder e buildpacks).

### 13.7 Networking / Certificates

- Já detalhados nas seções 7 e 8. Eventos: `domain.added/updated`, `route.updated`,
  `middleware.changed`, `proxy.reloaded`; `cert.issued/renewing/renewed/renewal_failed/revoked`.
- Concretização: `modules/domains` (provision, worker de provisionamento, traefik dynamic
  config em shared volume).

### 13.8 Storage

- `Volume`, `VolumeClaim`, `Snapshot`, `BlobRef`, quotas; blobs locais/S3; snapshots de
  volume por app. Eventos: `volume.created/resized/snapshot/removed`.
- Concretização: `modules/volumes` (listar/backup volume para destinos), snapshots via podman.

### 13.9 Backups / Restores

- BackupJob (queued→preparing→running→uploading→verifying→completed/failed/cancelled),
  configurações com schedule (hourly/daily/weekly/biweekly/custom + timezone), retenção
  (latest/all), destinos S3 (aws/cloudflare-r2/minio/custom) e Google Drive OAuth.
- Streamed + checksummed (sha256) + verificado após upload (HeadObject size) + aplica retenção.
- Restore assíncrono com download→verify (checksum/size)→restore→completed.
- Eventos `backup.*`/`restore.*`. Concretização: `modules/backups` (service/scheduler/
  worker/executor/config/http/infra + adapters de engine postgres/mysql/mariadb/mongo/mssql/
  oracle) e `platform/storage` (S3 via minio-go; GDrive com OAuth e tokens cifrados).

### 13.10 Logs e Metrics

- Logs: streaming SSE/WS, rotação, retenção, per-app diretório. Metrics: coleta sob demanda,
  agregação, exposição. Eventos: `log.rotated`, `metrics.subscribed/aggregated`.
- Concretização: `modules/host` (host stats via agent), `modules/stats`,
  `modules/monitoring` (collector batch, classifier por labels, History com persistência e
  ring buffer, recursos com storage via `podman system df -v`, CollectorStats).

### 13.11 Observability

- Alertas avaliados por evento; timeline = eventos por aggregate; audit append-only.
- Concretização: `modules/alerts` (rules/events/channels/notifications).

### 13.12 Git / Source Control

- GitHub App OAuth + webhooks; sinaturas HMAC; auto-deploy; repos/branches/files; imports de
- template. Concretização: `modules/sourcecontrol` + `platform/scm/github`.

### 13.13 Marketplace / Templates

- Catálogo de templates, categorias, featured/verified/installs, trending, instalação real
  (gera stack compose com volumes/env/secrets). Concretização: `modules/templates` (catalog +
  compose wizard/validator/editor/export/import/compose stack run).

### 13.14 Automation

- Cron jobs (rota CRUD + execução via scheduler), workers one-shot (start/stop via podman),
  autopilot (policy de scaling por métricas), pipelines, mirrors.
- Concretização: `modules/jobs` (cron/workers/policy), `modules/pipelines`,
  `modules/mirrors`.

### 13.15 Infrastructure

- Servers/clusters/agents; primitivas de nó; registries; images (podman images/rmi).
- Concretização: `modules/clusters`, `modules/registry`, `host` (agent/watchdog).

### 13.16 Plugins / Audit / Identity interações

- Plugins: portas (Git, DNS, Cloud, ObjectStore, Notify, Idp, Ai, Mcp, Exporter, Acme,
  Proxy, Runtime Driver, SecretStore) — ver seção 19.
- Audit: `audit_events` append-only; audit de variáveis; snapshot do estado.

### 13.17 Databases gerenciadas (produto)

Engines: PostgreSQL, MySQL, MariaDB, Redis, MongoDB, SQL Server, Oracle. Containers OCI com
volume próprio, credenciais cifradas (password cipher AES-GCM derivada da master key), DSN
interno (ex.: `postgres://user:pass@host:port/db`), portas publicadas, logs/stats, backup/dump
por engine (pg_dump, mysqldump, mongodump, redis save...) e restore. Acessível via Studio
(rota fullscreen `/studio/<dbId>`: query editor, object explorer, grid, create table, context
menu, metadata introspect via adapters) e via terminal/DBTS. **Redis aqui é banco de usuário** —
nunca infra do Aether.

---

## 14. Observabilidade

### 14.1 Logs

- Estrutura: JSON compacto `{ts, level, src, msg, fields}`; source por processo/componente.
- Rotação por tamanho (10 MB) e tempo (diária); compressão zstd; retenção 14 dias ou 100 MB
  por serviço (configurável); journald com `SystemMaxUse` limitado.
- Streaming: via SSE/WS sem acumular memória; seguir logs de deploy (file `deployments/<id>.log`
  + live) com tail/follow; terminal via WebSocket (pty).
- Nunca conter secrets; scrubbing no sink.

### 14.2 Métricas (formato Prometheus-like)

- Por processo: `/health`, `/ready`, `/metrics` (worker :8081, monitoring :8082; API via
  `/api/v1/runtime/metrics`).
- Métricas principais: `aether_jobs_active`, `aether_jobs_completed_total`,
  `aether_jobs_failed_total`, `aether_jobs_duration_seconds_total` (global e por `type`),
  `aether_scheduler_reconciliation_errors_total`, `aether_nats_publish_errors_total`,
  `aether_monitoring_collections_total/_errors_total/_duration_seconds_total`,
  fila JetStream: pending, ack_pending, redeliveries, dead_letter por stream
  (`/api/v1/runtime/metrics` → `queues`), cache hits/misses/sets/errors, subscribers.

### 14.3 Tracing

- Leve e amostrado (default 10% produção); propagação `X-Aether-Trace` + correlation id nas
  mensagens; export OTLP opcional via plugin; default sem export (zero custo).

### 14.4 Alertas

- Avaliação por evento (não polling): `service.crashed` ≥3 em 10 min; `cert.renewal_failed` ≥3;
  `deployment.failed`; disk >85% (métrica sob demanda); health check falhou.
- Canais: e-mail, Slack, Discord, Telegram, webhook (via plugin NotifyChannel);
  estados fired→resolved; deduplicação por regra+recurso+janela.
- Reconciliação de baixa frequência (60 s) apenas quando há regra ativa.

### 14.5 Timeline e Audit

- Timeline por recurso = eventos do bus filtrados por aggregate (custo zero).
- Audit: `audit_events`/`audit_logs` (quem, o quê, recurso, IP, quando); nunca UPDATE/DELETE
  via API; inclui login/logout, RBAC, deploys manuais, restores, configurações, variáveis
  (audit de changes).

---

## 15. Segurança

### 15.1 Modelo de confiança

```
[Internet] → (TLS) → [Proxy rootless] → (rede interna) → [App containers rootless]
                         └→ [Core] → [PostgreSQL] → [Worker/Monitoring]
Comunicação core↔agent: mTLS (CA interna). Segredos criptografados; master key protegida.
```

### 15.2 Rootless + least privilege

- Core/agent/proxy como usuários de sistema sem shell; apps rootless (podman).
- `NoNewPrivileges=yes`; capabilities bounding set vazio (proxy: CAP_NET_BIND_SERVICE via
  systemd); seccomp default-deny; AppArmor/SELinux quando disponível; `0600` chaves,
  `0700` diretórios; namespaces por app; sem daemon privilegiado.

### 15.3 Secrets

```
KEK (master key, host) → criptografa DEK → criptografa secrets (AES-256-GCM, AEAD)
```

- DEK rotacionável sem re-criptografar tudo; `secret_ref` referenciado (nunca o valor);
  injeção feita pelo worker com permissões restritas; nunca em logs (scrub em sinks).
- Master key em `~/.aether/keys/master.key` (0600) gerada pelo installer; env vars marcadas
  `IsSecret` cifradas; senhas de DB cifradas com cipher próprio; tokens Google OAuth cifrados.
- Secret store plugável (Vault/KMS futuro).

### 15.4 Auth, RBAC, API keys

- Login local: Argon2id, JWT curto + refresh rotation, rate limit de login (2 req/5s default) +
  lockout; logout invalida sessão; cookie `aether_token` (secure flag configurável).
- OIDC/SSO (via settings/IdpProvider), MFA TOTP (enroll/verify/disable) e SAML em fases.
- RBAC: **User → Organization → Role → Permissions** (Owner/Admin/Developer/Viewer/Custom;
  permissões granulares `app.deploy`, `secrets.read`, `backup.create`, `cert.manage`).
  Enforced no core; API sempre verifica (defense in depth); middleware define
  `ContextOrgID` (da sessão, não de header; header `X-Aether-Org` apenas refine).
- API keys: alta entropia (`sk_...`), hasheadas (Argon2id), exibidas uma vez, escopos,
  expiração, rotação, revogação imediata, audit de uso.

### 15.5 NATS

Autenticação por user/pass; `infra/nats.conf` com permissions `aether.>` (publish/subscribe);
credenciais geradas pelo installer persistidas em `~/.aether/keys/nats.auth`; propagadas para
API/worker/monitoring; validação de versão mínima no bootstrap (≥ 2.14.0); JetStream em disco
(volume `aether-nats-data`); connections com `Drain()` no shutdown.

### 15.6 Rede e hardening de sistema

- Proxy único ponto de entrada (80/443). UI/API em loopback ou via domínio com auth; nunca
  exposta sem TLS por padrão. sysctl hardening documentado; seccomp/AppArmor/SELinux
  aplicados quando disponíveis; validação de hosts de desafio ACME.

### 15.7 Supply chain

Binários e plugins assinados (cosign/sigstore); checksum publicado; builds reproduzíveis;
SBOM por release; scans de vulnerabilidade em CI; manifestos de plugin com assinatura;
permissões mínimas por declaração.

### 15.8 Response a incidentes

Alertas de segurança por evento (ex.: muitas falhas de login); revogação instantânea de tokens/
keys; rotação de KEK/DEK documentada; backup de estado como ponto de restauração forense.

## 16. Installer e operação

### 16.1 Modelo

Dois instaladores:

- **`install.sh`** (Linux/produção, executável por root ou sudo): clona o repo em
  `$INSTALL_DIR` (`/opt/aether` ou `~/.local/share/aether`), valida o checkout, exporta
  `AETHER_MODE=prod` e delega ao `install-dev.sh`. Comandos:
  `install|update|start|stop|status|logs`.
- **`install-dev.sh`** (macOS/VM podman e Linux): orquestra containers. Fases: config
  (state dir, rede `aether-net`), PostgreSQL (`aether-postgres`, 15432 host/5432 interno),
  NATS JetStream com auth (`aether-nats`, 4222/8222, volume persistente),
  registry (`aether-registry`, 5000), builder CNB (`ensure_builder`: ubuntu run image +
  buildpacks aether, com stamp sha256), images de API/worker/monitoring/web, ingress
  Traefik, agent/watchdog de host, migrações, seeds, health checks e nginx gateway (4000).

### 16.2 Comandos

- `./install-dev.sh start` — sobe tudo (build da imagem, registry, builder, postgres, nats,
  api:8080, worker:8081, monitoring:8082, web:4000, traefik, agent).
- `./install-dev.sh stop` / `status` — para/status por nome específico.
- `./install.sh install|update|start|stop|status|logs` — delega para o dev com `prod`.
- `./dev.sh` — API em dev no host (`air`, 127.0.0.1:8090, hot reload); carrega `.env`
  local + `.aether-db`; faz forward do socket podman (preferência: socket nativo do gvproxy
  no Mac; fallback `ssh -L` validado com retries); exporta credenciais NATS do
  `~/.aether/keys/nats.auth`; roda migrações e o air.
- `./dev-web.sh` — Vite (5173) com proxy `/api` → 8090.

### 16.3 Regras operacionais duras

- **Nunca** `DROP/CREATE/TRUNCATE/DELETE` em dados reais; escrita só com autorização explícita.
- **Nunca** `podman rm -f $(podman ps -aq)` nem filtros genéricos `--filter name=aether-`
  (pega infraestrutura) — operar containers **por nome específico**.
- Containers de teste (`aether-test-pg` 5433, NATs de teste 4223–4230, `aether-redis-test 6380`)
  NÃO devem ser derrubados; a suíte depende deles.
- Após mudanças de backend: `podman build -t aether.local/api:1 -f infra/Dockerfile .` +
  `./install-dev.sh start` (rebuild + restart); dockerfiles são multi-arch (ARG TARGETARCH
  para o binário pack: arm64 vs amd64).
- Podman machine 4 GB: cuidado com OOM em builds pesados; não rodar builds pesados em paralelo.
- Estado em `~/.aether` (certificados, keys, logs, builds, uploads, snapshots, traefik volume).

### 16.4 Update / rollback / uninstall

- Update: binário novo + migrações transacionais + rollback automático em falha; autoupdate
  desligável (P8). Rollback: binário anterior + migração reversa por versão.
- Uninstall: remove units/usuários/sockets; `--purge` remove dados; units de apps do usuário
  PARADAS mas NÃO removidas (migração segura).

---

## 17. Service Management (RFC-0030/0033)

### 17.1 Fluxo unificado

Application, Database, Worker, Cron e Compose são Services:

```
criação → escolher tipo → coletar mínimo → expandir configurações → validar
→ persistir estado desejado → provisioning/deploy assíncrono → progresso/erro/retry/cancel
```

- **Overlay Manager**: o wizard usa overlay por passos (dados parciais + mesclagem) para evitar
  estados parcialmente criados; confirmação antes do commit.
- **Deploy spec-first**: `Deployment.deploy_spec` carrega o spec completo (imagem/git/comandos/
  health check) como fonte única do build/run; `env_snapshot` congela variáveis no momento do
  deploy.
- Compose: wizard com parser YAML real (services/volumes/networks, erros/warnings,
  dependências/portas), editor numerado + highlight, validação ao vivo + grafo, export/import,
  deploy como stack (labels aether), `compose_hash` para skip quando nada mudou.
- App states: agregados via SSE para a UI; status pill + timeline.

---

## 18. API, CLI e plugins

### 18.1 API REST (`/api/v1`)

- JSON, OpenAPI-like types em `web/src/api/types.ts`; idempotência (`Idempotency-Key`);
  operações longas → `202` + recurso de progresso; paginação; ETag/If-Match.
- Auth: `cookie aether_token` (sessão) e `Authorization: Bearer` (sessão/API key);
  `X-Aether-Org` para scope de org; RBAC no core.
- Recursos (seleção real): `auth/*` (register/login/me/logout/members/keys/audit/totp),
  `organizations/*`, `projects/*` (+env/variables/audit/export/import/default),
  `apps` (+deploys/deploy/rollback/cancel/logs/timeline/states/compare/volumes/variables/
  source/domain/compose/policy), `databases` (+studio/catalog/terminal/queries),
  `backups`/`databases/:id/backup*` (configurations, jobs, restore preflight/202, cancel,
  restore-jobs), `snapshots` (+schedules), `domains`/`certificates`, `cron-jobs`, `workers`,
  `templates`/`marketplace`, `gitops`, `alerts`/`notifications`, `s3-destinations`,
  `webhooks`, `mirrors`, `services`/`volumes`, `registry/images`, `host`, `specs/analyze`,
  `stats`, `realtime/events` (+stream/presence/network-quality/runtime-metrics),
  `monitoring`, `source-control`, `settings` (SSO/OIDC/branding), `compose` (wizard/validate/
  deploy/export), `pipelines`, `clusters`.
- Streams: SSE `/events/stream`; WebSocket `/api/v1/ws/realtime` (hub por org, seq/replay,
  ping/pong); terminal WebSocket em `/databases/:id/terminal`, log-follow SSE.
- Erros: `{error, code}` com status HTTP coerentes; `mapErr` traduz PgError (23505→conflict,
  22P02→validation, 23503→conflict).

### 18.2 CLI (esperado/parcial)

`login|logout|whoami`; `apps list|create|deploy|logs|rollback|rm`; `projects`;
`servers add|list`; `backups create|restore|list`; `cron jobs`; `api-keys`;
`update|rollback|uninstall|status`; saída JSON (`--json`) para CI; import/export YAML
(`aether.yml`) com dry-run e validação de schema — caminho de migração entre plataformas
(migração zero-copy de compose + .env de Coolify/Dokploy com fixture testada).

### 18.3 Plugins

- Portas: GitProvider, DnsProvider, CloudProvider, ObjectStore, NotifyChannel, IdpProvider,
  AiProvider, McpProvider, MonitoringExporter, AcmeProvider, ProxyProvider, RuntimeDriver,
  SecretStore.
- Tipos: native (compilado), bundled (entregue, ativado sob demanda), external (assinado,
  subprocess sandbox com `NoNewPrivileges`, seccomp default-deny, rlimits, namespaces,
  timeout/retry).
- Manifesto `{id, version, runtime, port, permissions, signature}`; carregamento sob demanda;
  nada roda em idle; core nunca depende de plugin para funções essenciais (circuit breaker);
  audit de instalação/ativação/permissões.
- MCP: plugins que expõem ferramentas para LLMs (assistente consulta deployments etc).

---

## 19. Testes e gates

Backend (comandos oficiais):

```bash
AETHER_TEST_DATABASE_PORT=5433 AETHER_API_TEST_DATABASE_PORT=5433 \
AETHER_TEST_DATABASE_USER=postgres AETHER_TEST_DATABASE_PASSWORD=postgres \
AETHER_NATS_TEST_URL=nats://127.0.0.1:4228 \
go test ./api/internal/... -count=1 -p 1 -timeout 25m

go build -o /tmp/aether-api ./api/cmd/api
go build -o /tmp/aether-worker ./api/cmd/worker
go build -o /tmp/aether-monitoring ./api/cmd/monitoring
go vet ./api/internal/...
```

Frontend:

```bash
cd frontend/web && npm run typecheck && npm run build
```

Cobertura de testes existente:

- **Unit**: serialização de payload/envelope, classificação de erros (PermanentError),
  geração de subjects, decodificação legacy, retry delays, transições de estado, TTL do state,
  webhooks/providers, specs/analyze, cache, locks, scheduler primitives, rate limit.
- **Integração (NATS real + Postgres real)**: publish/consume/ack/nak/retry, redelivery após
  crash (worker A sem ACK → worker B), DLQ, scheduled job `@every`/cron, recurring reconcile
  (update/remoção sem duplicação; recriação após perda de estado), locks KV CAS, presence,
  event log NATS (append/recent/replay por org), outbox (claim/retry/markPublished, publicação
  direta `.queued` na Queue), auth NATS (credenciais válidas/rejeitadas), worker deploy de
  container real, monitoring collector, scheduler reconciler.
- **E2E real** (`api/internal/e2e/phase8_test.go`): HTTP de ponta a ponta — register/login,
  project, app, deploy via API, persistência Postgres, JetStream metrics, eventos realtime,
  worker processa até `ready`, health worker/monitoring; cenário destrutivo
  (`AETHER_E2E_RESTART=1`): restart de worker e NATS com reconexão.
- **Gates**: suíte completa passa; **teste skipped nunca é reportado como aprovado**;
  regressões >10% de recursos bloqueiam merge.

---

## 20. Roadmap e status

### Fase 0 — Fundação ✅
Runtime OCI, PostgreSQL (único banco), API, web, deploy, domains, TLS/certs,
observabilidade, instalador, migrações e seeds.

### Fase 1 — MVP / Fase 2 — Paridade Coolify/Dokploy ✅
Databases gerenciadas (provision→ready→backup/restore), backups/restores assíncronos,
snapshots, cron, workers one-shot, marketplace/templates (instalação real), Git/source-control,
previews, notifications, RBAC/organizations, compose wizard, API keys, registry, terminal/SSE,
Studio de databases (editor, object explorer, CRUD).

### Fase 3 — Eventos e infra duráveis ✅
Migração completa de Redis (runtime interno) para **Core NATS + JetStream** (com fase de
auditoria, remoção de adapters Redis, docs atualizadas, testes de integração em NATS real).
Componentes: outbox transacional; durable consumers com ACK/NAK/DLQ; scheduler nativo com
reconciliação; KV (locks CAS/presence TTL/estado com TTL real); event log com replay;
recovery idempotente; separação api/worker/monitoring; health + métricas por processo;
segurança NATS; E2E real.

### Fase 4 — Evolução (em andamento)
- Multi-server/HA (agentes, clusters, Patroni) e drivers adicionais (Docker/containerd/K8s).
- Plugins external + marketplace de plugins.
- Previews deployments completos; pipeline CI integrado.
- CLI definitiva; export/import aprimorado; observabilidade de DLQ operacional.

### Fase 5 — Enterprise
PostgreSQL HA (Patroni), MFA (WebAuthn), SAML, audit avançado, multi-region, quotas por org.

### Critérios de corte (gates por fase)
Metas de recursos verificadas em CI; E2E mínimo instalado→deploy→update→rollback→restore com
cronômetros; sem regressões funcionais (deployments, logs, status, backups manuais/agendados,
restore, snapshots, cron, presença, realtime, rate limit, locks, monitoring).

### Orçamento de risco
Podman/multi-distro (matrix CI); portas 80/443 ocupadas (detectar e orientar); instalação
interrompida (idempotência); update corrompido (backup automático + rollback); SELinux
(policy custom); performance rootless de rede (tuning/documentado).

---

## 21. Comparativo com concorrentes (matriz D1–D13 + aprendizados)

### 21.1 Resumo executivo da análise

Coolify (Laravel/Filament/Octane, Docker, Redis, Traefik, Monitor, Meilisearch) e Dokploy
(NestJS/tRPC/Prisma/BetterAuth, Docker compose, TRPC queues com BullMQ via Redis) foram
analisados por engenharia reversa. Ambos pagam alto custo de recursos por plano de controle em
containers ("só mais uma imagem para o painel"), daemon central, Redis para filas/cache,
workers residentes, build in-cluster, crons fixos de health check, watchtower e atualização
via "compose up".

### 21.2 Matriz de mitigação D1–D13

| Decisão do concorrente | Solução Aether | Estado |
|------------------------|----------------|--------|
| D1 Plano de controle em containers | processos host (binário único) | ✅ |
| D2 Docker Engine central | Podman rootless + crun + Quadlet | ✅ |
| D3 Redis para filas/cache | outbox Postgres + fila JetStream + cache local limitado | ✅ (migração completa) |
| D4 Traefik central em container | proxy como processo gerenciado pelo Networking Engine | ✅ container ingress |
| D5 Monitor Prometheus | observabilidade embutida e leve; sem subscriber = sem coleta | ✅ |
| D6 Queue workers residentes | workers consumindo JetStream (pull), concorrência por tipo | ✅ |
| D7 Build in-cluster buildkit | build rootless CNB local + fila + GC rigoroso | ✅ |
| D8 Crons fixos de health check | avaliação por evento/health check sob demanda | ✅ |
| D9 Watchtower | updates dirigidos por evento/deploy | ✅ |
| D10 MinIO/backup local | destinos S3 (aws/r2/minio/custom) + Google Drive; backup streamed/checksummed | ✅ |
| D11 Meilisearch | busca mínima (índices) sem serviço extra | ✅ |
| D12 Atualização "compose up" | binário + migração transacional | ✅ |
| D13 Estado Postgres+Redis+filas | Postgres = verdade + evento log/outbox + idempotência | ✅ |

### 21.3 Pontos fortes dos concorrentes a preservar

- UX de produto: wizard de deploy claro, painel por recurso, logs/timeline inline, terminal
  integrado, "one click install" de apps.
- Documentação atualizada quando a feature muda (aqui: chaves são o que o produto usa).

### 21.4 Fraquezas a evitar (registradas na análise)

- "kill = pkill -2 -f docker build / docker compose" (mata builds de outros tenants);
- assinatura/acme frágil dependente do provider;
- env secrets gravados em texto em artifacts de rollback;
- compose import apenas validando e retornando OK sem criar (paridade corrente: validar e
  informar ao usuário; criação via wizard);
- commands de build com shell injection (sanitize/quote) — aplicado no Aether.

### 21.5 Diferenças arquiteturais garantidoras da vantagem

1. **Topologia**: containers do Aether = app workloads + 4 serviços (pg, nats, registry,
   traefik) — sem stack pesada por tenant.
2. **Um binário**: Go compilado com pipeline hexagonal, sqlc, adapters NATS — sem runtime de
   linguagem de terceiros.
3. **Testes reais**: testpool com Postgres real (5433) e NATS real — superior a mocks.
4. **Event bus governado**: outbox + Journal + replay + idempotência — recovery determinístico.

### 21.6 Aprendizados de UX a incorporar

Migração guiada (import de compose existente), templates com readme pré-instalação, painél de
warnings no wizard, terminal com ctrl+C entrega à aba, botões de rollback em todos os status
não-termiais, timelines com agrupamento por deploy.

---

## 22. Regra de manutenção e decisões proibidas

Qualquer mudança arquitetural atualiza: (1) este documento, (2) a RFC especializada,
(3) a documentação operacional afetada, (4) os testes relevantes. Proibições permanentes:

- **Não reintroduzir Redis como transporte interno** (permanece como database de usuário).
- **Não transformar NATS em source of truth** (PostgreSQL é a verdade; NATS é transporte +
  coordenação + execução assíncrona).
- **Não declarar validação como concluída sem execução real.**
- **Não adicionar polling** para dados entregues por WebSocket.
- **Não adicionar comentários** em código Go/web (apenas docblocks de contratos públicos).
- **Inglês obrigatório** em código, logs, mensagens e textos da UI.
- **Não alterar design system** (estilos, ui components, shell) sem pedido explícito.

## 23. Mapeamento do código (guia de navegação real do repositório)

### 23.1 Backend — entrypoints

| Arquivo | Executável | Responsabilidade |
|---------|-----------|------------------|
| `api/cmd/api/main.go` | aether-api | carrega config, abre pool, migra (flag `-migrate`), resolve master key, `bootstrap.Run` |
| `api/cmd/worker/main.go` | aether-worker | pool, health server (:8081), `bootstrap.RunWorker` |
| `api/cmd/monitoring/main.go` | aether-monitoring | pool, publisher NATS, collector (loop 2s), health server (:8082) |

### 23.2 Bootstrap

- `api/internal/platform/bootstrap/bootstrap.go` — wiring da API: auth, apps, deployments,
  domains, jobs, databases+studio, backups, templates/compose, gitops, alerts/notifications,
  snapshots, clusters, pipelines, settings (S3/OIDC), webhooks, mirrors, volumes, orgs,
  variables, host, specs, stats, realtime (hub WS), monitoring (reader), source-control;
  readiness (pool + NATS), serve frontend, http.Server, shutdown com drain.
- `api/internal/platform/bootstrap/worker.go` — wiring do worker: stores, runtime podman,
  druntime NATS (shared conn), event log, realtime notifier, deploy worker (concorrência 4 +
  notifier/log notifier), watcher, provision worker, backups (outbox/queue/scheduler/locks),
  cron worker (2), snapshot worker (2), recovery de interrupts (deploy -30m, backups -90m),
  outbox dispatcher, scheduler global (liderança), health watch loop, ready.
- Auxiliares: `ensureIngress` (rede/ingress), `settingsDestProvider` (S3/Drive provider por
  destination), `auditRecorder`, `webhookDeployer`.

### 23.3 Módulos (cada um com domain/application/http/infra)

| Módulo | Arquivos | Destaques |
|--------|----------|-----------|
| auth | `modules/auth/*` | register/login/me/session/keys/TOTP/audit/SSO; org slug único; middleware `ContextOrgID` |
| apps | `modules/apps/*` | projects, environments, apps, env vars, secrets cipher, app states, latest deployments |
| deployments | `modules/deployments/*` | application (Deploy/Rollback/Cancel/Transition/Atomic+Outbox), infra (store + outbox), http (deploy/cancel/list/logs/compare), domain (máquina de estados) |
| platform/worker | `platform/worker/*` | build (CNB/Dockerfile/custom), run, health check, watcher, recovery, runtime podman |
| databases | `modules/databases/*` | provision por engine (containers + DSN), Studio (catalog/query/exec), terminal, restore |
| backups | `modules/backups/*` | DB backups (config/jobs/restore/cancel), engine adapters (pg/mysql/mariadb/mongo/mssql/oracle), scheduler, executor (stream/checksum/upload/verify), worker, HTTP |
| snapshots | `modules/snapshots/*` | snapshot de volume via podman (tar), schedules, worker, infra |
| jobs | `modules/jobs/*` | cron jobs CRUD, workers (containers one-shot), autopilot policy, cron worker/scheduler |
| monitoring | `modules/monitoring/*` | collector (batch podman ps + stats --no-stream), classifier por labels, aggregate Aether/User/System, history (ring + PG), storage `podman system df -v`, publisher/reader NATS, HTTP (overview/resources/history/stream/collector) |
| realtime | `modules/realtime/*` | app service (NotifyDeploy/Log/Backup/AppState, PublishEvent, SubscribeEvents, Authorize, presence, metrics, network quality), infra hub WS + NATSEventLog |
| settings | `modules/settings/*` | S3 destinations (aws/r2/minio/custom/google-drive), OAuth Google (state TTL, tokens cifrados, refresh/reauth), OIDC/SSO, branding |
| storage | `platform/storage/*` | contrato Provider (Put/Get/Head/Delete/List/Copy, Capabilities), S3 (minio-go v7), Google Drive (OAuth, upload/download) |
| volumes | `modules/volumes/*` | list volumes de apps, backup de volume para destino |
| alerts | `modules/alerts/*` | rules, events (fire/resolve), channels, notifications (Create/List/Count) |
| orgs | `modules/orgs/*` | orgs CRUD, assign project/member |
| variables | `modules/variables/*` | projeto+ambiente, audit, export/import, resolver Effective |
| templates | `modules/templates/*` | catálogo/marketplace, templates category/featured/trending, install, compose CRUD/export/validate/deploy (stack run) |
| gitops/clusters/pipelines/mirrors/webhooks/sourcecontrol | respectivos | integrações |
| host | `modules/host/*` | host stats (agent file), events, net quality, logs |
| specs | `modules/specs/*` | análise de fonte (analyze/plan/preview), planner (detect framework), compare |
| stats | `modules/stats/*` | stats de apps/DBs via runtime |
| monitoring http | `modules/monitoring/http/*` | overview/resources/history/resource-history/collector/stream |

### 23.4 druntime (adapter NATS) — detalhe de arquivos

| Arquivo | Papel |
|---------|-------|
| `nats.go` | Runtime: connection (auth + versão ≥2.14), streams (AETHER_JOBS workqueue + schedules; AETHER_EVENTS limits; AETHER_DLQ limits), KV buckets (AETHER_STATE, AETHER_LOCKS), Scheduler (at/every/cron + recurring + reconcile) |
| `queue.go` | Job Envelope; `NewConsumer` (AckExplicit, AckWait 30m, MaxDeliver 5, BackOff, MaxAckPending 16, DeliverAll); consumer Next/Ack/Nack/InProgress/Close; DLQ publish com metadata; `QueueMetrics` (pending/ack_pending/redeliveries/DLQ) |
| `events.go` | EventBus Publish (JetStream `aether.events.<topic>` + Core `aether.live.<topic>`; MsgID = event.ID) e Subscribe (Core live) |
| `pubsub.go` | PubSub Publish/Subscribe/Subscribers (Core NATS; counters) |
| `state.go` | State Set/Get/Del/Changes com envelope `{value, expires_at}` (TTL real) + notificação Core |
| `locks.go` | LockManager Acquire (Create CAS + stale takeover via Update/Revision), Renew, Release, Locked — TTL + owner |
| `presence.go` | Presença via KV com expires_at; Members/Count limpam expirados |
| `cache.go` | Cache local LRU (Get/Set/Add/Del/Invalidate, GetJSON/SetJSON, metrics) |
| `ratelimit.go` | Rate limiter local (token bucket) |
| `memory/*` | Mesmo contrato em processo (para testes) |

### 23.5 Frontend — estrutura

- `frontend/web/src/routes/` — rotas TanStack (file-based): `_shell/*` (apps, databases,
  backups, projects, settings, monitoring, etc.), `studio.$dbId` (fullscreen, fora do `_shell`),
  login/register.
- `frontend/web/src/stores/` — Zustand: `auth`, `org`, `realtime`, `notifications`, `overlay`,
  `palette` (estado global; providers só para lógica).
- `frontend/web/src/hooks/` — um hook por arquivo (kebab-case), `use-*.ts` + index barrel;
  TanStack Query; types em `api/types.ts`; client axios com `withCredentials` + `X-Aether-Org`.
- `frontend/web/src/components/` — `ui/*` (Button, Card, Input, Select, Field, Dialog,
  ConfirmDialog, StatusPill, Table, Popover, Spinner, EmptyState, CodeBlock, useToast,
  CardMenu, MetricCard...), `shell.tsx` (sidebar 256px + collapse), providers
  (NotificationProvider, RealtimeProvider, OrgProvider).
- `frontend/web/src/studio-intelligence/` — engine de SQL Intelligence: schema snapshot
  (IndexedDB versionado, TTL 7d), context extraction quote-aware, graph de relacionamentos
  (FK + inferência), ranker determinístico com reasons, learning (frequência×recency,
  join-pairs), provider Monaco (autocomplete/hover).
- Design system: `frontend/aether_ds` (`@aether/design-system`), tokens em
  `frontend/web/src/styles.css` (tema dark, `--radius-DEFAULT .5rem`, Inter) — **não alterar**
  sem pedido explícito.

### 23.6 Infra

- `infra/Dockerfile` — multi-arch: build de api/worker/monitoring (Go 1.26) + pack (arch),
  runtime alpine com podman/pack/git; EXPOSE 8080/8081/8082.
- `infra/web.Dockerfile` — build do frontend (aether_ds + web) + serve estático.
- `infra/docker-compose.yml` — postgres (aether), nats (auth via env `:?` + nats.conf com
  permissions), aether API/worker/monitoring com healthchecks (ready), web gateway.
- `infra/nats.conf` — auth + permissions `aether.>`.
- `infra/buildpacks/*` + `infra/builders/*` — buildpacks aether e script do builder
  (`builders/build-builder.sh` NOVO em `infra/buildpacks/builders/`; o antigo `infra/builders/`
  foi removido — sem Paketo).
- `infra/scripts/*` — host-agent.sh e host-watchdog.sh (métricas reais do host acopladas ao
  ciclo da API), cnb-e2e.sh (matriz de buildpacks).
- `install-dev.sh` / `install.sh` / `dev.sh` / `dev-web.sh` — ciclo de vida (seções 16).

---

## 24. Fluxos de domínio de ponta a ponta (detalhados)

### 24.1 Deploy de app (imagem ou git)

1. UI/API: `POST /api/v1/projects/:id/apps` (create app) → `POST /apps/:id/deploy`.
2. App service valida (org, RBAC), snapshota env, monta `deploy_spec`.
3. `Deployments.Deploy`: `NextNumber` → `CreateDeploymentAndOutbox`:
   transação única (deployment row + outbox `deployment.queued` cujo payload é o `queue.Job`
   com `ID=deploymentID`).
4. Commit. Resposta `202` com deployment.
5. Worker: outbox dispatcher claim → publica `aether.jobs.deployments` (MsgID = job.ID).
6. Deploy worker consome: guard `status==queued` (senão Ack, idempotente), `building` → build
   (CNB/Dockerfile/custom) → `starting` (run podman com labels aether, rede ingress, alias,
   mem/cpus) → `health_checking` (se habilitado; timeout/retries) → `ready` ou `failed`.
7. Em cada status: `NotifyDeploy` → event log JetStream (seq) → Core `notify:org` → WS hub →
   UI invalida queries; notification para status notificáveis.
8. `finished_at`/`error` persistidos; container antigo removido só após sucesso.

### 24.2 Backup manual

`POST /databases/:id/backups` → `StartManualBackup`: valida db/config única; `ListActiveJobs`
(conflito se ativo); `CreateJob(queued)`; `enqueue` via outbox (`backup.queued`) ou direto
(fallback). Worker: `runBackup`: `preparing` → busca config/engine/credenciais → lock
`db:<id>:backup` → `running` (adapter dump streamed para arquivo + sha256) → `uploading`
(storage put, metadata database_id/backup_id/engine/format/checksum) → `verifying`
(HeadObject size == size) → `completed`; qualquer falha → `failJob(code,msg)`; retenção
`latest` remove anteriores; notify `backup.*` em cada transição; ACK do job.

### 24.3 Restore

`POST /databases/:id/backups/:backupID/restore` → preflight (engine/status/ativo) →
`RequestRestore`: cria `RestoreJob(queued)`, envia via outbox (`restore.queued`), `202`.
Worker: `preparing` → lock `db:<id>:restore` (falha persistida `RESTORE_ALREADY_RUNNING`) →
`downloading` (get object, hash/checksum/size verificação) → `restoring` (adapter restore,
stream) → `verifying`/`completed`; falha → persistida. UI acompanha por polling de jobs
REST (fallback) + eventos realtime.

### 24.4 Backup agendado (scheduler)

`SaveConfiguration` valida schedule/timezone/retention/destino (`GetProvider`), calcula
`NextRunAt`, persiste, e `scheduleConfiguration`: cron → `ScheduleJobCron(aether.jobs.backups,
backups:<id>, backup.schedule, <cron>, tz)`; one-shot → `ScheduleAt` (jackson). No tick do
scheduler JetStream, o worker consome `backup.schedule` → `RunScheduled`: valida `next_run_at`
igual (senão no-op), checa ativos, cria `BackupJob(queued)` via enqueue/outbox, avança
`next_run`. Reconcile periódico (30s, com liderança) restaura/atualiza/remove schedules.

### 24.5 Snapshot agendado

Idêntico ao backup: `snapshot_schedules` + `aether.jobs.snapshots` +
`ScheduleJobCron(nativeSnapshotCron)`; worker executa `podman run` de tar da VM para o volume
(`snapshotWorker.execute`), persiste `snapshots` row, aplica retenção, avança `next_run`.

### 24.6 Cron de usuário

`POST /cron-jobs` valida expressão cron → `cron_jobs` row → Reconcile agenda
`cron.execute` no JetStream; worker consome → `RunOnce` um container one-shot com env
efetiva; `SetCronRun(last, next)`; falhas Nack/retry.

### 24.7 Destino S3 / Google Drive (settings)

`POST /settings/s3-destinations` (`type`: aws | cloudflare-r2 | minio | custom-s3 |
google-drive). Endpoints/região resolvidos por tipo (R2: `<account>.r2.cloudflarestorage.com`,
MinIO: custom endpoint with `UseSSL`, AWS: buckets regionais). Google Drive: OAuth por
destination (authorization URL com state TTL; callback troca código por token; tokens
encriptados no DB; refresh automático; `reauth_required` flag quando refresh falha).
Providers resolvidos centralmente (`settingsDestProvider`). "Test connection" via
Capabilities + Put/Head.

### 24.8 Monitoring pipeline

Aggregator a cada 2s: `podman ps -a` + `podman stats --no-stream` (batch único) → classifier
por labels (`aether.owner/service-type/service-id/project-id`) → agrega Aether/User/System →
rates (deltas) → storage por recurso (a cada 5 min, `podman system df -v`, cacheado) → snapshot
publicado no Core (`aether.monitoring.snapshot`) e persistido (agregado por tick; recursos a
cada 5 ticks; purge 7d). API expõe via reader NATS (latest/history/resource/collector).

### 24.9 Studio de database (fullscreen)

Rota `/studio/$dbId` (fora do `_shell`): tabs persistidas por DB (localStorage), editor
Monaco cursor-aware (seleção/statement sob cursor, Cmd+Enter), Object Explorer (introspect via
adapter por engine, com error state + retry), Create Table (schema grid inline, defaults, PK,
preview SQL), Context Menu (Edit/Rename/Delete), endpoints `tables/rename|drop|alter|catalog`,
SQL Intelligence (autocomplete, schema snapshot IndexedDB, rank com razões).

### 24.10 Realtime hub (WebSocket)

Handshake: connect → auth (cookie) → primeiro frame `{op:"subscribe", scope:"org", seq}`
→ `Authorize` (org/app/deployment + RBAC) → envio de replay (últimos N eventos ≥ seq) →
incremental: eventos do Core `notify:org` decodificados e emitidos; presence join/heartbeat/
leave; ping do servidor; close limpa presence + unsubscribes. Eventos efêmeros (seq=0) não
persistem ao replay.

### 24.11 Notificações

Sink `Notifications.Create` (tabela notifications) chamado em `NotifyDeploy` para status
notificáveis (queued/ready/failed/rolled_back/cancelled), backup/restore e alertas; UI recebe
por WS (toast) + contador do bell (com fallback REST 30s offline); badge atualizado por evento
real; recontagem por REST no mount.

## 25. Resumo das RFCs (blueprint completo do sistema)

| RFC | Tema | Essência |
|-----|------|----------|
| RFC-0000 | Template | Estrutura padrão: problema, contexto, princípios, decisão, alternativas, trade-offs, implementação, teste, impacto |
| RFC-0001 | Execution Engine | Abstração OCI + drivers; spec neutra; build; volumes/redes; GC |
| RFC-0002 | Networking Engine | Abstração proxy; rotas/middlewares/certs refs; config em memória |
| RFC-0003 | Certificate Engine | Soberania de certs; providers ACME; DNS-01 wildcard; store cifrado |
| RFC-0004 | Persistência | SQLite/PG/PG-HA por edição; camadas (state/event log/snapshot/secrets/audit); outbox; migrações |
| RFC-0005 | Event Bus | Eventos como fonte; outbox; replay; idempotência; sagas; timers |
| RFC-0006 | Runtime Driver API | Contrato completo do driver (imagens/containers/redes/volumes/build/info/GC) |
| RFC-0007 | Observabilidade | Logs estruturados; métricas sob demanda; tracing amostrado; alertas por evento; timeline/audit |
| RFC-0008 | Segurança | Rootless; KEK/DEK AES-GCM; RBAC; audit; supply chain |
| RFC-0009 | Installer | Fases idempotentes; update/rollback atômico; identidade inicial; uninstall seguro |
| RFC-0010 | Plugin System | Portas; manifestos; sandbox externe; carregamento sob demanda |
| RFC-0011 | Deployments | Saga e máquina de estados; rollback; previews; webhooks |
| RFC-0012 | Git Integration | Providers; webhooks; auto-deploy; branches; commits |
| RFC-0013 | Databases | Engines gerenciadas; creds cifradas; DSN; dumps; lifecycle |
| RFC-0014 | Backup e Restore | Agendamento; destinos; checksums; restore com validação |
| RFC-0015 | Multi-server | Agentes; mTLS; escalonamento; clustering |
| RFC-0016 | Marketplace | Templates; versionamento; instalação |
| RFC-0017 | API e CLI | Contratos; idempotência; CLI JSON; aether.yml |
| RFC-0018 | RBAC | Roles/permissions; enforcement no core |
| RFC-0021 | PostgreSQL | Banco único; migrações com advisory lock; prefered; bootstrapping |
| RFC-0031 | Notifications | Canal durável; notificação por evento; assinaturas |
| RFC-0032 | Organizations | Tenant model; membros; papéis; multi-tenant |
| RFC-0033 | Deployment spec first | Spec única no deployment; compose_hash; determinismo |
| RFC-0034 | OCI/Podman First | Podman como driver padrão; rootless; zero Docker |

## 26. Guia de troubleshooting operacional

| Sintoma | Checagem | Ação |
|---------|----------|------|
| API unhealthy | `podman logs aether-api`; DB reachable; migrações | conferir `aether-postgres` 15432 e `DATABASE_*` |
| Deploy falha | log do deployment (`~/.aether/logs/deployments/<id>.log`) e `podman logs <app container>` | ver tipos: `pack` ausente → rebuild imagem; `invalid run-image` → builder antigo (rebuild `build-builder.sh`); porta ocupada → remover container antigo por nome |
| Worker não processa | `curl :8081/ready`; `podman logs aether-worker`; consumers no NATS (`8222/jsz`) | consumers órfãos `integration-*`→ deletar; erro "invalid input" no recovery → ver NOT NULL (error_code '' | NULL) |
| Worker com "Authorization Violation" | env `AETHER_NATS_USER/PASSWORD` | usar `~/.aether/keys/nats.auth` no dev.sh |
| Realtime não atualiza UI | `/api/v1/events` (eventos da org) e stream `AETHER_EVENTS` | se só `queued`: notifier não configurado no worker (rever `Notifier`); se vazio: `PublishEvent` falhou (Append) |
| NATS down/estranho | `curl 127.0.0.1:8222/healthz`; `jsz` | restart por nome `aether-nats`; credenciais; versão ≥2.14 |
| Monitoring sem dados | `curl :8082/ready`; logs | host agent/watchdog; socket podman; `podman stats` na VM |
| Deploy fica queue | worker não acorda | ver outbox pendente (SELECT ... published_at IS NULL); workers órfãos; recover no startup |
| Socket podman (Mac) | `ls ~/.aether/podman.sock; podman --url unix://... info` | dev.sh usa gvproxy socket; fallback ssh -L validado |
| Builder Paketo no log | `podman images`/`builder metadata` | rebuild com `infra/buildpacks/builders/build-builder.sh` (ubuntu 24.04, sem Paketo) |

## 27. Referências documentais (apenas arquivo: existência, não conteúdo)

- Documentos históricos em `docs/00-*.md` … `18-roadmap.md` e `docs/spec/*` não são
  necessários para leitura — este arquivo é a fonte única. Eles existem para diff histórico e
  detalhes exóticos (ex.: análise paramétrica de consumo, experimentos pilot).
- RFCs em `docs/rfc/` guardam as decisões e trade-offs originais; resumidas na seção 25.
- `docs/AUDITORIA-DOKPLOY-vs-AETHER.md`: detalhe de paridade/divergência por recurso
  (comparação funcional linha a linha com o Dokploy); uso opcional para compatibilidade.

## 28. Glossário

- **Service**: unidade do produto (Application, Database, Worker, Cron, Compose) sob um Project.
- **Deployment**: execução versionada de um Service (número monotônico por app).
- **Outbox**: padrão transacional (grava intenção + estado na mesma transação; publica depois).
- **Durable consumer**: consumidor JetStream com estado de entrega persistido (sobrevive restart).
- **DLQ**: dead letter queue (`aether.dlq.<stream>`): falhas permanentes com metadata.
- **Reconcile**: re-sincronização Postgres → NATS (schedules) e estado de jobs no startup.
- **WorkQueuePolicy**: stream com semântica trabalho→consumo único por subject (jobs).
- **LimitsPolicy**: retenção por idade/tam para event log e DLQ.
- **Secrets**: valores cifrados (AES-256-GCM, DEK cifrado por KEK), nunca em claro.
- **Strap de deployment**: spec única JSON (build/run) persistida no deployment.

## 29. Checklist de release (mudança arquitetural)

1. Testes: suíte completa (`go test ./api/internal/... -p 1`), build dos 3 executáveis,
   `go vet`, `npm run typecheck` + `npm run build`.
2. Se mexeu em NATS: validar em NATS real (`AETHER_NATS_TEST_URL`) com integração + crash +
   scheduler + auth.
3. Se mexeu em outbox/recovery: validar cenários outbox pendente + restart (E2E real).
4. Se mexeu em workers: conferir concorrência, ACK/NAK classificado, notifier presente.
5. Se mexeu em UI: `tsgo --noEmit` + build; sem polling para dados via WS.
6. Atualizar: este documento + RFC afetada + AGENTS.md + testes.
7. Subir stack: `podman build -t aether.local/api:1 -f infra/Dockerfile . &&
   ./install-dev.sh start` (Mac) ou composição equivalente em Linux.
8. Validar ready: `/api/v1/ready`, `:8081/ready`, `:8082/ready`; deploy de teste → `ready`.
9. Sem comentários novos em código; inglês em tudo; design system intacto.
