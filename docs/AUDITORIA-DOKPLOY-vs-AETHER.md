# Auditoria Técnica Comparativa — Dokploy 0.29.14 × Aether PaaS

> Engenharia reversa + auditoria de paridade. Fonte: código-fonte do Dokploy (`dokploy-0.29.14/`, 186k LOC TS) e código-fonte do Aether (`api/internal/`, `web/`, `infra/`). Análise estática; nenhuma exploração destrutiva. Itens não comprovados marcados como `UNVERIFIED`.

---

## 1. Executive Summary

**O Aether está em ~35–40% de paridade funcional com o Dokploy**, com o núcleo (deploy de apps + domínios + auth + variables) sólido e vários módulos em estado de *scaffold* (tabela + endpoint + UI, sem executor).

**Principais findings:**

1. **Deploy pipeline real e funcional** — único ponto forte de ponta a ponta: fila em Postgres (polling), 4 caminhos de build (Dockerfile/Nixpacks/SmartBuild-Paketo/custom com nginx), healthcheck HTTP, rollback, logs com streaming. Superior ao Dokploy em **estados de deploy** (8 estados com máquina de transições vs 4 do Dokploy) e em **healthcheck explícito**.
2. **Muitos módulos são cascas**: databases (sem provisionamento), backups/snapshots (stubs que retornam sucesso), cron/schedules (sem scheduler), alerts (sem evaluator), autopilot (morto), previews (dead-end), API keys (criadas, nunca verificadas), storage/gdrive (desconectado), volumes (sem criação), gitops (record-only), outbound webhooks (sem dispatch), servers/clusters (sem agente).
3. **Vantagens reais do Aether**: gerador próprio de Dockerfile/nginx para builds custom, machine de estados de deploy, SSE/terminal próprios, análise de fonte (specs/planner) com geração de Dockerfile, detecção SPA com fallback nginx. *(Nota: a criptografia at-rest deixou de ser diferencial — ver §4/§11: o Dokploy também cifra as colunas `env` via `encryptedText` AES-256-GCM.)*
4. **Gaps críticos (P0)**: provisionamento de databases; backups/restores reais + S3; scheduler de cron/snapshots; cancelamento de deploy; API keys funcionais; previews de PR funcionais.
5. **Diferença arquitetural fundamental**: Dokploy = fila em memória por grupo + execução via **Docker Swarm** + servidores remotos via **SSH** + workers BullMQ dedicados (schedules). Aether = **worker único in-process com polling** + podman CLI local. Isso limita concorrência, cancelamento e multi-host.
6. **Bugs críticos descobertos na segunda passada (ver §20)**: (a) **ACME/HTTPS nunca ativa** — `VerifyCertificate` sempre retorna false (`TraefikBin` vazio + `wget -q` sem "200"), o worker de domínios só escreve config HTTP e o `cert_status` fica pending até os retries morrerem; (b) **build "custom" ignora `start_command`/`install_command`** — o gerador sempre produz Dockerfile nginx estático, então apps SSR (Next/Nuxt) não são deployáveis com o build default; (c) **callback OIDC quebrado** — `redirect_uri` GET vs rota POST + `state` não validado.

**Risco nº 1**: a confiança enganosa — endpoints retornam 200 (restore, snapshot, gitops sync) sem efeito real. **Risco nº 2**: módulos construídos sem executor (o padrão CRUD-only se repete em 10+ slices).

---

## 2. Arquitetura Comparativa

| Dimensão | Dokploy | Aether |
|---|---|---|
| Backend | Next.js + tRPC + better-auth + drizzle (packages/server) | Go 1.26 + Gin + pgx + sqlc (hexagonal por feature) |
| Frontend | Next.js pages + shadcn/ui + tRPC client | React 18 + Vite + TanStack Router/Query + Tailwind v4 |
| Fila de deploy | In-memory FIFO por grupo (`in-memory-queue.ts`, concurrency por server) | Polling SQL (`deployments.status='queued'`, ticker 3s, sem lock) |
| Workers dedicados | Sim — apps/schedules (BullMQ, 3 workers × 100) + apps/monitoring (Go) | Não — 1 worker de deploy + 1 worker de domínios in-process |
| Runtime de containers | Docker daemon (dockerode) + **Swarm services** | **podman CLI** via os/exec (+ shim docker→podman) |
| Multi-host | Sim — servers remotos via SSH (deploy/build separados) | Não — worker local único (`Deployment.ServerID` nunca setado) |
| Domínios/TLS | Traefik (file provider + docker/swarm provider), ACME, configs por app | Traefik file provider + worker de provisionamento — **mas ACME/HTTPS nunca ativa** (bug: `VerifyCertificate` sempre false, ver §20) |
| Secrets | Colunas `env` cifradas at-rest (`encryptedText`, AES-256-GCM — `packages/server/src/db/schema/application.ts:90` + `lib/encryption.ts`) | Env criptografado (AES-GCM) em `app_env`/`env_variables` — **paridade** (Dokploy cifra TODAS as colunas env; Aether só vars marcadas secret) |
| Monitoramento | apps/monitoring (Go, SQLite, séries temporais) + WSS stats + alertas de threshold | gopsutil + SSE + `podman stats` (sem persistência, sem evaluator) |
| Backup | Dumps reais (pg_dump etc.) + rclone S3 + restores + volume backups + schedules | Registro de rows; restore = no-op |
| Observabilidade | WSS: deploy logs (tail -f), container logs (search/since), terminal SSH, stats | SSE: logs de container, host stats, eventos; terminal pty (podman exec) |
| DB do produto | Postgres + SQLite (monitoring) | Postgres (+ Redis disponível via druntime, mas não usado) |

**Leitura**: o Dokploy é um orquestrador Docker Swarm multi-host com UI rica; o Aether é um orquestrador podman single-host com arquitetura limpa e tipos fortes. A estrutura do Aether (hexagonal, sqlc, testpool) é **mais fácil de testar e estender**; faltam os mecanismos de execução.

---

## 2.1 Riscos Arquiteturais — Performance, Segurança e Integridade

Análise dedicada do impacto da diferença arquitetural (worker único in-process + polling + podman CLI vs filas por grupo + Docker Swarm + SSH remoto).

### 2.1.1 Performance — risco BAIXO–MODERADO

| Ponto | Impacto | Evidência |
|---|---|---|
| Worker único serializa builds | 1 deploy por vez (loop sequencial em `processQueued`) — Dokploy também roda `buildsConcurrency=1` por default (**empate** no default) | `worker/worker.go:69-78` |
| Sem build server remoto | Dokploy permite offload de build (`buildServerId`); Aether não | `Deployment.ServerID` nunca setado |
| `checkHealth` bloqueia o loop | deploy lento (até 60s de retries) atrasa todos os outros | `worker/worker.go:175-202` |
| Polling 3s | +~3s de latência para iniciar deploy — irrelevante | `worker.go:56-67` |
| Builds não bloqueiam o HTTP | processos filhos (pack/nixpacks/podman) fora do servidor Gin | — |

**Conclusão**: a perda de performance é **paralelismo** (N builds simultâneos por partição), não latência nem throughput básico. No uso single-host atual é irrelevante; corrigível com goroutine por deploy + semáforo.

### 2.1.2 Segurança — PARIDADE (superfície até menor)

- **Acesso root equivalente**: podman CLI como root no container da API = acesso à VM host; o Dokploy expõe o socket do Docker (dockerode como root) — **mesmo risco**, inerente a todo PaaS orquestrador.
- **Superfície menor**: Aether **não tem SSH remoto** (sem chaves privadas, sem ssh2, sem execução remota) — o Dokploy adiciona essa superfície. Neste ponto o Aether é **mais seguro hoje**.
- Bugs pontuais (não arquiteturais): `InsecureSkipVerify` no terminal WS (`realtime/http/terminal.go:27`), API keys mortas, debug prints.
- **Conclusão**: a arquitetura não introduz risco de segurança; o risco é o mesmo de qualquer PaaS single-host (comprometimento = host inteiro).

### 2.1.3 Integridade — O RISCO REAL (remove-then-run vs update-or-create)

| Risco | Mecanismo | Consequência | Dokploy (comparação) |
|---|---|---|---|
| **Downtime em todo deploy** | `removeOldContainers` apaga o container antigo **antes** do `Run` do novo | janela de indisponibilidade a cada deploy | `service.update` do Swarm = rolling, zero-downtime |
| **Deploy falho derruba o app** | `fail()` remove o container do deploy que falhou; o antigo já foi removido | app fora do ar até um novo deploy | Swarm faz rollback automático da task anterior |
| **Sem recuperação de órfãos** | se a API crashar no meio do build, o deployment fica `building` para sempre; container velho segue rodando | estado inconsistente após crash | `initCancelDeployments` no startup (`cancel-deployments.ts`) |
| Colisão de porta/estado | sem lock por app na fila | dois deploys do mesmo app podem colidir | fila FIFO por grupo (`in-memory-queue.ts:48-53`) |
| Rollback frágil | re-usa `ImageRef` local; imagem pode já ter sido podada | rollback falha sem registry persistente | imagem empurrada ao registry (`rollbacks.ts` + `upload.ts`) |

**Conclusão**: o modelo atual é correto para o caso feliz e simples de operar; os riscos aparecem na **falha** — um deploy quebrado pode derrubar produção. É o único ponto onde a diferença arquitetural pode prejudicar disponibilidade e consistência.

### 2.1.4 Plano de mitigação (ordem de impacto)

| Fix | Esforço | Efeito |
|---|---|---|
| **Blue/green com swap de alias**: subir o novo container → healthcheck → só então remover o antigo | M | elimina downtime + app-down-on-failure |
| **Recuperação de órfãos no startup**: transicionar `building/starting` → `failed` + limpar containers | S | integridade sob crash |
| **Lock por app** na fila (deploys concorrentes do mesmo app) | S | evita colisão de porta/estado |
| **Goroutine por deploy** + semáforo (N builds) | S–M | desbloqueia paralelismo |
| **Registry local** para persistir imagens de deploy/rollback | M | durabilidade do rollback |

Os dois primeiros cabem no worker atual sem mudança de arquitetura e resolvem os problemas de maior impacto.

---

## 3. Inventário de Features (Dokploy) — com mecanismo

### 3.1 Organizações / Membros / RBAC
- Modelo: `organization`, `member` (role owner/admin/member/custom), `invitation` (48h), `organization_role` (custom roles enterprise), `audit_log`.
- ACL estático: `packages/server/src/lib/access-control.ts` — `statements` por resource/action; `checkPermission` (`services/permission.ts:76`), `checkServiceAccess` (`:263`), `checkProjectAccess` (`:218`), `checkEnvironmentAccess` (`:293`); member com arrays `accessedProjects/Environments/Services/GitProviders/Servers`.
- Invite: `organization.inviteMember` (`routers/organization.ts:259`) → email → aceite via better-auth; owner intransferível; admin não muda admin; ninguém muda o próprio role.
- **Aether**: `orgs` (CRUD + members + assignments + export/import YAML), roles simples (`auth/domain/entity.go:20-40`), audit de org. **Sem convites, sem RBAC por recurso, sem custom roles.**

### 3.2 Projects / Environments
- `project` (com `env` criptografado e `organizationId` CASCADE), `environment` (env, isDefault, CASCADE), `project_tag` N:N.
- `environment.duplicate` clona env + serviços; `project.duplicate` duplica tudo; `project.homeStats` (contadores por status); filtro por acesso do member em `project.all`.
- **Aether**: CRUD de projects/environments + env vars por escopo; **sem duplicate**, sem homeStats agregado (UNVERIFIED), sem filtro por member.

### 3.3 Services — tipos suportados
| Tipo | Tabela | Provisionamento |
|---|---|---|
| Application | `application` | Build (6 engines) → Swarm service |
| Compose | `compose` | `docker compose up -d --build` ou `docker stack deploy` |
| Postgres/MySQL/MariaDB/Mongo/Redis/Libsql | tabelas próprias | **Container real via dockerode** (`utils/databases/*.ts`): env vars, volume `{app}-data`, porta externa, healthcheck, restart policy, `changePassword` via exec |
| Preview | `preview_deployments` | appName `preview-*`, wildcard sslip, criado por webhook de PR |

### 3.4 Deployment pipeline (reconstruído)
```
tRPC (application.deploy/redeploy | compose.* | previewDeployment.deploy)
  → myQueue.add("deployments", job)            [self-hosted: in-memory; cloud: POST /deploy → Inngest]
  → processDeploymentJob (deployments-queue.ts:18)
  → deployApplication/rebuildApplication (services/application.ts:178)
      1. createDeployment → row running + logPath (/etc/dokploy/logs/<app>/...)
      2. monta script bash: set -e; <clone por provider>; <patches>; <getBuildCommand>
      3. execAsync(script >> logPath 2>&1)  [local ou SSH remoto]
      4. mechanizeDockerContainer → dockerode create/update Swarm service
      5. updateDeploymentStatus("done") + notificações; erro → status error + notificação
```
- Estados: `running|done|error|cancelled`. Concorrência: 1 job por grupo (mesmo serviço), N por partição (server). Cancelamento: remoção da fila (`removeWaiting`) + `killDockerBuild` (`pkill -2 -f "docker build"`).
- Rollback: `services/rollbacks.ts` — snapshot `fullContext` JSONB da application; `rollback()` re-cria serviço com imagem antiga do registry.
- Cleanup: `removeLastTenDeployments` (mantém 10), `clearOldDeployments`, docker cleanup agendado.

### 3.5 Build engines (Dokploy)
`getBuildCommand` (`utils/builders/index.ts:41-77`) dispatch por `buildType`:
| Engine | Comando | Extras |
|---|---|---|
| dockerfile | `docker build -f <Dockerfile> .` | `--target` (stage), `--no-cache`, `--build-arg` (resolvidos), `--secret type=env`, contexto de monorepo (`dockerContextPath`), `.env` via base64 |
| nixpacks | `nixpacks build <dir> --name <app>` | `--env`, `--no-cache`, `--no-error-without-start`; `publishDirectory` → docker cp + serve estático |
| railpack | `railpack prepare` → **buildx** com `BUILDKIT_SYNTAX=ghcr.io/railwayapp/railpack-frontend` | build via BuildKit |
| paketo | `pack build --builder paketobuildpacks/builder-jammy-full` | `--clear-cache`, `--env` |
| heroku_buildpacks | `pack build --builder heroku/builder:v24` | Procfile nativo |
| static | Gera Dockerfile nginx + nginx.conf SPA + .dockerignore | delega ao dockerfile |
| docker (sourceType) | `docker pull` | registry login |
| drop | upload ZIP → extração com checks de path-traversal | |

### 3.6 Git
- Providers OAuth/App: GitHub (App/JWT), GitLab/Gitea (OAuth refresh), Bitbucket (basic), git custom via SSH deploy keys (`ssh-key` service, `GIT_SSH_COMMAND`).
- Webhooks: `pages/api/deploy/github.ts` (push/tag/PR com previews + comentário no PR), `[refreshToken].ts` universal (gitlab/gitea/bitbucket/dockerhub/ghcr), `compose/[refreshToken].ts`. Skip keywords (`[skip ci]`), `watchPaths`, `autoDeploy`, `triggerType`.
- Patch system: tabela `patch` (create/update/delete de arquivos) aplicado antes do build (`generateApplyPatchesCommand`).

### 3.7 Domains / Networking / SSL
- `domain` (host, https, port, path, domainType, certificateType, middlewares, forwardAuth) + `certificate` (PEM custom) + `redirect` (regex→replacement) + `port` (publishMode ingress/host) + `network` (bridge/overlay custom) + `security` (basic auth via middleware Traefik bcrypt).
- Config Traefik dinâmica por app: `utils/traefik/application.ts` / `domain.ts` (routers web/websecure, stripPrefix, redirects, certresolver letsencrypt/custom, punycode IDN).
- ACME: `traefik-setup.ts` httpChallenge + acme.json; HTTP/3; previews wildcard `*.sslip.io`.

### 3.8 Variables / Secrets
- Escopos: `prepareEnvironmentVariables` (`utils/docker/utils.ts:399-450`) — merge org→project→environment→service com placeholders `${{project.X}}`/`${{environment.X}}`/`${{service.X}}` e erro em referência ausente. `buildArgs`/`buildSecrets`/`preview*` criptografados. **Sem encryption at-rest do env** (JSON em texto).

### 3.9 Databases — provisionamento real
- Cada engine tem `services/<db>.ts` + `utils/databases/<db>.ts`: create → INSERT + volume mount automático → deploy → `docker pull` + criação do Swarm service (env vars da engine, porta externa `PublishMode: host`, healthcheck, restart policy, limits) → status done/error. `changePassword` via `docker exec` + UPDATE. `rebuild` = remove + recria.

### 3.10 Backups / Schedules / Volume backups
- `backup` (cron, keepLatestCount, destinationId S3) → `utils/backups/<engine>.ts`: `docker exec pg_dump|mysqldump|mongodump` → **rclone rcat :s3:** ; rotate via `rclone lsf/delete`. Restore com subscription de logs. `volume_backup` (tar de volume + rclone, com `turnOff`). Schedules via `node-schedule` (self-hosted) ou apps/schedules (BullMQ).
- Docker cleanup diário (`CLEANUP_CRON_JOB`).

### 3.11 Monitoring / Notifications
- apps/monitoring (Go/Fiber): métricas de host + containers persistidas em SQLite, endpoints `/metrics`, `/metrics/containers`; alertas de threshold (cpu/mem) com `receiveNotification` webhook.
- Notificações: 12 canais (slack/telegram/discord/email/resend/gotify/ntfy/mattermost/custom/lark/pushover/teams) em eventos de deploy/backup/cleanup/restart/threshold.

### 3.12 WSS / Tempo real
- `/listen-deployment` (tail -f do logPath, local ou SSH), `/docker-container-logs` (tail/since/search), `/terminal` (SSH shell), `/docker-container-terminal` (pty docker exec), `/docker-stats`, `/drawer-logs` (tRPC subscriptions de deploy/backup/setup).

### 3.13 Enterprise (proprietary)
Audit logs, custom roles (RBAC), SSO (OIDC+SAML), SCIM, Forward Auth (SSO por domínio via Traefik), whitelabeling, license keys, Stripe billing, plan limits, impersonation, AI (analyzeLogs/suggest/deploy via Vercel AI SDK).

---

## 4. Matriz de Paridade (Aether)

Legenda: ✅ completo · 🟡 parcial · ❌ ausente · 🔴 falso positivo (parece, mas não funciona)

| Feature | Dokploy | Aether | Evidência Aether | Status |
|---|---|---|---|---|
| **Core** | | | | |
| CRUD Apps/Projects/Environments | ✅ | ✅ | `apps/http/handler.go:246,310,329` | ✅ |
| Deploy pipeline | ✅ | ✅ | `worker/worker.go:80-157` | 🟡 (sem cancel/timeout/remoto) |
| Estados de deploy | 4 | 8 | `deployments/domain/deployments.go:21-50` | ✅ superior |
| Healthcheck pós-deploy | 🟡 (Swarm cuida) | ✅ explícito HTTP | `worker.go:175-202` | ✅ superior |
| Cancelamento de deploy | ✅ | ❌ | só constante `StatusCancelled` | 🔴 |
| Rollback | ✅ (snapshot fullContext) | 🟡 (re-usa ImageRef) | `deployments/application/deployments.go:68-93` | 🟡 |
| **Build** | | | | |
| Dockerfile com --target/--build-arg/--secret | ✅ | ❌ | `worker.go:359-381` (sem flags) | ❌ |
| Nixpacks | ✅ | ✅ | `worker.go:344-357` | 🟡 (sem --env/--no-cache) |
| Paketo | ✅ (jammy-full) | ✅ (jammy-base/noble) | `buildSmartBuild` `worker.go:291-322` | 🟡 |
| Railpack/BuildKit | ✅ | ❌ (railpack pendura no podman) | — | ❌ |
| Heroku buildpacks/Procfile | ✅ | ❌ | — | ❌ |
| Build estático (nginx SPA) | ✅ (buildpack static) | ✅ (gerador Dockerfile) | `generateCommandDockerfile:544-584` | ✅ superior (não depende de buildpack) — **mas o gerador ignora start/install command: apps SSR não são deployáveis com build custom** (ver §20) |
| Monorepo (contexto/root folder) | ✅ | 🔴 | `RootFolder` persistido, não aplicado | 🔴 |
| Upload ZIP (drop) | ✅ | ✅ | `specs/http/handler.go:98` + `0005_app_upload` | ✅ |
| Upload imagem p/ registry | ✅ | ❌ | — | ❌ |
| **Docker/Infra** | | | | |
| Compose up/down + validação | ✅ | ✅ | `templates/application/compose.go:226-248,113-170` | ✅ |
| Swarm | ✅ | ❌ | — | ❌ |
| Servers remotos (SSH) | ✅ | ❌ | `ServerID` nunca setado | ❌ |
| Networks custom | ✅ | ❌ | — | ❌ |
| Ports extras (publishMode) | ✅ | ❌ | — | ❌ |
| Terminal exec | ✅ (pty) | ✅ (pty podman exec) | `realtime/http/terminal.go:16-86` | ✅ |
| **Domains/SSL** | | | | |
| Domínio + HTTPS + ACME | ✅ | 🔴 (HTTPS/ACME nunca ativa — bug `VerifyCertificate`) | `domains/application/provision.go:131-139` | 🔴 — config fica HTTP-only; ver §20 |
| Free domain | ✅ (traefik.me) | ✅ (`<slug>-<rand5>`) | `provision.go:38-42` | ✅ |
| Certificado custom (upload PEM) | ✅ | 🟡 | `certificates` endpoint existe | 🟡 |
| Redirects | ✅ | ❌ | — | ❌ |
| Basic auth (middleware) | ✅ | ❌ | — | ❌ |
| Forward auth SSO | ✅ (enterprise) | ❌ | — | ❌ |
| Preview wildcard | ✅ (sslip) | 🔴 (previews sem deploy) | `UpdatePreviewResult` sem chamador | 🔴 |
| **Variables/Secrets** | | | | |
| Escopos org/project/env/service | 🟡 (project+env+service) | ✅ (project+env+service) | `variables/application/resolver.go:46-108` | ✅ |
| Encryption at-rest | ✅ (todas as colunas env, AES-256-GCM) | ✅ AES-GCM (só vars secret) | `security/encrypt.go` + `packages/server/src/db/schema/utils.ts:12` | 🟡 paridade |
| Placeholders ${{...}} | ✅ | ✅ (+cycle guard) | `resolver.go:184-219` | ✅ |
| Build secrets (--secret) | ✅ | ❌ | — | ❌ |
| **Databases** | | | | |
| Provisionamento (container real) | ✅ | ❌ | `databases/application/databases.go:36-72` (só row) | 🔴 |
| Connection string exposta | ✅ | ❌ | `ConnectionString` sem endpoint | 🔴 |
| Change password via exec | ✅ | ❌ | — | ❌ |
| **Backups/Snapshots** | | | | |
| Dump real (pg_dump etc.) | ✅ | ❌ | `backups/application/backups.go:23-42` (só row) | 🔴 |
| Restore real | ✅ | 🔴 no-op 200 | `backups.go:51-78` | 🔴 |
| S3 destination (rclone) | ✅ | ❌ (só gdrive desconectado) | `storage/gdrive` sem imports | 🔴 |
| Volume backup | ✅ (tar+rclone) | 🔴 | `volumes/application/volumes.go:30` | 🔴 |
| Schedules de backup | ✅ (node-schedule/BullMQ) | 🔴 (sem runner) | `snapshots/application/snapshots.go:50-73` | 🔴 |
| **Cron/Jobs** | | | | |
| Cron jobs executados | ✅ (schedules app) | 🔴 CRUD-only | `jobs/application/jobs.go` (robfig só valida) | 🔴 |
| Workers com replicas | ✅ | 🔴 | `jobs.go:137-170` (replica ignorada) | 🔴 |
| **Monitoring/Alerts** | | | | |
| Container stats | ✅ | ✅ `podman stats` | `stats/application/stats.go:52-103` | ✅ |
| Host metrics | ✅ | ✅ gopsutil | `host/application/host.go:24-60` | ✅ |
| Séries temporais persistidas | ✅ | ❌ | — | ❌ |
| Alertas com evaluator | ✅ | 🔴 | `alerts/application/alerts.go` (sem loop) | 🔴 |
| Notificações (canais) | ✅ 12 canais | 🔴 (CRUD, sem envio) | `alerts.go:115-117` | 🔴 |
| **Auth/Org** | | | | |
| Login/sessão | ✅ | ✅ HMAC/JWT | `auth/infra/security.go:91-143` | ✅ |
| 2FA | ✅ (TOTP+passkeys) | 🟡 TOTP | `auth/http/handler.go:259-286` | 🟡 |
| API keys | ✅ | 🔴 (criadas, não verificadas) | `GetKeyByHash` sem chamador | 🔴 |
| Convites | ✅ | ❌ | — | ❌ |
| RBAC granular | ✅ (enterprise) | ❌ | — | ❌ |
| SSO OIDC | ✅ (enterprise) | ✅ OIDC | `settings/http/handler.go:136-207` | 🟡 |
| **Git** | | | | |
| Webhooks GitHub/GitLab/Bitbucket | ✅ | ✅ (inbound, HMAC) | `webhooks/application/providers.go` | ✅ |
| OAuth de providers (repo list/branches) | ✅ | ❌ | — | ❌ |
| Deploy keys SSH | ✅ | ❌ | — | ❌ |
| Patches em arquivos | ✅ | ❌ | — | ❌ |
| Previews de PR + comentário | ✅ | 🔴 | — | 🔴 |
| **Logs/Terminal** | | | | |
| Deploy log live | ✅ (WS tail -f) | 🟡 (arquivo + poll 2.5s) | `logs.go:20-47` + poll | 🟡 |
| Container logs search/since | ✅ | ❌ | `logs.go:100-146` (SSE, sem filtros) | ❌ |
| **UI/UX** | | | | |
| Command palette | ✅ ⌘J | ✅ (command-palette.tsx) | — | ✅ |
| Whitelabel | ✅ (enterprise) | 🟡 branding | `settings/http/handler.go:39-48` | 🟡 |
| Terminal na UI | ✅ | ✅ | Terminal.tsx | ✅ |

---

## 5. Gaps Backend (por prioridade)

### P0 — Critical
| Gap | O que falta | Evidência |
|---|---|---|
| **Provisionamento de databases** | worker/scheduler que faça `podman run` do container do banco (imagem, volume, porta, healthcheck), status real, connection string em endpoint | `databases/` não executa nada; `status="creating"` eterno |
| **Backups/restores reais** | dumps via `docker exec pg_dump`/`mongodump`, upload S3 (contrato já existe em `storage/contract.go`), restore real; parar de retornar 200 sem efeito | `backups.go:51-78` no-op; `storage/gdrive` desconectado |
| **Scheduler de cron/snapshots/mirrors** | runner in-process (ou druntime queue) que dispare `cron_jobs`, `snapshot_schedules`, `registry_mirrors.schedule`, jobs de backup; atualizar last_run/next_run | `main.go` só sobe worker de deploy + domínios |
| **Cancelamento de deploy** | endpoint que remova o job da fila (update status) + kill de build/container em andamento | só a constante existe |
| **API keys funcionais** | middleware valida `aether_*` contra `api_keys` (hash já existe), touch last_used_at | `GetAPIKeyByHash` sem chamadores |
| **Previews de PR** | webhook de PR cria preview + worker deploya com domínio wildcard | `UpdatePreviewResult` sem chamador |

### P1 — High
| Gap | O que falta |
|---|---|
| RBAC granular (roles por recurso, accessedServices) | service de permissões + filtro em listas |
| Convites de org (invitation + aceite + email) | tabela + fluxo |
| Build args / --target / build secrets no Dockerfile | flags no `buildDockerfile` |
| Monorepo: aplicar `RootFolder` no build | usar rootFolder no context/build |
| Outbound webhooks (dispatch de eventos de deploy/alerts) | chamar `Webhooks.Deliver` nos eventos |
| Alert engine (evaluator de regras + envio a canais) | loop de avaliação sobre stats |
| Autopilot funcional (avaliar policy e escalar) | evaluator + `CreateAutopilotEvent` |
| GitOps Sync real (clonar, ler aether.yml, drift, aplicar) | reescrever `gitops.go` |
| Pipelines assíncronos (fila + triggers auto/webhook) | mover run para worker |
| Notificações multi-canal (slack/telegram/email) | dispatch real |
| Concorrência no worker (1 deploy por app) + retry/backoff | lock na fila + re-queue |
| **ACME/HTTPS ativo** (bug descoberto na 2ª passada) | `VerifyCertificate` sempre false (`TraefikBin` não setado no install.sh + `wget -q` sem "200"); corrigir detecção ou verificar via `traefik exec`; sem isso toda config de domínio fica HTTP-only (`domains/application/provision.go:131-139`) |
| **Build "custom" com servidor (SSR)** | `generateCommandDockerfile` sempre gera nginx estático e ignora `start_command`/`install_command` (`worker/worker.go:544-584`) — Next/Nuxt (detectados como SSR pelo planner) não são deployáveis com o build default; gerar Dockerfile servidor quando houver start command |
| **Callback OIDC quebrado** | `redirect_uri` é GET (`settings/infra/discovery.go:126-131`) mas a rota é POST (`api/router.go:127`) → 404 com IdP real; `state` não validado no callback |
| **Compose: secrets criptografados no `.env`** | `writeEnvFile` usa valores crus do store (ciphertext p/ secrets) → `.env` do stack corrompido (`templates/application/compose.go:253-278`) |
| **DeleteProject soft-delete incompleto** | `UPDATE projects SET deleted_at` sem cascata; apps/containers seguem vivos e visíveis (`api/db/queries/projects.sql:23-26` + `apps.sql:57-65`) |
| **Webhooks inbound aceitam request sem assinatura** | verificação HMAC só roda se o header existir (`webhooks/application/providers.go:54,78,125`) — rota pública = deploy anônimo; exigir assinatura |
| **Org export/import corrompe secrets** | export grava ciphertext e import re-insere como não-secret (`orgs/application/export.go:92-97,170-177`) |
| **Master key com fallback inseguro** | `"dev-secret-please-override"` hardcoded se `AETHER_API_SECRET`/keys/master.key ausentes (`api/cmd/api/main.go:332`); CORS `*` (`main.go:286`) |

### P2 — Medium
| Gap |
|---|
| Servers remotos (SSH) / agente (CA existe mas órfã) |
| Networks custom + ports extras (publishMode) |
| Redirects + basic-auth (middlewares Traefik) |
| Custom certificates (upload PEM) funcional |
| Container logs com search/since + WS stats streaming |
| Previews de upload: `Detect`/`planForApp` aceitar `upload_id` |
| `?secrets=1` respeitado; DTO stubs (`server_id`, `cluster_id`, `volumes`) |
| Workers de jobs com replicas reais; pipelines com timeout |
| host logs/events reais (escrever aether.log) |
| druntime: ativar adapter redis (queue/locks/ratelimit) |
| PATCH de domínio não re-provisiona config Traefik (`domains/infra/store.go:69-74`) |
| Compose import é casca (valida YAML e retorna ok sem criar — `templates/http/compose.go:135-147`) |
| Mirror ignora `TagsFilter` no Run (`mirrors/application/mirrors.go:53-67`) |
| `DeleteEnvironment` do default retorna 404 silencioso; `UpdateApp` aceita env de outro projeto (`apps.go:147-149`) |
| Config órfãs: `AETHER_PROXY_ENDPOINT`/`AETHER_CHALLENGE_ADDR`/`AETHER_AGENT_ADDR` definidas e nunca usadas (`config/config.go:92-94`); `AlertIntervalSeconds` nunca setado |

### P3 — Low
| Gap |
|---|
| Swarm (provavelmente fora do escopo — é podman) |
| GPU setup, HTTP/3, IDN/punycode |
| i18n, impersonation, billing/Stripe |
| AI (analyzeLogs) |
| Debug prints em produção (`auth/http/handler.go:71,74`; `auth/application/auth.go:72,83`); commit SHA do webhook GitHub não propagado ao deploy (`providers.go:66`); `setWebhook`/`TouchAPIKey` órfãos |
| Debug prints em produção (`auth/http/handler.go:71,74`) |

---

## 6. Gaps Frontend

| Área | Estado | Gaps |
|---|---|---|
| Wizards de criação | ✅ fortes (ApplicationWizard 3 passos, CreateServiceModal, Compose/Template/Database) | — |
| Deploy log live | 🟡 | trocar poll 2.5s por SSE/WS dedicado (`logs.go` já tem SSE AppLogs) |
| Tabela central de deployments/fila | 🟡 | sem fila visível, sem cancelar/kill |
| Settings de services | 🟡 | faltam abas equivalentes: domains avançados, redirects, security/basic-auth, patches, ports extras, rollbacks, schedules, volume-backups |
| Databases | 🔴 | UI existe (DatabaseWizard, databases.*) mas o backend não provisiona — UX mente |
| Backups/Snapshots | 🔴 | UI de criar/restaurar mostra sucesso sem efeito |
| Command palette | ✅ | — |
| Terminal | ✅ | — |
| Monitoring | 🟡 | stats por container OK; sem histórico/séries |

**Falsos positivos de UI**: databases (cria e nunca sobe), backups (restaurar = 200 falso), schedules/snapshots (CRUD sem execução), API keys (tela existe, chave inútil), previews (cria registro, nada acontece).

---

## 7. Gaps Infraestrutura

| Item | Estado | Gap |
|---|---|---|
| Traefik ingress + ACME | ✅ (infra do ingress; **ACME inativo por bug — ver §20**) | — |
| Build multi-arch (nixpacks/railpack/pack no Dockerfile) | ✅ | railpack inútil no podman (hang de export); paketo SPA cai no fallback nginx (documentado) |
| Podman socket no path da VM | ✅ (fix aplicado) | — |
| S3 destination | ❌ | infra de rclone/destination não existe |
| Servers remotos | ❌ | sem SSH |
| Registry local | ❌ | imagens ficam só no podman local (**mas há endpoints funcionais de listagem/deleção** — `GET /registry/images`, `DELETE /registry/images/:repo/:tag`, `api/internal/modules/clusters/application/clusters.go:110-144`) |
| Volumes de dados de apps | 🟡 | mounts existem no modelo, criação sem endpoint |
| Backup externo do estado | 🔴 | snapshot é row apenas |
| Testes | ✅ | testpool real (postgres 5433 / redis 6380) — superior ao Dokploy (jest com mocks em parte) |

---

## 8. Gaps Banco de Dados

| Tabela Aether | Estado |
|---|---|
| `apps`, `projects`, `environments`, `deployments`, `domains`, `env_variables`, `previews`, `webhooks`, `settings`, `branding`, `oidc_providers`, `s3_destinations`, `compose_apps`, `templates` | ✅ em uso |
| `variable_audit` | ✅ em uso (auditoria de env vars — não mapeado antes) |
| `databases` | 🟡 sem fluxo (status creating) |
| `backups`, `snapshots`, `snapshot_schedules` | 🟡 sem execução |
| `cron_jobs`, `workers`, `app_policies`, `autopilot_events` | 🟡 sem scheduler/evaluator |
| `alert_rules`, `alert_events`, `notifications`, `notification_channels` | 🟡 sem evaluator/dispatch |
| `api_keys` | 🟡 sem verificação |
| `app_volumes` | 🟡 sem criação |
| `gitops` | 🟡 record-only |
| `pipelines`, `pipeline_runs` | 🟡 síncrono |
| `clusters`, `servers`, `server_tokens`, `registry_settings` | 🟡 sem agente |
| `mirrors` | 🟡 sem scheduler, sem org scoping |
| `out_webhooks` | 🟡 sem dispatch |

**Não existem**: `invitation`, `organization_role` (custom roles), `certificate` (custom PEM), `redirect`, `port` (extras), `network` (custom), `security` (basic auth), `patch`, `rollback` (snapshot de contexto), `volume_backup`, `destination` (S3), `schedule` (executável), `registry`, `ssh-key`, `git_provider`, `ai`, `audit_log` (org-level existe via auth.Audit), `sso_provider` (OIDC existe em settings), `forward_auth_settings`, `scim`.

---

## 9. Análise do Build System (aprofundada)

| Capacidade | Dokploy | Aether | Cobertura |
|---|---|---|---|
| Dockerfile | `--target/--build-arg/--secret/--no-cache/context` | `podman build -f --pull` | 40% |
| Nixpacks | `--env/--no-cache/--no-error-without-start` + publishDirectory | `nixpacks build --name` | 50% |
| Paketo | jammy-full, `--clear-cache` | jammy-base/noble por arch, `--platform`, fallback SPA nginx | 70% (falha conhecida do export com nginx no podman — podman#8132) |
| Railpack | buildx + BuildKit | ❌ (hang no export via BuildKit) | 0% no podman |
| Heroku/Procfile | ✅ | ❌ | 0% |
| Static/nginx SPA | buildpack static | gerador próprio de Dockerfile+nginx (independente de buildpack) | ✅ superior |
| Build env/secrets | `.env` + `--secret type=env` | `.env` (writeBuildEnv) | 50% |
| Cache | `cleanCache` por build + docker buildx cache | cache nativo podman | 60% |
| Monorepo | `buildPath`/`dockerContextPath` | `RootFolder` não aplicado | 0% |

---

## 10. Networking Analysis

| Capacidade | Aether | Dokploy |
|---|---|---|
| HTTP/HTTPS + ACME via Traefik | 🔴 (ACME nunca ativa — ver §20) | ✅ |
| Free domain | ✅ próprio | ✅ traefik.me |
| Custom domain + path | ✅ (Path/StripPath) | ✅ (+InternalPath) |
| Redirects | ❌ | ✅ |
| Basic auth por app | ❌ | ✅ |
| Forward auth SSO | ❌ | ✅ (ent.) |
| Custom cert | 🟡 | ✅ |
| Ports extras TCP/UDP | ❌ | ✅ |
| Networks custom bridge/overlay | ❌ | ✅ |
| Wildcard preview | ❌ | ✅ |
| Middlewares custom Traefik | 🟡 (config raw?) | ✅ |
| Multi-tenant isolates | ✅ (por org) | ✅ (por org + member access) |

---

## 11. Security Analysis

**Aether é superior em**: validação de upload ZIP (path traversal checks), HMAC em webhooks inbound (quando o header existe — ver falha abaixo), sanitização de shell, org scoping em todas as rotas, 2FA TOTP.

**Criptografia at-rest — PARIDADE (correção da 2ª passada)**: o Dokploy **também cifra** todas as colunas `env`/`buildArgs`/`buildSecrets`/`preview*` via `encryptedText` (AES-256-GCM, chave derivada de `ENCRYPTION_KEY ?? BETTER_AUTH_SECRET` — `packages/server/src/db/schema/utils.ts:12`, `lib/encryption.ts:22,76`), inclusive em compose/environment/databases. O Aether cifra apenas vars marcadas como secret (`IsSecret`); vars comuns ficam em texto. Resultado: empate técnico com nuances diferentes (Dokploy cifra tudo; Aether cifra só secrets).

**Aether falha em** (incl. achados da 2ª passada): API keys inúteis (superfície de segurança falsa), `InsecureSkipVerify` no terminal WS (`api/internal/modules/realtime/http/terminal.go:27`), debug prints em produção, logout sem revogação server-side, DTOs vazando stubs, rotas públicas globais de SSO sem org, mirrors sem org scoping, worker sem lock (colisão de porta), `TraefikBin` local em VerifyCertificate, **master key com fallback `"dev-secret-please-override"`** (`api/cmd/api/main.go:332`), **CORS `*`** (`api/internal/platform/api/router.go`), **webhooks inbound aceitam request sem assinatura** (`api/internal/modules/webhooks/application/providers.go:54,78,125`), **org export/import vaza ciphertext como secret** (`api/internal/modules/orgs/application/export.go:92-97`), **callback OIDC sem validação de state**.

**Dokploy**: melhor em RBAC granular, deploy keys, patches, validação anti-injeção em comandos (shell-quote), sanitizeCommand em compose, senhas de DB com regex anti-shell. **Nota (2ª passada)**: o `fullContext` de rollback grava **credenciais de registry em texto plano** no JSONB (`packages/server/src/services/rollbacks.ts:74-79`) — achado de segurança do lado do Dokploy.

---

## 12. UX Analysis

| Aspecto | Aether | Dokploy |
|---|---|---|
| Design system | ✅ tokens Material3 dark, consistente | shadcn/ui (também consistente) |
| Wizards | ✅ 3 passos com auto-detecção | dialogs por tipo |
| Deploy log | 🟡 poll 2.5s | ✅ WS realtime |
| Terminal | ✅ | ✅ |
| Empty/loading/error states | ✅ (AppEmptyState/Skeleton) | ✅ (AlertBlock/skeletons) |
| Command palette | ✅ | ✅ |
| Onboarding/setup | 🟡 | ✅ (wizard de server com logs streaming) |
| Whitelabel | 🟡 branding básico | ✅ enterprise (CSS global, título, favicon) |
| Feedback enganoso | 🔴 (restore/snapshot/backup "ok" sem efeito) | — |

---

## 13. Hidden Features (Aether — não documentadas)

- `planner` + `specs`: motor de detecção de framework e geração de Dockerfile/nginx com preview no wizard — **único no nosso produto, mais transparente que o Dokploy**.
- `ExportRuntime` (compose/k8s/nomad) + `Compare` (diff de deployments) + `SystemSummary`.
- `settings`: SSO OIDC por org com discovery (recurso que o Dokploy vende como enterprise) — *mas com o bug do callback GET×POST (ver §20)*.
- `orgs.Export/Import` (YAML de projetos/apps/dbs) — *mas corrompe secrets (ver §20)*.
- `webhooks`: inbound GitHub/GitLab/Bitbucket com verificação de assinatura (quando o header existe).
- `druntime`: fila/locks/ratelimit com adapter redis pronto (órfão, mas disponível).
- `storage/gdrive`: provider completo (órfão).
- `security/ca.go`: CA + assinatura de agente (órfão, base para multi-server).
- `variables` resolver com cycle-guard.
- **Funcionais e não mapeados antes (2ª passada)**: `GET /api/v1/health` + `/ready` (pool ping); `GET /auth/status`; SSO público de login (`/sso/public` + auth-url); `GET /apps/:id/variables/effective`; timeline de deploys (`/apps/:id/timeline`); estados agregados + SSE (`/apps/states` + `/stream`); cron jobs globais (`/cron-jobs`); registry images reais (`/registry/images` + delete, via `podman images/rmi`); variables audit/export/import (project + environment, tabela `variable_audit`); presence (`/presence/*`) + network quality (`/network/quality` — probe HTTP p50/p95/uptime/HTTP3); runtime metrics (`/runtime/metrics`); export de compose de app/deploy (`/apps/:id/compose`, `.../deployments/:depID/compose`); mirror run manual; workers start/stop (podman run real).

---

## 14. False Positives (parecem implementadas, não estão)

| Feature | Parece | Realidade |
|---|---|---|
| Databases | CRUD + UI | nenhum container criado |
| Backups/Snapshots | criar/restaurar com sucesso | rows apenas; restore no-op |
| API keys | tela + geração | nunca verificadas |
| Previews | registro criado | nenhum deploy |
| Cron jobs / workers | CRUD + start | sem scheduler; replicas ignoradas |
| Autopilot | policy salvo | sem avaliador |
| Alerts | regras + canais | sem evaluator, sem envio |
| Outbound webhooks | CRUD | sem dispatch |
| GitOps | sync | record-only |
| Pipelines | run | síncrono, sem triggers |
| Snapshots schedules | CRUD | nunca rodam |
| Volumes | backup | sem criação de volume |
| Storage/GDrive | provider completo | desconectado |
| Clusters/Servers | tokens | sem agente/heartbeat |
| Host logs/events | endpoints | arquivo nunca escrito |
| Metrics | endpoint | `"total": 0` hardcoded |
| **Compose import** | importa YAML | valida e retorna `{"ok": true}` sem criar nada (`templates/http/compose.go:135-147`) |
| **Database stats/logs** | abas na UI | dependem de `ContainerID` que nunca é setado — sempre vazios/404 |
| **OIDC SSO** | fluxo configurado | callback quebrado (GET×POST) — login via IdP nunca completa |
| **Registry settings** | config de registry | sem container de registry (status sempre "stopped") |
| **`cert_status` dos domínios** | "pending" com retries | ACME nunca dispara — morre em silêncio após 10 tentativas |

---

## 15. Our Advantages (o que já é superior)

1. **Máquina de estados de deploy** (8 estados, transições validadas vs 4 do Dokploy).
2. **Healthcheck HTTP explícito** pós-deploy (Dokploy delega ao Swarm).
3. **Gerador de Dockerfile/nginx próprio** para builds custom/SPA — independe de buildpack e funciona no podman (o static buildpack do Dokploy é o equivalente, mas dele depende do fluxo).
4. **Análise de fonte com preview** (specs/planner: detecta framework, gera Dockerfile/nginx.conf, mostra no wizard).
5. **Arquitetura hexagonal + sqlc + testpool real** (testes com postgres/redis reais) — mais testável que o monorepo tRPC do Dokploy.
6. **Terminal e SSE próprios** sem dependência de Docker API.
7. **SSO OIDC por org** (no Dokploy é enterprise) — *uma vez corrigido o callback*.
8. **Upload ZIP com validação de path-traversal** e detecção automática.
9. **Ciclo de dev rápido**: hot reload (`air` + Vite), imagem multi-arch, dev containers isolados.
10. **Criptografia at-rest**: paridade com o Dokploy (que também cifra env via `encryptedText`) — Aether cifra secrets, Dokploy cifra todas as colunas env; não é mais diferencial, é empate.

---

## 16. Dependency Graph (gaps)

```
Git Integration (OAuth providers, deploy keys)
      ↓
Source (monorepo RootFolder, patches)
      ↓
Build Engine (build args/secrets, --target, registry push)
      ↓
Image Registry (local registry / upload)
      ↓
Deployment (cancelamento, concorrência, timeout, retry, multi-server)
      ↓
Container Runtime (networks custom, ports extras)
      ↓
Domain/SSL (redirects, basic-auth, custom certs, forward-auth)
      ↓
Observability (logs search, alert engine, notifications)

Scheduler (cron jobs, snapshots, mirrors, backups)
      ↓
Backups/Snapshots (dumps reais, S3 destination, restores)
      ↓
Databases (provisionamento) ── depende do Container Runtime ──┐
      └──────────────────────────────────────────────────────────┘

RBAC granular + invitations  →  multi-tenant seguro
API keys funcionais          →  integrações externas
Previews de PR               →  depende de Git + Domains(wildcard) + Deploy
```

**Ordem recomendada de implementação:**
1. **Fundação**: cancelamento/concorrência do worker; scheduler genérico; API keys.
2. **DBs**: provisionamento (desbloqueia stats, logs, backups de DB).
3. **Backups reais** (dumps + S3) + restores.
4. **Cron/agendamentos** (jobs, snapshots, mirrors).
5. **Build avançado** (args, secrets, --target, monorepo).
6. **Git avançado** (OAuth providers, deploy keys, patches, previews).
7. **RBAC + convites**.
8. **Observabilidade** (alert engine + canais de notificação).
9. **Domínios avançados** (redirects, basic auth, custom certs).
10. **Multi-server** (agente/SSH) — o maior salto.

---

## 17. Roadmap de Paridade

**Phase 0 — Foundation**
- Cancelamento de deploy + lock por app no worker + retry/backoff.
- API keys funcionais (middleware).
- Scheduler genérico in-process (cron de jobs, snapshots, backups, mirrors) ou druntime-queue.
- Parar de reportar sucesso sem efeito (restores/gitops).
- **Corrigir bugs de fluxo real (2ª passada)**: ACME/`VerifyCertificate`; build custom com suporte a start/install (SSR); callback OIDC; secrets no compose `.env` e no org export/import; webhooks exigindo assinatura; master key sem fallback inseguro.

**Phase 1 — Critical Parity (P0)**
- Provisionamento de databases (postgres/mysql/mariadb/redis/mongo) + connection string + stats/logs.
- Backups reais (dumps via docker exec) + S3 destination + restores.
- Previews de PR (webhook + deploy + wildcard).

**Phase 2 — Core Parity (P1)**
- Build args/secrets/--target/monorepo; outbound webhooks; alert engine; autopilot; pipelines assíncronos; notificações; gitops real; RBAC granular; convites.

**Phase 3 — Advanced (P2)**
- Servers remotos/agente; networks/ports custom; redirects/basic-auth; custom certs; logs search/since; workers com replicas.

**Phase 4 — Differentiation**
- Manter o que já é superior: secrets cifrados, gerador Dockerfile/nginx, machine de estados, planner com preview, arquitetura testável.
- Adicionar: AI (analyzeLogs), whitelabel avançado, auditoria completa, CLI.

---

## 18. Implementation Specs — P0/P1 (resumo técnico)

### P0-1: Provisionamento de Databases
- **Como o Dokploy resolve**: `services/postgres.ts` + `utils/databases/postgres.ts` — deploy cria Swarm service com env da engine, volume `{app}-data`, porta externa, healthcheck, limits; `changePassword` via docker exec.
- **Como resolver no Aether**: worker/job que, ao criar database (status `creating`), executa `podman run -d --name aether-db-<id8> -e <ENGINE_ENV> -p 127.0.0.1:<port>:<containerPort> -v <volume> <image>`; persistir `container_id`, `status`, porta; endpoints `GET /databases/:id/connection-string`, `PATCH /databases/:id/password` (exec + update), stats/logs.
- **Backend**: `api/internal/modules/databases/application` + novo worker; **Frontend**: mostrar DSN + start/stop/restart; **Infra**: nada novo (podman).
- **Complexidade**: M.

### P0-2: Backups reais + S3
- **Dokploy**: `utils/backups/postgres.ts` → `docker exec pg_dump | gzip | rclone rcat :s3:` + rotate `rclone delete`; restore invertido; destination S3 com testConnection.
- **Aether**: implementar runner em Go: `podman exec <db> pg_dump` → stream → `storage` provider (implementar S3, o contrato existe em `storage/contract.go`); `RestoreDatabase` real; schedules via o scheduler da Phase 0.
- **Complexidade**: M (S3 provider novo).

### P0-3: API keys
- **Dokploy**: tabela `apikey` com rate limit + permissions, middleware better-auth.
- **Aether**: no middleware (`auth/http/middleware.go`), se header `Authorization: Bearer aether_...`, consultar `GetAPIKeyByHash` (já existe a query), validar e tocar `last_used_at`.
- **Complexidade**: S.

### P0-4: Cancelamento de deploy
- **Dokploy**: fila in-memory com `removeWaiting` + `killDockerBuild` (`pkill -2 -f "docker build"`).
- **Aether**: endpoint `POST /deployments/:id/cancel` → se `queued`, transição para `cancelled`; se `building`, cancelar ctx do build (o worker já usa `CommandContext`) + matar container se criado.
- **Complexidade**: M.

### P1-1: Scheduler genérico
- Runner in-process (ticker 30s) que resolve due-jobs: `cron_jobs`, `snapshot_schedules`, `registry_mirrors`, backups; atualizar `last_run`/`next_run`; executar com timeout e log.
- **Complexidade**: M.

### P1-2: Build avançado
- `buildDockerfile`: `--build-arg` (variáveis não-secret), `--target` (campo Dockerfile já existe), `--secret` para secrets; aplicar `RootFolder` (cd no subdir) para dockerfile/nixpacks.
- **Complexidade**: S–M.

### P1-3: RBAC granular
- Modelo `member` com flags/arrays de acesso (espelhar Dokploy `member.accessedServices` etc.) + `checkServiceAccess` nas rotas + filtro em listas.
- **Complexidade**: M.

---

## 19. Final Assessment

1. **Percentual aproximado de paridade**: **~35–40%** (núcleo deploy/auth/variables em ~70–90%; **domínios/ACME caem para ~30%** após o bug do `VerifyCertificate`; databases/backups/cron/monitoring em 0–35%; UI em ~60%).
2. **Maiores gaps**: provisionamento de databases; backups/restores reais (hoje mentem 200); scheduler de cron; cancelamento de deploy; API keys; previews; RBAC/convites; **+ correções P1 descobertas na 2ª passada**: ACME/HTTPS nunca ativa, build custom sem suporte a SSR, callback OIDC quebrado, secrets corrompidos no compose e no org export/import (ver §5 e §20).
3. **Maiores riscos**: (a) **confiança enganosa** — módulos retornam sucesso sem efeito; (b) **scaffold sem executor** se espalhando; (c) **integridade do deploy** — modelo remove-then-run gera downtime em todo deploy, deploy falho derruba o app e não há recuperação de órfãos após crash (ver seção 2.1.3); (d) dependências de CLIs externas (nixpacks/pack) sem fallback; (e) **HTTPS/ACME não funcionando** no principal fluxo de domínios.
4. **O que já está melhor que o Dokploy**: máquina de estados de deploy; healthcheck explícito; gerador Dockerfile/nginx próprio; planner com preview; SSO OIDC por org (após corrigir o callback); arquitetura hexagonal testável com testpool real; hot reload dev. *(Criptografia at-rest: agora é paridade, não vantagem — o Dokploy também cifra env.)*
5. **O que implementar primeiro**: cancelamento/concorrência do worker → **corrigir ACME** → databases → backups reais → scheduler → API keys.
6. **Partes da arquitetura a alterar**: (a) worker único → worker com fila por app (lock) + scheduler; (b) druntime memory → redis (queue/locks reais); (c) backups/snapshots → executores reais; (d) middleware de auth → API keys; (e) DTOs stubs → dados reais; (f) `VerifyCertificate`/provisionamento de domínios (detecção de certificado via `traefik exec` em vez de `wget`+binário).
7. **Features que NÃO valem a pena copiar**: Swarm (podman single-host é o modelo); Stripe/billing (fora do escopo self-hosted); railpack/BuildKit no podman (quebrado no nosso ambiente — documentado); SCIM (sem market).
8. **Features a implementar de forma diferente**: previews (usar nosso domínio free em vez de sslip); backups (Go + S3 provider próprio em vez de rclone via shell); monitoramento (SSE + nosso storage em vez de app Go separado); notificações (reutilizar nosso `notification_channels` com dispatch real).
9. **Onde criar vantagem competitiva**: (a) preview de Dockerfile/nginx gerado no wizard (transparência de build); (b) auditoria + ciclo de vida de variáveis (audit/export/import já existem); (c) DX: dev loop hot-reload + testes reais; (d) planner/export (compose/k8s/nomad) — ninguém mais tem; (e) terminal/SSE nativos sem Docker.
10. **Caminho mais eficiente para paridade funcional**: seguir a ordem da seção 17 (Foundation → P0 → P1), atacando primeiro os falsos positivos (fazer o que já parece feito) e os bugs de fluxo real (ACME, SSR, OIDC, secrets), depois os gaps de competitividade (databases, backups, cron, RBAC). Estimar: **6–10 semanas** para P0 completo (1 dev), **12–16 semanas** para P1, com o worker/scheduler como pré-requisito de quase tudo.

---

*Relatório gerado por auditoria estática. Itens sem evidência direta: marcados UNVERIFIED. Precisão > velocidade.*

---

## 20. Double-check — Segunda Passada (verificação linha a linha)

Passada de verificação sobre o relatório original (método: seguir fluxo real endpoint → application → store → sql → worker/runtime em ambos os codebases). Resultado: **2 correções de claims do relatório, 10+ bugs novos no Aether e 8 refinamentos no Dokploy**.

### 20.1 Correções ao relatório original (claims ERRADOS)

| # | Claim original | Correção | Evidência |
|---|---|---|---|
| C1 | "Dokploy não cifra env at-rest; Aether é superior" | **ERRADO**: o Dokploy cifra TODAS as colunas env via `encryptedText` (AES-256-GCM, chave de `ENCRYPTION_KEY ?? BETTER_AUTH_SECRET`): `application.env/previewEnv/buildArgs/buildSecrets`, `compose.env`, `environment.env`, env de todos os databases. Paridade (nuance: Dokploy cifra tudo; Aether só `IsSecret`) | `packages/server/src/db/schema/application.ts:90-114`, `schema/utils.ts:12`, `lib/encryption.ts:22,76` |
| C2 | "Domínio + HTTPS + ACME ✅ implementado" | **ERRADO**: o fluxo existe, mas **ACME nunca ativa** — `VerifyCertificate` sempre false (`TraefikBin` default `""` e `install.sh` não seta `AETHER_TRAEFIK_BIN`; além disso `wget -q` não imprime "200" em sucesso → `strings.Contains` sempre false). O worker escreve config HTTP-only, `cert_status` fica pending e os retries morrem em silêncio após 10 tentativas | `api/internal/platform/config/config.go:95`, `api/internal/modules/domains/application/provision.go:131-139`, `worker.go:11,60-67`, `install.sh:359-366` |

### 20.2 Bugs novos no Aether (não estavam no relatório)

| # | Bug | Impacto | Evidência |
|---|---|---|---|
| B1 | **Build "custom" ignora `start_command`/`install_command`** — `generateCommandDockerfile` sempre gera nginx estático | Apps SSR (Next/Nuxt — o planner detecta como `TypeSSR`) não são deployáveis com o build default | `api/internal/platform/worker/worker.go:544-584` vs `api/internal/platform/planner/planner.go:108-116` |
| B2 | **VerifyDomain não faz verificação DNS** e **PATCH de domínio não re-provisiona** (config Traefik antiga permanece) | Domínios editados ficam com roteamento velho até reprovision manual | `domains/http/handler.go:112-128`, `domains/application/domains.go:128-135`, `domains/infra/store.go:69-74` |
| B3 | **Callback OIDC quebrado**: `redirect_uri` GET vs rota POST + `state` não validado | Login via IdP real nunca completa | `settings/infra/discovery.go:126-131` vs `api/internal/platform/api/router.go:127`; `settings/http/handler.go:207-233` |
| B4 | **Compose `.env` recebe secrets criptografados** (ciphertext) | Secrets quebrados em stacks compose | `templates/application/compose.go:253-278` + `variables/infra/store.go:61-77` |
| B5 | **DeleteProject soft-delete sem cascata** — apps/containers vivos e ainda listados | Dados órfãos; `GET /apps` mostra apps de projetos "deletados" | `api/db/queries/projects.sql:23-26` + `api/db/queries/apps.sql:57-65` |
| B6 | **Webhooks inbound aceitam request sem assinatura** (verificação HMAC condicional) | Deploy anônimo via rota pública | `webhooks/application/providers.go:54,78,125` |
| B7 | **Org export/import corrompe secrets** (ciphertext exportado, reimportado como não-secret) | Secrets inutilizáveis após migração | `orgs/application/export.go:92-97,170-177` |
| B8 | **Master key com fallback hardcoded `"dev-secret-please-override"`** + **CORS `*`** | Risco de segurança em setup sem env | `api/cmd/api/main.go:332,286` |
| B9 | **DeleteEnvironment do env default** → 404 silencioso; **UpdateApp aceita env de outro projeto** | Erros confusos / dados cruzados | `api/db/queries/environments.sql:30-33`, `api/internal/modules/apps/application/apps.go:147-149` |
| B10 | **Compose import é casca** (valida YAML, retorna ok, não cria nada); **mirror ignora `TagsFilter`**; **commit SHA do webhook não propagado**; `TouchAPIKey`/`SetWebhook` sem uso; configs `AETHER_PROXY_ENDPOINT`/`CHALLENGE`/`AGENT` órfãs; `AlertIntervalSeconds` nunca setado | Superfície morta | `templates/http/compose.go:135-147`, `mirrors/application/mirrors.go:53-67`, `webhooks/application/providers.go:66`, `config/config.go:19,92-94` |

### 20.3 Refinamentos nas claims do Dokploy

| # | Claim original | Refinamento | Evidência |
|---|---|---|---|
| D1 | "env não criptografado" | ver C1 — é criptografado | — |
| D2 | "kill = pkill -2 -f docker build" | também `pkill -2 -f "docker compose"` para compose | `apps/dokploy/server/queues/queueSetup.ts:129,137` |
| D3 | "concurrency 1 por grupo" | modelo real: **concurrency por partição (server)**, FIFO por grupo dentro da partição (`activeGroups`) | `in-memory-queue.ts:221-243`, `concurrency.ts:20,27` |
| D4 | "createRollback faz upload" | o upload (tag+push) só roda se `rollbackActive && rollbackRegistry`, via `uploadImageRemoteCommand` dentro de `getBuildCommand` | `utils/cluster/upload.ts:52-72,134-143`, `utils/builders/index.ts:68-74` |
| D5 | "rollback usa imagem do registry" | sem `rollbackRegistry`, usa a tag **local** `{app}:v{N}` | `services/rollbacks.ts:162-182,274-277` |
| D6 | "swarm router com join commands" | join commands estão em `cluster.ts` (swarm.ts só leitura de nodes/apps/stats) | `routers/cluster.ts:110,143` |
| D7 | "drawer-logs = subscriptions" | é tRPC-over-WebSocket (`applyWSSHandler`) | `wss/drawer-logs.ts:17-21` |
| D8 | "patches sempre no pipeline" | patches só para `sourceType !== "docker"` | `services/application.ts:217-223` |
| D9 | (segurança, novo) | `fullContext` de rollback grava **credenciais de registry em texto plano** no JSONB | `services/rollbacks.ts:74-79` |

### 20.4 Features funcionais do Aether que não estavam no relatório (agora em §13)

`/health`+`/ready`, `/auth/status`, SSO público de login, `/apps/:id/variables/effective`, timeline de deploys, `/apps/states` + SSE, cron jobs globais, registry images (podman), variables audit/export/import, presence, network quality (probe HTTP p50/p95/uptime/HTTP3), runtime metrics, export de compose de app/deploy, mirror run manual, workers start/stop. Infra: auto-provisionamento do ingress Traefik (`api/internal/platform/bootstrap/ingress.go`), master key em `$STATE/keys`, modo prod com gateway nginx, seeds de templates (Affine), `infra/docker-compose.yml` alternativo, `variable_audit` (tabela).

### 20.5 Resumo

- **Claims originais verificados**: 12/12 slices Aether IMPLEMENTED → confirmadas (com os bugs acima); 18/18 claims de PARTIAL/🔴 → confirmadas; 13/13 claims Dokploy → confirmadas (com 8 refinamentos).
- **Nenhuma feature "quebrada" escondida encontrada em runtime** — o padrão CRUD-only se mantém; `main.go` só sobe os 2 workers documentados.
- **Correção mais impactante**: C2 (ACME) — muda o status de domínios/HTTPS de ✅ para 🔴 e adiciona um P1 crítico.
