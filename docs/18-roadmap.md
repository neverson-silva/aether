# 18 — Roadmap Técnico

> **Status:** Plano de execução em fases.
> **Objetivo:** Ordenar o desenvolvimento da arquitetura até paridade com Coolify/Dokploy,
> funcionalidades exclusivas e camada Enterprise.

---

## 1. Visão geral das fases

| Fase | Nome | Objetivo | Critério de saída |
|------|------|----------|-------------------|
| F0 | Fundação | Spikes técnicos e provas de conceito | Decisões ADR validadas em benchmark |
| F1 | MVP | Núcleo funcional end-to-end | Deploy real de app + UI + API + CLI |
| F2 | Paridade Coolify | Cobertura de features do Coolify | Checklist funcional Coolify |
| F3 | Paridade Dokploy | Cobertura de features do Dokploy | ✅ implementada (nixpacks/buildpacks, registry interno, webhooks de saída, /metrics) |
| F4 | Exclusivas | Feature set próprio | ✅ implementada (costs, autopilot, gitops, mirrors, k8s driver, migrate, netq, snapshots dedup) — AI pendente |
| UI zero-mock | Auditoria 22 telas | Nenhum botão/tela mockado | ✅ implementada (migração 7: branding persistente + aplicado no shell, pipelines CI/CD reais com stages/containers, clusters com afinidade no scheduler, SSO OIDC completo discovery→auth→token→userinfo→provisionamento) + testes |
| F4 | Exclusivas | Diferenciais competitivos | GTM/validação de mercado |
| F5 | Enterprise | HA, SSO, compliance | Vendas enterprise |

## 2. Fase 0 — Fundação (spikes)

**Objetivo:** validar as hipóteses técnicas mais arriscadas antes de escrever produto.

| Spike | Pergunta | Saída |
|-------|----------|-------|
| Podman rootless + Quadlet | performance, rede rootless, limites | doc de decisão |
| Execution Engine (porta + driver) | latência de operações, mTLS | protótipo de driver |
| SQLite WAL sob carga do core | contenção de escrita | benchmark |
| Outbox + event sourcing em SQLite | throughput de eventos | benchmark |
| Buildah rootless + cache | custo de build, GC | benchmark |
| Traefik via API dinâmica | configuração em memória | protótipo |
| Comparativo Coolify/Dokploy/Aether | números de referência | relatório (ver [02] §8) |

**Decisões F0:** linguagem final — **GO** (decidido em F0 via spike
[`spikes/lang-go-rust`](../spikes/lang-go-rust/); critérios: ecossistema OCI (podman/buildah/
skopeo/containers são Go), RAM idle dentro do orçamento (5 MB vs meta 120 MB), velocidade de
entrega). Spikes concluídos: SQLite WAL (contenção zero), outbox/event sourcing (~80k ev/s),
linguagem (Go).

## 3. Fase 1 — MVP

**Escopo mínimo para "deploy real de app" no mesmo servidor:**

- Instalador (1 servidor), binário único.
- Core: API REST, RBAC mínimo (owner/admin/developer/viewer), UI básica.
- Agent local (embedded): Execution Engine (podman driver) + Quadlet + build (Dockerfile).
- Deploy de aplicações por imagem e por Git (GitHub) com webhook.
- Domains + HTTPS (Let's Encrypt HTTP-01) via Networking/Cert Engine.
- Logs e métricas básicas; timeline.
- Env vars/secrets (cifrados), volumes, health check, rollback.
- Backups de estado (SQLite) e restore.
- CLI: install/update/login/apps/logs/rollback.

**Features adiadas de v1:** multi-server, preview deployments, databases gerenciadas,
marketplace completo, OIDC/MFA, workers/cron avançados.

**Aceite:** todos os cenários de [`03`](03-metas-engenharia.md) §3 passando em CI + benchmark
idle dentro das metas.

### Status da Fase 1 (implementação)

> **Em andamento — núcleo vertical completo e validado por teste E2E real (Docker).**

| Item | Status |
|------|--------|
| Binário único (`aether`), CLI (`install/serve/login/apps/deploy/logs/rollback/backups/...`) | ✅ |
| Core: API REST + SSE + RBAC (owner/admin/developer/viewer) + API keys | ✅ |
| Execution Engine: porta `RuntimeDriver` + PodmanDriver + DockerDriver | ✅ |
| Deploy por imagem E por Git (clone + build Dockerfile) com health check | ✅ |
| Rollback (restaura imagem/commit do deployment anterior) | ✅ |
| Env vars/secrets cifrados (AES-256-GCM, KEK/DEK) injetados no container | ✅ |
| Logs em tempo real (LiveLog + SSE), stats sob demanda, timeline de eventos | ✅ |
| Networking Engine: config dinâmica em memória + reconcile no boot | ✅ |
| Certificate Engine: ACME HTTP-01 completo (validado contra CA fake em teste) | ✅ |
| Webhook GitHub (HMAC-256) com política de branch | ✅ |
| Backups de estado (VACUUM INTO) + listagem | ✅ |
| UI estática básica (login, apps, deploy, logs, env, domínios, backups) | ✅ |
| Instalador (`install/uninstall/update/rollback` + unit systemd) | ✅ |
| Testes: unitários + E2E completo com Docker (install→deploy→rollback→cleanup) | ✅ |
| UI React+TS (TanStack Query/Router, RHF+zod, Tailwind v4, design system próprio) | ✅ 20 telas adaptadas das 26 telas de referência |
| **Fase 0 fechada** | H6 validado em Linux real (VM): Quadlet/rootless/Buildah ✓ · benchmark idle (14 MB RAM, 284 KB SSD, 0.1% CPU) ✓ · SQLite em ext4 (2× macOS) ✓ · ACME real contra Pebble (cert issued) ✓ |
| **Pendente** | multi-server, preview deployments, databases, marketplace, LE staging/real no VPS do usuário (domínio confirmado), telemetria idle em VPS físico |

## 4. Fase 2 — Paridade com Coolify

Adicionar sobre o MVP:

- Marketplace + One-Click Apps + Templates.
- Docker Compose (mapeado para unidades OCI).
- GitLab e Bitbucket.
- Preview Deployments (por PR).
- Cron Jobs e Workers.
- Health checks configuráveis + alertas.
- Multi-server (1 core + N agents; mTLS; gRPC).
- Databases gerenciadas (PostgreSQL, MySQL/MariaDB, Redis, MongoDB) via runtime OCI.
- Backup de volumes de app + alvos S3.
- Notificações (e-mail, Slack, Discord, Telegram).
- OIDC/SSO e MFA.
- Terminal via WS.
- Export/import `aether.yml`.

**Checklist de paridade Coolify:** compare feature-a-feature com o catálogo público do Coolify;
cada item tem teste E2E.

### Status da Fase 2 (implementação)

> **Em andamento — núcleo F2 validado com Docker real (provision, preview, compose, backup).**

| Item | Status |
|------|--------|
| Databases gerenciadas (PostgreSQL/MySQL/MariaDB/Redis/MongoDB) — provision, creds cifradas, DSN interno, backup dump+gzip, restore | ✅ + E2E Docker (provision→ready→backup) |
| Cron Jobs (schedule 5 campos, scheduler persistido 15s) | ✅ |
| Workers (long-running, start/stop, replicas) | ✅ + smoke (running) |
| Preview Deployments (branch→deploy→domínio temporário→teardown) | ✅ + smoke (active/teardown) |
| GitLab + Bitbucket webhooks (push + PR events, auto-preview por branch) | ✅ + smoke (preview via webhook) |
| Docker Compose (parser YAML → units OCI, up/down) | ✅ + smoke (container up/down) |
| Marketplace + Templates (catálogo 8 apps, install → stack compose) | ✅ + smoke |
| Volume backups + S3 destinations (SigV4 assinado) | ✅ |
| Notificações (slack/discord/telegram/email/webhook) | ✅ |
| OIDC/SSO + MFA (TOTP RFC 6238, enroll/verify/disable, login com código) | ✅ + unit tests |
| Terminal via WebSocket (`docker exec -it`, bidirecional) | ✅ + smoke (handshake 101) |
| Export/import `aether.yml` (org → yaml → org nova) | ✅ + unit tests |
| CLI F2 (`databases/cron/workers/compose/previews/templates/export/import`) | ✅ |
| UI F2 (Databases real, AppDetail cron/workers/previews/terminal, Marketplace install, Notifications canais, Storage S3, Export/Import) | ✅ |
| **F3** | Nixpacks/Buildpacks (`build_type`), preview_domain, registry interno (Skopeo+registry:2, push pós-deploy, catalog UI), webhooks de saída (HMAC, 6 eventos), `/metrics` Prometheus | ✅ + unit tests (HMAC, filtro eventos, registry roundtrip, build errors) + smoke Docker real |
| **Multi-server (RFC-0015)** | 1 core + N agents mTLS (CA interna, token single-use 24h), heartbeat 5s + watchdog 30s, scheduler least-loaded, deploy remoto (imagem), failover → local, `server` CLI, UI Servers real | ✅ + unit tests (token, scheduler, dequeue) + smoke 2 nós (deploy remoto ready, unhealthy 28s, failover) |
| **Pendente** | deploy remoto de git (agente faz clone/build), multi-server de composições/bancos; LE staging/real no VPS |
| Environments (RFC-0020) | Projeto → Environment (1:N) → Services; production auto-criado default em transação; CRUD (rename/desc/color), default transacional (1 por projeto), regras (não deletar último, não deletar com serviços), migração idempotente com backfill, página do projeto com selector + summaries; apps herdam env default | ✅ + 4 testes (production auto, CRUD+default tx, bloqueios, app→default) + smoke API completo |
| Env Switcher + Env Vars (RFC-0022) | Switcher único (dropdown escalável, menu contextual rename/default/delete, sem pills horizontais), URL `?environment=<slug>`, persistência localStorage, default automático; variáveis por Environment (Projeto+Ambiente) com editor KEY=value (duplicatas, máscara de secrets, reveal, export/import), herança automática em todos os services, precedência Service > Environment > System, interpolação `${{environment.KEY}}` centralizada, secrets criptografados, cache em memória com invalidação, auditoria | ✅ 3 testes (precedência+interpolação, máscara+parser, audit+cache) + smoke container real (herança, secret descriptografado, override) |
| PostgreSQL (RFC-0021) | SQLite removido 100%; PostgreSQL 15+ único banco; bootstrap automático (cria banco, valida versão, schema, migrations + advisory lock, seeds); retry backoff; healthz com DB/pool/latência; DATABASE_* vars; backup/restore via pg_dump/pg_restore; Dockerfile + docker-compose (postgres + healthcheck); testes com schema por teste | ✅ 6/6 pacotes verdes + TestConcurrentMigrateAdvisoryLock + smoke (bootstrap, 2ª inicialização sem re-migrar, retry com PG caído) |

## 5. Fase 3 — Paridade com Dokploy

Status: **implementada e validada (smoke Docker real)** — migração 4 (build_type, preview_domain, out_webhooks, registry_settings).

- ✅ Builds avançados: Nixpacks (`nixpacks plan` -> Dockerfile -> driver) e Buildpacks (`pack build`), selecionáveis por app via `build_type` (dockerfile|nixpacks|buildpacks).
- ✅ Preview/PR environments robustos: `preview_domain` por app (subdomínio `app-branch.<domínio>`); sem wildcard mantém `*.preview.aether.local`.
- ✅ Workers de background geridos (F2).
- ✅ Backup/restore com dump específico (pg_dump, mysqldump, mongodump, redis save) (F2).
- ✅ Terminal completo + execução em containers (F2).
- ✅ Registry privado interno: container `registry:2` + push automático pós-deploy via Skopeo (`--override-os linux`), catalog API com tags+size (OCI manifests), deletar imagem; UI Registry real.
- ✅ Webhooks de saída: eventos `deployment.started/ready/failed`, `backup.started/finished/failed`, HMAC-SHA256 (`X-Aether-Signature`), assinatura opcional por secret; UI em Notificações.
- ✅ Observabilidade: `/metrics` e `/api/v1/metrics` em formato Prometheus (plataforma + apps: cpu, mem, net, io), endpoint scrape-ready; OTLP fica como plugin (F4).
- ⏳ Plugins de infra (Hetzner/AWS) — plugin work, adiado.

**Checklist de paridade Dokploy:** idem, comparando com o escopo público do Dokploy.

## 6. Fase 4 — Funcionalidades exclusivas

Status: **implementada** (migração 6) — exceto AI (adiado, próximo ciclo).

- ❌ ~~Cost-Aware Scheduling~~ — **removido**: Aether é self-hosted PaaS, não cloud — nenhum cálculo de custo na UI nem no backend.
- ✅ **Resource Autopilot**: policy por app (cpu/mem min-max, thresholds up/down %, cooldown), loop 60s → `docker/podman update`; eventos persistidos; UI em App Detail (config + ações recentes).
- ✅ **Declarative GitOps**: configs watch repo a cada 60s (aether.yml), drift +/−~ por sync, org alvo dedicada (gitops-<name>), reconcile idempotente (create/update por nome), apply manual ou auto; UI página GitOps.
- ✅ **Multi-registry mirror**: replicação Skopeo source→dest com filtro de tags, on-demand; UI seção Mirrors na página Registry.
- ✅ **Driver Kubernetes**: implementação REST pura (Deployment + Service, scale/rollout/patch, exec/logs limitados), ativado via `AETHER_RUNTIME=k8s` + AETHER_K8S_API/TOKEN/CACERT/NAMESPACE; testado contra mock API (payloads Deployment/Service/resources). Cluster real: pendente validação E2E.
- ⏳ **AI assistente (MCP)**: adiado (próximo ciclo).
- ✅ **Zero-copy migration**: `aether migrate --platform coolify|dokploy --dir DIR [--apply]` — detecta docker-compose + .env, marca secrets, importa como compose apps com env; testado com fixtures.
- ✅ **Network quality dashboard**: probe HEAD a cada 30s por app (latência, uptime, HTTP/3 via header Alt-Svc); p50/p95; UI página Networking.
- ✅ **Snapshot de volumes com dedup**: tar via busybox helper → chunks 1MiB → zstd (klauspost) → content-addressed (sha256) com dedup; restore via exec tar no volume; UI seção em Storage; validado com dedup real (167B salvos em snapshots idênticos) e restore completo.

### RFC-0030/0031 — Service Management (implementado em ciclos recentes)

- ✅ **Notificações real-time (RFC-0031)**: migration 11, NotificationEngine + SSEHub, eventos deploy completos (queued/building/starting/healthcheck/ready/failed/rolled_back com triggered_by), backup.* e server.*; `GET /api/v1/events/stream` (token→Bearer→cookie), Bell + badge + dropdown, Toast com níveis, reconexão SSE com backoff 1s→30s + fallback polling; E2E duas sessões + offline (PASS).
- ✅ **Marketplace (RFC-0030 §7/8)**: migration 12, catálogo ~80 templates em 25 categorias (databases, cms, monitoring, ai, devtools, security, media, automation, networking, email, productivity, wiki, homelab, messaging, erp, web), featured/verified/installs, trending, busca + filtros, template detail com readme, instalação real (gera compose stack com volumes/env/secrets aleatórios).
- ✅ **Launcher (Cmd+K) (§3)**: modos `>` (comandos) / `@` (projetos) / `#` (templates), recents + favorites (localStorage), seções, create service direto do launcher, navegação por teclado.
- ✅ **Framework Detection (§4)**: shallow clone + heurísticas (Next/Nuxt/SvelteKit/Vite/Express/Nest/Remix/Astro/Go/Rust/Python/FastAPI/Django/Rails/Elixir/Java/Deno/Bun/Dockerfile), endpoint `POST /api/v1/detect` + `POST /api/v1/apps/{id}/detect`, botão no wizard com card de resultado (framework/build/start/port); testado (9 casos).
- ✅ **Deploy Comparison (§10)**: migration 13, env_snapshot por deployment (capturado no Deploy), `GET /apps/{id}/deployments/compare` com diff real de image/commit/env (added/removed/changed); UI com seleção de 2 deploys + modal de comparação.
- ✅ **Failure Analysis (§10.4)**: sugestões heurísticas (OOM, port conflict, image unavailable, registry auth, Dockerfile missing, healthcheck timeout, DNS, build errors) com severidade no histórico de deploys.
- ✅ **Logs Engine (§11)**: ANSI strip, detecção de JSON com pretty print, filtro regex/literal, filtro por nível (error/warn/info/debug), json-only, follow/pause, export de logs, `GET /apps/{id}/logs/history` para paginação.
- ✅ **Compose Wizard (§6)**: `POST /api/v1/compose/validate` (parser YAML real: services/volumes/networks/erros/warnings/dependências/portas), editor com linha numerada + highlight, painel de validação ao vivo + grafo de serviços, aba "Docker Compose" no Create Service.
- ✅ **Database Wizard (§5)**: seleção de engine categorizada (Relational/NoSQL/Cache) no modal de criação.
- ✅ **Alert Rules Engine (§16)**: migration 14, regras por métrica (cpu/memory/memory_pct) com threshold/severity/target app, loop 30s avaliando containers reais, eventos persistidos + notificação real-time (`alert.warning/critical`), resolve de eventos, painel na página Monitoring; testado (CRUD + avaliação).
- ✅ **Rate Limiting (§18)**: middleware token bucket por IP (120 req/min, Retry-After, 429) em todo /api/ exceto login.
- ✅ **Design System (§17)**: Skeleton + SkeletonList, animações fade-in/modal-pop/shimmer no CSS, modal com animação de entrada.
- ✅ **Overview tab (§9.3)**: health status, memory, domínios com link direto no card Service Details.

### Auditoria §19 (checklist RFC-0030) — segunda rodada

Após auditoria linha a linha do checklist §19 (base + expansão), foram fechados nesta rodada:

- ✅ **100 templates / 30 categorias** (era 79): +21 na onda 2 (pgadmin, phpmyadmin, adminer, dokuwiki, synapse, element, tailscale, cloudflared, postfix, dovecot, grasshopper, firefly, calcom, linkwarden, stirling-pdf, trilium, webtop, opencodex, grafana-oncall, bazarr, navidrome, forgejo, librechat, supabase).
- ✅ **Editor's Choice**: migration 15 (`editors_choice`), 6 curados (grafana, prometheus, minio, n8n, wordpress, ollama), endpoint `?editors_choice=true`, upsert no seed (corrige flags em bancos existentes), cleanup de templates órfãos no seed (removeu 4 duplicados do catálogo antigo no banco local).
- ✅ **Progressive disclosure** no Create Service (seção Advanced colapsável).
- ✅ **Skeleton loading** na lista de services (SkeletonList).
- ✅ **Auto-suggest de imagens populares** (8 imagens no campo Image).
- ✅ **Template source type** no modal (grid 18 trending preenche o form).
- ✅ **Database Wizard grid**: engines por categoria (Relational/Document/Cache) com seleção visual.
- ✅ **Readme renderizado** com mini-markdown (código/bold/headers/listas/quote) no template detail.
- ✅ **Danger Zone**: delete de app exige digitar o nome exato (requireType).
- ✅ **Variable editor auto-save** (debounce 2s) + duplicate detection + secrets reveal.
- ✅ **A11y**: role=dialog/aria-modal no Modal e Launcher, aria-live no launcher, prefers-reduced-motion global.
- ✅ **Snapshot scheduling** (Fase 6): migration 16 (`snapshot_schedules`), mini-cron (5 campos + @daily/@weekly/@hourly, horizonte 60d), scheduler 30s, retenção (remove além de N), notificação `backup.finished`, UI na página Storage; testes: `TestNextCron` (5 casos) + `TestSnapshotScheduleCRUD`.
- ✅ **Rate limiting §18.9**: headers `X-RateLimit-Limit/Remaining/Reset` + 429 + Retry-After.
- ✅ **Deploy com tag custom**: `ImageOverride` agora flui para o deployment (resolveImage honra override para apps de imagem).

### Checklist §19 — status final

- Fase 1 (Fundação): **8/8** ✅
- Fase 2 (Marketplace): **4/4** ✅
- Fase 3 (UX Premium): 4/5 ✅ (~1: keyboard nav Tab audit)
- Fase 4 (Wizards): 2/3 ✅ (~1: Application Wizard sem OAuth GitHub/GitLab)
- Fase 5 (Detalhe Premium): **8/8** ✅
- Fase 6 (Enterprise): 4/8 ✅ (faltam: autoscaling horizontal, backup S3 automático, metrics storage/retention, vault integration)
- Expansão: F1 7/10, F2 4/5 (community submission), F3 2/6 (tokens aplicados ok; motion ok; popover ok; skeleton ok parcial — listas de projetos/timeline faltam shimmer), F4 2/6, F5 8/10, F6 3/8

**Pendências assumidas para ciclos futuros** (fora do escopo desta rodada): AI assistente (MCP), plugins de infra (Hetzner/AWS), driver K8s em cluster real, GitHub/GitLab OAuth no wizard, autoscaling horizontal, backup S3 automático, storage de métricas com retenção, integração Vault, Monaco editor, virtual scroll 100k+ linhas.

### Higiene de imagens e delete completo de apps

- ✅ **Delete de app agora limpa tudo** (era: só container + banco): volumes nomeados do app (`VolumeRemove`), imagens próprias de git apps no registry interno + local (`docker/podman rmi`). Imagens públicas/base (nginx, postgres...) **nunca** são tocadas — são compartilhadas.
- ✅ **Política de retenção de imagens**: migration 17 (`apps.image_retention`), global via `AETHER_IMAGE_RETENTION` (default 5, 0 = desliga), per-app via PATCH `/apps/{id}` + campo na Settings UI. Job a cada 30min: mantém as N tags mais recentes (`aether.local/<name>:<número-deploy>`) + a do container ativo, deleta o resto (registry + local); notificação `system.images_cleanup`. Teste com registry fake (6 tags → retenção 2 → deleta 4, mantém 5-6) PASS.
- ✅ Smoke real: PATCH retention → detail reflete → delete limpa.

### Reestruturação do fluxo de criação (RFC-0030 §2)

- ✅ **Create Service Launcher** (substitui o modal genérico): command-palette com pesquisa + seções (Applications com 10 presets de framework, Databases com 7 engines, Compose, Marketplace, Import). Só escolhe — nunca abre formulário genérico.
- ✅ **Application Wizard** dedicado: 4 steps (Repository → Runtime → Resources → Variables) com presets de framework, memory presets (256MB→8GB/Unlimited), CPU presets (0.25→8) + custom em millicores (500m).
- ✅ **Database Wizard** dedicado: 5 steps (Engine → Version → Storage → Resources → Credentials) com grid por categoria.
- ✅ **Compose Wizard** dedicado: Monaco-style editor + validação ao vivo + grafo (sem formulário).
- ✅ **Marketplace**: tabs (All/Trending/Latest/Favorites), favoritos ⭐ (localStorage), cards com installs + license + verified, trending strip.
- ✅ **Advanced fora do modal**: resources com presets (CPU/RAM) direto na Settings do serviço; criação fica mínima (configurar depois, estilo Railway/Render).
- ✅ **Bug rate limiter**: 120 req/min matava o app real (polling stats/SSE) — subiu para 600/min com exclusão de streams (events/stats/logs).

### Overlay Manager + transições (fluxo de criação corrigido)

- ✅ **OverlayManager centralizado** (`OverlayManager.tsx` + provider no main.tsx): um único overlay ativo por vez, scroll lock no body, Escape global (fecha só o overlay ativo), foco restaurado ao fechar, `useOverlayGate` (mount/fade-out 180ms/onClosed).
- ✅ **Create Service Launcher nunca coexiste com wizards**: escolher opção → palette fecha com fade-out (180ms) → desmonta → somente então o wizard abre. Zero sobreposição de overlays (validado por polling de 50ms em browser real).
- ✅ **CommandPalette (Cmd+K)**: Create Service agora fecha a palette antes de montar o launcher (antes: o launcher nem abria — bug do early-return).
- ✅ **Escape contextual**: na palette fecha só a palette; no wizard fecha só o wizard.
- ✅ **Foco**: entrada no campo de busca/primeiro campo do wizard; restaurado ao fechar.
- ✅ **3 bugs reais corrigidos no caminho**: (1) useEffect do palette resetava `serviceCreate` e matava o launcher; (2) ComposeWizard quebrava com `warnings: null` do backend (agora arrays não-nil no `ValidateComposeYAML` + guards `?? []` no front); (3) rate limiter 429 matando o app (600/min + exclusão de streams).

### Testes de UI (browser real, Chrome headless via playwright-core)

1. New Service → launcher ✓
2. Launcher → Next.js: sobreposição **zero em qualquer momento**; wizard abre após fade-out ✓
3. Escape → fecha só o wizard ✓
4. Cmd+K → Create Service → launcher (palette desmontada) ✓
5. Cmd+K → launcher → Docker Compose → wizard ✓
6. Cmd+K → launcher → PostgreSQL → Database wizard ✓
7. Launcher → Browse Marketplace → navega (launcher desmontado) ✓
8. Scroll lock ativo com overlay; liberado ao fechar ✓

## 7. Fase 5 — Enterprise


- **PostgreSQL HA** (Patroni) + agentes em cluster.
- **SAML** e IdP enterprise; SCIM (provisionamento).
- **HSM/TPM** para KEK; criptografia de volumes.
- **Compliance**: SOC2 (documentação), audit export, data residency.
- **Suporte** a SLA, releases LTS, canais privados.
- **Kubernetes driver** (rodar workloads em cluster K8s existente) como oferta.
- **Multi-tenant SaaS-ready** (isolamento por org).

## 8. Critérios de corte de fase (gates)

Toda fase exige:
1. Benchmark de recursos dentro das metas ([`03`](03-metas-engenharia.md)).
2. Testes E2E da checklist da fase.
3. Migração backward-compat (banco + CLI) verificada.
4. Documentação atualizada (RFCs correspondentes).

## 9. Orçamento de risco (o que pode atrasar)

| Risco | Mitigação |
|-------|-----------|
| Podman rootless networking | spikes em F0; fallback rede host/socket |
| Event sourcing complexidade | começar com outbox simples; evoluir |
| Builds em servidores fracos | build externo opcional desde v1 |
| Multi-server (fase 2) | adiável: MVP já é utilizável single-node |
| Paridade ambiciosa demais | priorizar checklist por peso de uso |

## 10. Referências

- Metas: [`03-metas-engenharia.md`](03-metas-engenharia.md).
- Concorrentes: [`02-analise-concorrentes.md`](02-analise-concorrentes.md).
- RFCs correspondentes a cada fase.

## RFC-0032 — Enterprise Organizations & Multi-Tenancy ✅
- Migração 21: orgs+slug/soft-delete, users.global_role, projects+slug, project_assignments, audit_logs.
- RBAC: owner/admin/member/viewer + Global Admin; membership validada por request (X-Aether-Org).
- Project assignments: Member/Viewer enxergam apenas projetos atribuídos (backend + UI).
- Org APIs CRUD + members + invite + assign + audit; /me multi-org.
- Frontend: OrgProvider + OrgSwitcher no shell, troca de org sem reload, /organizations/new e /organizations/$id.
- Testes: TestMultiTenancyProjectScoping. Suite 7/7.
