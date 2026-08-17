# 06 — Domínios do Sistema (Bounded Contexts)

> **Status:** Definição de fronteiras de domínio.
> **Objetivo:** Delimitar responsabilidades de cada bounded context, com agregados, eventos e
> relações. Base para a modelagem de código (DDD).

---

## 0. Mapa de contexto geral

```
                 +-----------------------+
                 |      Identity        |   (orgs, users, teams, RBAC)
                 +----------+-----------+
                            |
+----------------+          |          +----------------+
|   Projects     |----------+----------|  Applications  |
+----------------+                     +-------+--------+
                                                 |
+----------------+    +---------------+    +-----v------+    +----------------+
|    Git         |----|   Deployments |    |  Build     |    |  Networking    |
+----------------+    +-------+-------+    +-----+------+    +-------+--------+
                              |                   |                  |
+----------------+    +-------v-------+    +-----v------+    +-------v--------+
| Certificates   |----|  Runtime      |----| Execution  |    |  (proxy/TLS)   |
+----------------+    +---------------+    | Engine     |    +----------------+
                              |            +-----+------+
+----------------+            |            +-----v------+    +----------------+
|     Logs       |            +------------|  Storage   |----|    Backups     |
+----------------+                         +-----+------+    +----------------+
+----------------+            +------------+-------------+    +----------------+
|    Metrics     |            | Observability | Timeline |    |  Infrastructure|
+----------------+            +---------------+-----------+    +----------------+
+----------------+            +----------------+              +----------------+
|   Marketplace  |------------|  Automation    |--------------|  Plugins       |
+----------------+            +----------------+              +----------------+
```

---

## 1. Lista consolidada de bounded contexts

| # | Contexto | Resumo |
|---|----------|--------|
| 1 | **Identity** | Users, Organizations, Teams, Sessions, RBAC, API keys, OIDC/SSO, MFA |
| 2 | **Projects** | Agrupamento de apps/environments por projeto |
| 3 | **Applications** | Modelo de aplicação, env vars, secrets, health checks |
| 4 | **Deployments** | Pipeline de deployment, rollback, preview, webhooks |
| 5 | **Runtime** | Execução, units systemd, redes, volumes (fronteira com runtime OCI) |
| 6 | **Execution Engine** | Abstração OCI; drivers (podman/docker/containerd/k8s) |
| 7 | **Build** | Build de imagens (Buildah), cache, registro |
| 8 | **Networking** | Domains, proxy provider, middlewares, LB, HTTP/3, service discovery |
| 9 | **Certificates** | Emissão, renovação, revogação, ACME providers |
| 10 | **Storage** | Volumes, quotas, blobs, object store |
| 11 | **Backups** | Backup/restore de estado e de dados de app |
| 12 | **Logs** | Streaming, retenção, rotação, buscas |
| 13 | **Metrics** | Coleta, agregação, exposição |
| 14 | **Observability** | Tracing, alertas, timelines |
| 15 | **Git** | GitHub/GitLab/Bitbucket providers, webhooks |
| 16 | **Marketplace** | Templates, one-click apps, categorias |
| 17 | **Automation** | Cron jobs, workers, automações de ciclo de vida |
| 18 | **Infrastructure** | Servers, clusters, agents, primitivas de nó |
| 19 | **Plugins** | Host, catálogo, sandbox |
| 20 | **Audit** | Audit log, trilha de auditoria |

---

## 2. Contexto por contexto

### 2.1 Identity

- **Agregados:** `User`, `Organization`, `Team`, `ApiKey`, `Session`, `RoleBinding`.
- **Responsabilidades:**
  - Autenticação (password hashed com Argon2id; OIDC/SSO; MFA na fase 2+).
  - Autorização (RBAC por organização; escopos de API key).
  - Gestão de sessões e tokens (JWT com rotação; refresh).
  - Provisionamento de convites (e-mail/invite).
- **Eventos:** `user.created`, `org.created`, `team.invited`, `user.role.changed`,
  `apikey.created`, `session.revoked`.
- **Fronteiras:** nenhum outro contexto decide quem pode agir; apenas consulta `authorize()`.

### 2.2 Projects

- **Agregados:** `Project`, `Environment`.
- **Responsabilidades:** agrupar apps; separar ambientes (prod/staging/dev); definir
  variáveis por ambiente; scope de RBAC.
- **Eventos:** `project.created`, `environment.added`, `environment.deleted`.
- **Relações:** 1 Projeto tem N Environments; 1 Environment tem N Applications.

### 2.3 Applications

- **Agregados:** `Application`, `AppConfig`, `EnvVar`, `SecretRef`, `HealthCheck`, `VolumeRef`.
- **Responsabilidades:**
  - Definir fonte (imagem, git, compose).
  - Definir resources (cpu/ram/limits), replicas, health checks.
  - Gerenciar env vars/secrets (referências a secret store, não valores).
  - Gerenciar volumes.
- **Eventos:** `app.created`, `app.updated`, `app.removed`, `envvar.changed`,
  `secret.referenced`, `healthcheck.updated`.
- **Fronteiras:** não executa nada; delega a Deployments/Runtime.

### 2.4 Deployments

- **Agregados:** `Deployment`, `DeploymentTarget`, `Rollback`, `PreviewDeployment`.
- **Responsabilidades:**
  - Orquestrar a saga de deploy (build → schedule → health → promote).
  - Rollback (restaurar deployment anterior).
  - Preview deployments (por PR/branch).
  - Receber webhooks de Git/registry e converter em comandos.
- **Eventos:** `deployment.created`, `deployment.building`, `deployment.scheduled`,
  `deployment.ready`, `deployment.failed`, `deployment.rolledback`, `deployment.health_failed`.
- **Fronteiras:** o deployment conhece *conceitos* de Build/Runtime/Networking, mas só os
  aciona por eventos/portas.

### 2.5 Runtime

- **Agregados:** `ServiceInstance`, `Network`, `Volume`, `UnitSpec`.
- **Responsabilidades:**
  - Representar o estado desejado de execução (spec).
  - Converter spec em units systemd (Quadlet).
  - Gerenciar ciclo de vida (start/stop/restart/scale).
  - Service discovery e redes.
- **Eventos:** `service.started`, `service.stopped`, `service.restarted`,
  `service.crashed`, `service.scaled`, `network.created`.
- **Fronteira crítica:** Runtime **não conhece Podman** — conhece a porta `RuntimePort`
  do Execution Engine.

### 2.6 Execution Engine

- **Agregados:** `RuntimeHandle`, `ImageRef`, `ContainerSpec`, `DriverSpec`.
- **Responsabilidades:**
  - Traduzir `ContainerSpec`/`ComposeSpec` (declarativo, neutro) em operações do driver.
  - Implementar drivers: `podman` (default), `docker`, `containerd`, `k8s` (futuros).
  - Isolamento da semântica OCI (pull/push/run/stop/exec/logs/stats/network/volume).
- **Eventos:** `runtime.op_started`, `runtime.op_finished`, `runtime.op_failed`.
- **Fronteira:** único contexto que fala "contêiner". Interface em RFC-0006.

### 2.7 Build

- **Agregados:** `Build`, `BuildCacheEntry`, `ImageDigest`.
- **Responsabilidades:**
  - Compilar fonte em imagem OCI (Dockerfile, Buildpacks CNB, custom).
  - Gerenciar fila de builds, concorrência, janelas.
  - Gerenciar cache de build (LRU + GC).
  - Registrar imagens no registry local.
- **Eventos:** `build.started`, `build.finished`, `build.failed`, `build.cache_pruned`.
- **Fronteiras:** usa Execution Engine (buildah via driver `build`); não sabe de apps.

### 2.8 Networking

- **Agregados:** `Domain`, `Route`, `ProxyConfig`, `Middleware`, `LoadBalancer`.
- **Responsabilidades:**
  - Modelar domínios vinculados a apps/serviços.
  - Gerar configuração do proxy provider (Traefik hoje) **em memória**.
  - Middlewares (rate limit, forward auth, headers, rewrites).
  - HTTP/3, TLS (delega cert refs ao Certificates), LB.
  - Service discovery (rotas dinâmicas por serviço).
- **Eventos:** `domain.added`, `route.updated`, `middleware.changed`, `proxy.reloaded`.
- **Fronteiras:** Networking chama Certificates para obter cert refs; chama o provider de proxy
  via porta `NetworkPort`.

### 2.9 Certificates

- **Agregados:** `Certificate`, `AcmeAccount`, `Challenge`, `CertRenewal`.
- **Responsabilidades:**
  - Emissão, renovação (com jitter de tempo), revogação.
  - Gerenciar contas ACME; desafios HTTP-01/DNS-01.
  - Histórico e auditoria de emissões; alertas de falha.
  - Wildcard (DNS-01) e SAN.
- **Eventos:** `cert.issued`, `cert.renewing`, `cert.renewed`, `cert.renewal_failed`,
  `cert.revoked`.
- **Fronteiras:** próprio Certificate Manager (nunca o proxy controla certs).

### 2.10 Storage

- **Agregados:** `Volume`, `VolumeClaim`, `Snapshot`, `BlobRef`.
- **Responsabilidades:**
  - Provisionar volumes (local, NFS, CSI futuro) via Execution Engine.
  - Quotas e limites.
  - Blob store (backups, imagens, templates) com driver (local, S3, etc.).
- **Eventos:** `volume.created`, `volume.resized`, `volume.snapshot`, `volume.removed`.

### 2.11 Backups

- **Agregados:** `BackupJob`, `Backup`, `Restore`, `Retention`.
- **Responsabilidades:**
  - Agendar backups (por evento/timer).
  - Executar backup de estado e de dados de app.
  - Restaurar (validação, reconciliação).
  - Política de retenção e destinos remotos.
- **Eventos:** `backup.scheduled`, `backup.started`, `backup.finished`, `backup.failed`,
  `restore.started`, `restore.finished`.

### 2.12 Logs

- **Agregados:** `LogStream`, `LogEntry`.
- **Responsabilidades:**
  - Streaming de logs (sockets unix → SSE).
  - Rotação/compressão/retenção.
  - Busca (com índices mínimos).
- **Eventos:** `log.rotated`, `log.retention_enforced`.

### 2.13 Metrics

- **Agregados:** `MetricSeries`, `MetricWindow`.
- **Responsabilidades:**
  - Coleta sob demanda (cgroup v2, /proc).
  - Agregação em memória (janelas).
  - Exposição `/metrics` (formato Prometheus) quando há subscriber.
- **Eventos:** `metrics.subscribed`, `metrics.aggregated`.

### 2.14 Observability

- **Agregados:** `Alert`, `AlertRule`, `TraceSpan`, `TimelineEvent`.
- **Responsabilidades:**
  - Avaliar regras de alerta em eventos (não polling).
  - Rastreamento leve de operações.
  - Timeline consolidada por recurso (déjà vu de eventos).
- **Eventos:** `alert.fired`, `alert.resolved`, `timeline.updated`.

### 2.15 Git

- **Agregados:** `GitSource`, `GitProvider`, `Webhook`.
- **Responsabilidades:**
  - Integrar com GitHub/GitLab/Bitbucket (apps OAuth, tokens).
  - Clonar/buscar fontes.
  - Receber webhooks e emitir eventos de trigger.
- **Eventos:** `git.webhook_received`, `git.source_fetched`, `git.commit_detected`.
- **Fronteiras:** Git emite `source.updated` que Deployments consome; não executa deploy.

### 2.16 Marketplace

- **Agregados:** `Template`, `TemplateCategory`, `AppDefinition`.
- **Responsabilidades:**
  - Catálogo de one-click apps e templates.
  - Versionamento de templates (imutáveis por versão).
  - Instanciação → gera spec de Application.
- **Eventos:** `template.installed`, `template.updated`.
- **Fronteiras:** Marketplace produz spec; não executa.

### 2.17 Automation

- **Agregados:** `CronJob`, `WorkerSpec`, `Automation`, `Schedule`.
- **Responsabilidades:**
  - Cron jobs (agendados pelo scheduler determinístico).
  - Workers (processos de longa duração).
  - Automações de ciclo de vida (ex.: auto-deploy, auto-scale simples).
- **Eventos:** `cron.triggered`, `worker.started`, `worker.stopped`.

### 2.18 Infrastructure

- **Agregados:** `Server`, `Agent`, `Cluster`, `ProviderConfig` (cloud), `NodeSpec`.
- **Responsabilidades:**
  - Registrar servidores/agentes.
  - Clusterização lógica; afinidade; tolerância.
  - Configuração de providers de infra (Hetzner, AWS...) via plugins.
- **Eventos:** `server.registered`, `server.heartbeat`, `agent.upgraded`, `cluster.formed`.

### 2.19 Plugins

- **Agregados:** `Plugin`, `PluginManifest`, `PluginInstance`.
- **Responsabilidades:**
  - Carregar/descarregar plugins sob demanda.
  - Verificar assinatura/trust.
  - Sandbox e permissões de plugin.
  - Catálogo de plugins disponíveis.
- **Eventos:** `plugin.installed`, `plugin.enabled`, `plugin.disabled`.

### 2.20 Audit

- **Agregados:** `AuditEvent`.
- **Responsabilidades:** registrar ações administrativas (quem, o quê, quando, de onde).
- **Eventos:** `audit.recorded` (append-only).

---

## 3. Relações e dependências entre contextos

```
Identity ──► (autoriza) ──► todos os demais
Projects ──► Applications
Applications ──► Deployments, Build, Runtime, Networking, Storage, Git
Deployments ──► Build, Runtime, Networking, Certificates, Observability
Runtime ──► ExecutionEngine ──► (drivers OCI)
Build ──► ExecutionEngine, Storage
Networking ──► Certificates, ProxyProvider
Certificates ──► (ACME providers via porta)
Backups ──► Storage, ExecutionEngine, Infrastructure
Observability ──► Logs, Metrics, Audit, Deployments
Infrastructure ──► Runtime, ExecutionEngine, Storage
Plugins ──► (implementam portas de vários contextos)
```

Regras:
- Dependência de **interface** (portas) é permitida; dependência de **implementação** cruzada é proibida.
- Contextos se comunicam por **eventos** (assíncrono) ou **comandos** (síncrono local) — nunca
  por chamada direta entre módulos de domínio distintos fora das portas.

---

## 4. Mapas de eventos por contexto (exemplo)

| Contexto origem | Evento | Consumidores |
|-----------------|--------|--------------|
| Identity | `user.role.changed` | Audit, Notify |
| Applications | `app.created` | Deployments, Marketplace |
| Deployments | `deployment.ready` | Observability, Notify, Networking |
| Git | `source.updated` | Deployments (auto-deploy) |
| Certificates | `cert.renewal_failed` | Notify, Observability |
| Runtime | `service.crashed` | Observability, Notify |
| Metrics | `alert.fired` | Notify |

---

## 5. Métricas de design de contexto

- Cada contexto é **opcionalmente extraível** para processo separado (graças ao event bus).
- Nenhum contexto importa a camada de infra de outro.
- Testes de cada contexto rodam com mocks das portas.

---

## 6. Referências

- Interfaces de portas: RFC-0006 (Runtime), RFC-0002 (Network), RFC-0003 (Cert).
- Eventos detalhados: RFC-0005.
- Roadmap de amadurecimento dos contextos: [`18-roadmap.md`](18-roadmap.md).
