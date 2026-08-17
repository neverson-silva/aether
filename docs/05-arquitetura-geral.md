# 05 — Arquitetura Geral

> **Status:** Documento central de arquitetura.
> **Objetivo:** Definir a arquitetura lógica, física, de domínio, modular, em camadas, de
> eventos, de plugins, de APIs, de persistência, de cache, de observabilidade, de instalação,
> de atualização, de backup e de segurança. Este documento consolida e referencia os demais.

---

## 1. Visão de um diagrama de um parágrafo

```
+-----------------------------------------------------------+
|                    UI / CLI / API (clients)                |
+-----------------------------+-----------------------------+
|                            Core (Plano de Controle)        |
|  +---------+  +----------+  +----------+  +-------------+ |
|  | Domain  |  | EventBus |  | Scheduler|  | Cert Engine | |
|  | Model   |  | EventLog |  |          |  |             | |
|  +---------+  +----------+  +----------+  +-------------+ |
|  | Identity|  | RBAC     |  | Plugins  |  | Marketplace | |
|  | Git     |  | Notify   |  | Cache    |  | Observab    | |
|  +---------+--+----------+--+----------+--+-------------+ |
+-----------------------------+-----------------------------+
                              | (gRPC/IPC sobre eventos)
+-----------------------------+-----------------------------+
|                       Agent (por servidor)                 |
|  +----------+   +--------------+   +--------------------+ |
|  | Runtime  |   | Unit Manager |   | Logs/Metrics/      | |
|  | Execution|   | (Quadlet/    |   | Monitoring         | |
|  | Engine   |   |  systemd)    |   +--------------------+ |
|  +----------+   +--------------+   | Build (Buildah)    | |
|       |                             +--------------------+ |
|  +----------------+                 +------------------+   |
|  | OCI Runtime    |                 | Certificate      |   |
|  | Drivers        |                 | Storage          |   |
|  +----------------+                 +------------------+   |
+-----------------------------+-----------------------------+
                              |
                    +---------+----------+
                    | Runtime OCI (host)  |  ← Podman rootless, crun, conmon
                    | Units systemd      |  ← Quadlet
                    | Networking Engine  |  ← proxy (Traefik provider)
                    | Persistence (SQLite/PG) |
                    +--------------------+
```

A regra de ouro do diagrama: **as camadas superiores (Core) nunca conversam com o runtime OCI
diretamente.** Toda a execução passa pelo Agent → Execution Engine → OCI Runtime Driver.

---

## 2. Arquitetura lógica

### 2.1 Camadas lógicas

| Camada | Componentes | Responsabilidade |
|--------|-------------|------------------|
| **Presentation** | UI (SPA), CLI, API REST, API Keys | Interação do usuário e automação |
| **Application** | Command handlers, Use cases, Sagas | Orquestrar casos de uso; coordenação |
| **Domain** | Entidades, Value Objects, Aggregates, Domain Events | Regras de negócio puras; sem I/O |
| **Infrastructure** | Repositórios, EventStore, Runtime Drivers, Providers | I/O real: banco, runtime, git, cloud, proxy |
| **Runtime (host)** | OCI runtime, systemd units, proxy, storage | Execução física |

### 2.2 Regras de dependência

- **Dependência unidirecional**: Presentation → Application → Domain ← Infrastructure
  (Domain nunca depende de Infrastructure; a infraestrutura implementa *portas* definidas pelo domínio).
- O domínio define **interfaces (portas)**; a infraestrutura fornece **adaptadores** (Hexagonal/Ports & Adapters).
- Nenhuma entidade de domínio conhece Podman/Docker/HTTP/banco.

### 2.3 Portas principais do domínio

```
RuntimePort            (Execution Engine)
NetworkPort            (proxy provider)
CertificatePort        (ACME providers)
StoragePort            (volumes, blobs)
GitPort                (github/gitlab/bitbucket)
RegistryPort           (OCI registries)
MailPort / NotifyPort  (notificações)
MetricsPort            (telemetria)
SecretStorePort        (crypto + storage de secrets)
```

Cada porta tem uma implementação concreta por provider; trocar provider não muda o domínio.

---

## 3. Arquitetura física (implantação em um servidor)

### 3.1 Componentes instalados no host

| Componente | Tipo | Usuário | Init |
|-----------|------|---------|------|
| `aether-core` | binário | `aether` (system) | systemd `aether-core.service` |
| `aether-agent` | binário | `aether-agent` (system) | systemd `aether-agent.service` |
| `aether-proxy` (Traefik) | binário | `aether-proxy` (rootless) | systemd `aether-proxy.service` |
| Runtime OCI (podman, crun, conmon, buildah, skopeo, fuse-overlayfs) | binários | system | systemd (unit pods para apps) |
| SQLite | arquivo | `aether` | — |
| App units | Quadlet units | usuário app dedicado | systemd |

### 3.2 Isolamento por usuário

- `aether` (core): dono do banco, event log, caches, chaves.
- `aether-agent`: dono de diretórios de build, cache de build; executa buildah.
- `aether-proxy`: dono de sockets de proxy, certs (leitura), config em memória.
- `app-<id>` (um usuário por app OU um usuário compartilhado com namespace por app — decidido
  por balanceamento de overhead): dono das units do app.

**Decisão de overhead vs segurança:** criar um usuário systemd por app adiciona entradas em
`/etc/passwd` e custo mínimo; em servidores com muitos apps, usar **um usuário de execução por
grupo** com `systemd` user instances separadas por app quando necessário. Detalhes na RFC-0006.

### 3.3 Rede física

- `aether-proxy` escuta :80/:443 (cap net_bind_service via systemd `AmbientCapabilities` ou
  porta alta + redirect).
- Apps escutam em IP de loopback / sockets unix / redes de containers.
- Service discovery via `systemd` (socket activation) ou rede de containers do podman.

### 3.4 Storage físico

```
/var/lib/aether/
├── state/               # SQLite + WAL + snapshots
├── eventlog/            # event log (append-only, por segmento)
├── cache/               # LRU snapshots, object cache
├── keys/                # chaves de criptografia (0600)
├── certs/               # certificados emitidos
├── plugins/             # plugins carregáveis
├── builds/              # workspace de build (efêmero)
└── apps/
    ├── <app-id>/
    │   ├── units/       # units systemd geradas
    │   ├── env/         # environment files (0600)
    │   └── data/        # volumes locais (opcional)
/var/log/aether/         # logs do sistema + rotacionados
```

---

## 4. Arquitetura de domínio

Bounded contexts completos em [`06-dominios-sistema.md`](06-dominios-sistema.md). Resumo:

- **Applications**, **Projects**, **Deployments**, **Runtime**, **Networking**, **Certificates**,
  **Observability**, **Identity**, **Marketplace**, **Automation**, **Storage**, **Infrastructure**,
  **Plugins**, **Git**, **Build**, **Metrics**, **Logs**, **Backups**.

Cada bounded context tem: agregados, eventos, serviços de domínio, políticas e fronteiras.

---

## 5. Arquitetura modular

### 5.1 Módulos do binário `core`

Módulos compilados no binário (não são processos):

| Módulo | Responsabilidade |
|--------|------------------|
| `http-api` | REST API, SSE/WebSocket, API keys |
| `command` | Application layer (use cases) |
| `domain` | Entidades e regras |
| `eventbus` | Publicação/inscrição + event log |
| `scheduler` | Timers determinísticos (cron, retries) |
| `cache` | LRU + snapshot |
| `certengine` | ACME, renew, revoke |
| `git` | Clones, webhooks, providers |
| `notify` | Notificações (mail, webhook, discord, slack) |
| `marketplace` | Templates, one-click apps |
| `rbac` | Identidade, orgs, teams, permissões |
| `audit` | Audit log |
| `observability` | Telemetria leve, `/metrics` |
| `gc` | Política central de coleta |
| `pluginhost` | Carregador e runtime de plugins |
| `agentrpc` | Comunicação com agentes |

### 5.2 Módulos do binário `agent`

| Módulo | Responsabilidade |
|--------|------------------|
| `runtimeport` | Execution Engine |
| `unitmanager` | Geração/aplicação de units systemd (Quadlet) |
| `execution` | Fluxos de execução de containers (podman) |
| `build` | Buildah rootless, fila de builds |
| `logs` | Streaming de logs (sockets) |
| `metrics` | Coleta sob demanda (cgroup v2, /proc) |
| `storage` | Volumes, quotas |
| `certlocal` | Coordenação local de renovação de certs (com core) |
| `registry` | Pull/push (skopeo), auth |
| `networklocal` | Configuração de redes, service discovery |

### 5.3 Módulo do proxy

O proxy é uma **instância do provider** (default Traefik). A plataforma o configura via API em
memória. Em versões futuras o módulo `proxy` do core troca o binário do provider mantendo a
mesma interface (`networking`).

---

## 6. Arquitetura em camadas (detalhe por operação)

### 6.1 Exemplo: Deploy de aplicação

```
[UI/API] → (POST /api/apps/:id/deploy)
→ Core: valida RBAC, cria Deployment (Domain)
→ Core: emite DomainEvent `deployment.created`
→ EventBus: persiste no event log; agenda
→ Core (command): coordena saga de deploy
   → Agent: build (se necessário) via ExecutionEngine → Buildah
   → Agent: escreve unit Quadlet (UnitManager)
   → Agent: `systemctl --user start` (via quadlet → podman)
   → Agent: executa health check (se configurado)
   → Agent: emite `deployment.ready` / `deployment.failed`
→ Core: atualiza projeção; agenda certificados (se domínio novo)
→ Core: NetworkingEngine → Traefik (dynamic config em memória)
→ Core: emite eventos para UI (SSE) e audit log
```

Nenhuma camada superior chamou Podman diretamente. O Agent é a única fronteira com o runtime.

---

## 7. Arquitetura de eventos

Detalhes: [`12-event-bus.md`](12-event-bus.md). Resumo dos princípios:

1. **Postgres é fonte de verdade** dos estados; eventos são o canal de entrega realtime + trilha recente.
2. **Barramento Redis (produção)**: pub/sub `notify:org:<org>` para fanout realtime + Streams para
   event log (`ev:org:<org>`, seq por org, MAXLEN) e fila de deploys (`q:deployments:*`, consumer groups).
   Em dev/teste, `AETHER_RUNTIME_BACKEND=memory` usa pub/sub e log em processo.
3. **Zero polling no frontend**: dados via REST bootstrap + WebSocket único (`/api/v1/ws/realtime`);
   hub server-side com subscriptions por escopo autorizadas (org/app/deployment), `seq` para replay
   (persistido em `localStorage`) e eventos **efêmeros** (build log, app state) que não persistem.
4. **Pull-policy=never no build CNB**: builder/run/lifecycle provisionados localmente pelo
   `install.sh` — builds não dependem de rede/registry no momento da compilação.
5. **Idempotência** nos handlers (processar evento repetido é seguro; consumer group garante 1×).
6. **Polling legítimo restante**: telemetria (net-q 15s, presence 30s), SSE para follow de logs
   (app/host/databases) e host-stats 2s; fallback de notificações 30s só quando o WS está fora.

---

## 8. Arquitetura de plugins

Detalhes: [`14-plugin-system.md`](14-plugin-system.md). Resumo:

- Plugins implementam **portas** (provider de git, dns, cloud, storage, etc.) — nenhum plugin
  é obrigatório.
- Carregamento **sob demanda**: plugin é baixado/ativado apenas quando o usuário configura o
  provider correspondente.
- Dois tipos: **native** (compilados na plataforma — leves, alta performance) e **external**
  (processos/bundles — extensibilidade de terceiros, isolados por sandbox).
- A plataforma core nunca depende de plugin.

---

## 9. Arquitetura de APIs

Detalhes: [`17-api-cli.md`](17-api-cli.md). Resumo:

- REST (v1) com OpenAPI; SSE para eventos; WebSocket para terminal/logs.
- Versionamento por URL (`/api/v1`).
- Autenticação: API keys por usuário/org + OIDC/SSO (fase 2+).
- CLI (`aether`) é um cliente fino da API + funções locais (instalação/update).
- Idempotência: requisições de operações longas retornam `202` + `location` para o recurso
  criado (Deployment), com polling **curto** (ou SSE) para status.

---

## 10. Arquitetura de persistência

Detalhes: [`10-persistencia.md`](10-persistencia.md). Resumo:

| Edição | Persistência | Racional |
|--------|--------------|----------|
| Community | SQLite (WAL) | Zero processo, zero setup, suficiente para self-host |
| Business | PostgreSQL | Concorrência, extensibilidade, multi-tenant maior |
| Enterprise | PostgreSQL HA (Patroni) | Alta disponibilidade |

Camadas:
- **State store** (estado operacional: apps, deploys, orgs, etc.).
- **Event log** (append-only).
- **Snapshot store** (checkpoints de projeções).
- **Secrets** (criptografados, ver `16-seguranca.md`).
- **Object store** (imagens OCI → registry; backups → blob storage).

---

## 11. Arquitetura de cache

Detalhes: [`11-cache.md`](11-cache.md). Resumo:

- Cache em memória (LRU, TTL, max_bytes/max_entries) dentro do `core`.
- Snapshot para disco (compacto) com checkpoint; recuperação na inicialização.
- Cache de configuração do proxy (em memória no provider).
- Cache de camadas de imagem (deduplicação nativa do podman + GC).
- Nada cresce infinitamente (limites + GC).

---

## 12. Arquitetura de observabilidade

Detalhes: [`13-observabilidade.md`](13-observabilidade.md). Resumo:

- **Logs**: estruturados, rotacionados, streaming via SSE.
- **Metrics**: agregadas em memória; expostas em `/metrics` (formato Prometheus) apenas quando
  há subscriber (ou sob demanda).
- **Tracing**: spans leves para operações de deploy/backup; amostragem configurável.
- **Alerts**: regras avaliadas em eventos (não por polling).
- **Timeline/Events**: UI mostra timeline de eventos por recurso.
- **Audit**: log imutável de ações administrativas.

---

## 13. Arquitetura de instalação

Detalhes: [`15-installer.md`](15-installer.md). Resumo:

- Instalador bash (mínimo) + binário instalador interno (`aether install`).
- Detecta distro/arch/init; instala dependências mínimas; cria usuários; escreve units;
  inicializa banco; bootstrap token; sem downloads de imagens.
- Idempotente; desinstalação limpa.

---

## 14. Arquitetura de atualização

Detalhes: [`15-installer.md`](15-installer.md). Resumo:

- `aether update`: baixa binário assinado → valida checksum → backup automático do estado →
  troca binário → migrações transacionais → recarrega units → agentes atualizam por evento.
- Rollback: restaura binário anterior + migração reversa.
- Zero-downtime: core roda atrás de socket systemd; substituição atômica.

---

## 15. Arquitetura de backup

Detalhes: RFC-0014. Resumo:

- Backup de estado: SQLite `VACUUM INTO` / `pg_dump` + event log + certs + secrets.
- Backup de volumes de app: `tar` incremental ou snapshot de filesystem.
- Destinos: blob storage (S3-compatível) ou diretório remoto (SSH).
- Agendamento dirigido por eventos + janelas de baixa atividade; retenção configurável.
- Restore: importar backup → restaurar estado → reconciliar runtime (idempotente).

---

## 16. Arquitetura de segurança

Detalhes: [`16-seguranca.md`](16-seguranca.md). Resumo:

- Rootless por padrão (podman rootless, proxy rootless, agentes não-root).
- Least privilege em todos os processos/usuários.
- Secrets criptografados (AES-256-GCM; chave protegida no host; master key via env/TPM).
- Comunicação TLS (core↔agent muttual TLS com certificados de identidade).
- RBAC, OIDC/SSO (fase 2+), MFA (fase 2+), API keys hasheadas.
- Audit logs imutáveis.
- Hardening de sistema: `CapabilityBoundingSet`, `NoNewPrivileges`, seccomp, AppArmor/SELinux
  quando disponível.

---

## 17. Arquitetura física multi-servidor (visão futura)

Detalhes: RFC-0015. Resumo:

- 1 servidor **central** (core + banco + proxy opcional) e N servidores **workers** (agent).
- Comunicação core↔agent por gRPC com mTLS sobre rede privada/VPN.
- Deploy distribuído: scheduler roteia workloads; agent executa; eventos replicam estado.
- Clusters: agrupamento lógico; afinidade; tolerância a falhas.

---

## 18. Padrões de design transversais

| Padrão | Uso |
|--------|-----|
| Ports & Adapters (Hexagonal) | Todo provider é adaptador de uma porta de domínio |
| Command/Query Separation (CQRS leve) | Leitura via projeções; escrita via comandos+eventos |
| Event Sourcing | Event log como fonte primária de verdade |
| Saga | Deploy, backup, restore como sagas com compensação |
| Outbox pattern | Eventos persistidos junto com o comando (transação local) |
| Idempotency keys | Todos os handlers e API de operações longas |
| Circuit breaker | Integrações externas (git, cloud, ACME) |
| Leader election (multi-server) | Garantir 1 scheduler ativo por cluster |
| Sidecar-free | Nada de sidecars obrigatórios |

---

## 19. Decisões arquiteturais de alto nível (registro de decisão)

| ADR | Decisão | Justificativa |
|-----|---------|---------------|
| ADR-001 | Linguagem: **Go** (ou Rust, decisão final da equipe; premissa Go neste doc) | Binário estático, baixa RAM, concorrência nativa, compilação rápida, ecossistema de OCI maduro (podman é Go) |
| ADR-002 | Runtime padrão: **Podman + crun + Quadlet + systemd** | Daemonless, rootless, declarativo via systemd, zero container de suporte |
| ADR-003 | Proxy padrão: **Traefik via provider** | Dynamic config em memória, service discovery, middleware rico; interface abstrai troca futura |
| ADR-004 | Persistência padrão: **SQLite** | Zero processo/setup; perfeito para o custo-alvo; PG disponível por edição |
| ADR-005 | Event log próprio (não NATS/Kafka) | Sob demanda; evita processo externo; suficiente para o volume |
| ADR-006 | UI: SPA estática servida pelo core (sem Node no runtime) | Sem processo Node extra; build front-end separado no pipeline |
| ADR-007 | Estado de apps: **units systemd declarativas** | systemd já faz restart, resource limits, boot order, socket activation — de graça |
| ADR-008 | Um binário `core` monolítico modular (não microserviços) | Menos processos, menos RAM, atualização simples; módulos internos dão modularidade |

### Justificativa do ADR-008 (monólito modular)

Microserviços resolveriam "escalar" com custo de N processos, N pontos de estado, observabilidade
complexa e atualização complicada — exatamente o que queremos evitar. O monólito modular com
boundaries internas claras (ports & adapters + eventos) oferece a mesma separação lógica com 1/10
do custo fixo. Quando houver demanda real (muitos usuários), **extraímos** módulos (ex.: proxy,
agent) que já têm interface de processo definida, sem reescrita.

---

## 20. Matriz de conformidade com o manifesto

| Princípio | Onde é garantido |
|-----------|------------------|
| P1 OCI First | RFC-0001, RFC-0006 (Execution Engine, Runtime Driver API) |
| P2 Zero desperdício estrutural | Cap. 2–5 deste doc; 04 |
| P3 Mínimos processos | Cap. 5 (modular), ADR-008 |
| P4 Poucos containers | Cap. 6, 7 |
| P5 Eventos como verdade | Cap. 7; RFC-0005 |
| P6 Simplicidade operacional | Cap. 13, 14; RFC-0009 |
| P7 Menos, mas composto | ADR-001..008 |
| P8 Dados do usuário | Cap. 15; RFC-0014; [`16-seguranca.md`](16-seguranca.md) |
| P9 Segurança por padrão | Cap. 16 |
| P10 Crescimento sem reescrita | Cap. 17; RFC-0015 |

---

## 21. Referências cruzadas

- Domínios: [`06-dominios-sistema.md`](06-dominios-sistema.md)
- Execution Engine: [`07-execution-engine.md`](07-execution-engine.md) + RFC-0001/RFC-0006
- Networking: [`08-networking-engine.md`](08-networking-engine.md) + RFC-0002
- Certificados: [`09-certificate-engine.md`](09-certificate-engine.md) + RFC-0003
- Persistência: [`10-persistencia.md`](10-persistencia.md) + RFC-0004
- Cache: [`11-cache.md`](11-cache.md)
- Eventos: [`12-event-bus.md`](12-event-bus.md) + RFC-0005
- Observabilidade: [`13-observabilidade.md`](13-observabilidade.md) + RFC-0007
- Plugins: [`14-plugin-system.md`](14-plugin-system.md) + RFC-0010
- Instalação/atualização: [`15-installer.md`](15-installer.md) + RFC-0009
- Segurança: [`16-seguranca.md`](16-seguranca.md) + RFC-0008
- API/CLI: [`17-api-cli.md`](17-api-cli.md) + RFC-0017
- Roadmap: [`18-roadmap.md`](18-roadmap.md)
