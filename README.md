# Aether

**A self-hosted PaaS experiment. Nothing more.**

> ⚠️ **READ THIS FIRST (really, please):** This project is **NOT a professional product**.
> It is a **personal hobby project**, built for fun and as a live test of what an AI
> assistant (DeepSeek 4 Flash) can actually build end-to-end. It was written by a single
> human + an LLM over a series of long sessions. There is **no company behind it, no
> support, no security guarantees, no roadmap backed by a team, and no promise that
> anything will keep working after the next commit**.
>
> Please do **not** run this on anything you care about, do **not** put real
> production workloads on it, do **not** trust it with secrets, and do **not** use it as
> a base for your company's infrastructure. Treat it as a learning artifact — a big,
> slightly chaotic experiment in "how far can a hobby-grade AI-driven codebase go?".
>
> If you still want to look around: welcome, the code is real, the tests run, and the
> architecture is genuinely interesting — but you've been warned.

---

## What is this?

Aether is a self-hosted PaaS (Platform as a Service) that lets you run applications,
databases, cron jobs, workers, snapshots and backups on your own machine(s) with a web
console. It started as an attempt to answer "how far can one person (plus an AI copilot)
get in building something that resembles Coolify/Dokploy/Railway?" — and it got very far.

## Stack

| Layer | Technology |
|-------|------------|
| **API / backend** | Go 1.26, Gin (HTTP), `log/slog` |
| **Database** | PostgreSQL 16 (source of truth), `pgx` + `sqlc` generated queries, versioned SQL migrations |
| **Messaging / events** | NATS 2.14+ — **Core NATS** (ephemeral realtime, live signals) + **JetStream** (durable jobs, event log, replay, scheduler, KV locks/state, DLQ) via `github.com/nats-io/nats.go` |
| **Queue / worker model** | JetStream durable consumers, pull-based, ACK/NAK with backoff, dead-letter queue, transactional outbox in Postgres |
| **Container runtime** | Podman (rootless) with crun, Buildah, Skopeo, Quadlet, conmon; CNB buildpacks (100% own buildpacks, no Paketo) |
| **Builds** | `pack` (Cloud Native Buildpacks) with a custom local builder (`ubuntu:24.04` base, `aether/*` buildpacks) + Dockerfile builds + custom command builds |
| **Ingress / TLS** | Traefik (dynamic config, HTTP/3, ACME/Let's Encrypt, middlewares) |
| **Frontend** | React 18 + TypeScript, Vite 5, TanStack Router, TanStack Query, Zustand, Tailwind CSS v4 + custom design system (`@aether/design-system`), Monaco editor, Phosphor icons |
| **Realtime UI** | Single WebSocket (`/api/v1/ws/realtime`) with seq/replay, heartbeat, ephemeral events; Zustand stores for global state |
| **Infra scripts** | `install.sh` (Linux production-ish), `install-dev.sh` (macOS podman-machine), Podman `docker-compose` as an alternative, local image registry |
| **Observability** | Per-process health servers (`/health`, `/ready`, `/metrics`), job/queue/collection metrics |
| **Studio** | SQL engine for managed databases (introspection, query editor, object explorer, SQL autocomplete with an "intelligence" engine on IndexedDB) |

## Architecture in one diagram

```text
                    ┌──────────────────┐
                    │    PostgreSQL    │
                    │  source of truth │
                    └────────┬─────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
            ▼                ▼                ▼
       aether-api       aether-worker   aether-monitoring
            │                │                │
            └────────────────┼────────────────┘
                             │
                             ▼
                    ┌──────────────────┐
                    │       NATS       │
                    │  Core + JS       │
                    └────────┬─────────┘
                             │
           ┌─────────────────┼─────────────────┐
           │                 │                 │
           ▼                 ▼                 ▼
       Core NATS         JetStream KV      JetStream Streams
           │                 │                 │
       realtime            locks             jobs
       events              presence          retry
       signals             ephemeral         schedules
                                             replay
                                             DLQ
```

Three binaries, clean responsibilities:

- `aether-api` — HTTP/WebSocket request/response only. Never executes long work.
- `aether-worker` — consumes JetStream jobs: deploys, backups, restores, snapshots,
  cron, scheduler, outbox dispatcher, recovery, watchers.
- `aether-monitoring` — isolated daemon collecting host + container metrics every ~2s.

## Quick start

```bash
# Linux server (or macOS with podman machine)
./install.sh start

# macOS dev loop (API on :8090 with hot reload)
./dev.sh          # terminal 1
./dev-web.sh      # terminal 2 (Vite on :5173)

# Configuration lives in ~/.aether + .env
```

## Why it's hobby-grade (the honest list)

- Written by **one human + one LLM session history** — code style is a mix of "heavily
  iterated" and "quickly written to pass the test".
- Architecture and code change **frequently and without notice**. Branches rebase for fun.
- Not audited by anyone. No security review. Secrets exist; please don't give it real ones.
- No CI/CD on anything real. No releases. No versioned artifacts.
- Lots of work happens at 2am in a terminal. Expect ✨ surprises ✨.
- **The entire point of this repository is to answer this question:** *"What can
  DeepSeek 4 Flash build when pointed at a big ambitious problem and given lots of turns?"*
  Everything else is a side effect.

## Testing

...actually, there ARE tests, and they run. But use them as evidence of the journey,
not as a guarantee:

```bash
AETHER_TEST_DATABASE_PORT=5433 AETHER_API_TEST_DATABASE_PORT=5433 \
go test ./api/internal/... -count=1 -p 1 -timeout 25m
```

## License

Probably nothing, honestly. If you want to use any of this, contact the author
and negotiate. Or fork it and do whatever — it's a hobby, be a bit nice.

---

**Have fun. Don't deploy to production. And never judge a codebase by the stories its
author tells about it.**
