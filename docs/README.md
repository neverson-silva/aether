# Aether Platform Operations

Aether is a self-hosted PaaS for applications, databases, Compose stacks, workers, jobs,
backups, snapshots, domains, and realtime deployment operations.

## Runtime boundary

Docker Engine is the only container runtime. The API, worker, monitoring process, and
Compose execution adapter use the injected Docker Engine contract. Docker CLI usage is
restricted to installer and Compose boundary code; feature packages do not shell out to a
container runtime.

PostgreSQL is the functional source of truth. NATS Core carries ephemeral realtime signals,
while NATS JetStream carries durable deployments, retries, cancellation, event replay, and
dead-letter handling.

## Components

- `aether-api`: HTTP and WebSocket control plane.
- `aether-worker`: queued deployments, backups, restores, snapshots, cron, pipelines, and recovery.
- `aether-monitoring`: host and Docker container metrics.
- PostgreSQL, NATS JetStream, Docker Compose, Traefik, and the web gateway.

## Installation

The migration is destructive and intended for a non-production environment. Existing runtime
containers, networks, volumes, images, builder caches, and checkout state may be removed during
the cutover. Rollback requires an external repository and host snapshot.

```bash
./install.sh install
./install.sh update
./install.sh start
./install.sh stop
./install.sh status
```

Development uses the same Docker Engine target:

```bash
./install-dev.sh start
./dev.sh
./dev-web.sh
```

The installer validates Docker CLI, Docker Compose, daemon availability, architecture, socket
access, ports, storage, and required directories before starting the platform. The Docker socket
is a privileged local control boundary and must never be exposed publicly.

## Deployment model

Every service kind uses one durable deployment envelope:

`persist → enqueue → FIFO worker claim → build or materialize → Docker operation → observe → persist → realtime event`

Applications use Docker builds or CNB. Databases use the managed engine adapters. Compose stacks
clone the selected Git branch when requested and execute the selected Compose file with its own
directory as the project directory. Relative build contexts, Dockerfiles, volumes, and `.env`
references therefore retain Compose semantics, including nested files.

Creation never deploys implicitly. Deployment state describes the deployment job; service state
describes the Docker container or Compose project. Runtime events reconcile start, stop, restart,
exit, and health transitions without browser polling.

## Security and data handling

- Secrets are encrypted at rest and never written to deployment logs.
- Generated environment files are private and are kept outside Git checkouts.
- Container and terminal operations validate organization and service ownership.
- Snapshot and backup paths are validated before filesystem operations.
- Destructive cleanup operates only on explicit Aether resource names.

## Verification

```bash
cd api
AETHER_TEST_DATABASE_PORT=5433 AETHER_API_TEST_DATABASE_PORT=5433 \
AETHER_TEST_DATABASE_USER=postgres AETHER_TEST_DATABASE_PASSWORD=postgres \
AETHER_NATS_TEST_URL=nats://127.0.0.1:4228 \
go test ./internal/... -count=1 -p 1 -timeout 25m
go build -o /tmp/aether-api ./cmd/api
go vet ./internal/...
```

The final migration gate also verifies Docker lifecycle and Compose integration, realtime
events, FIFO and cancellation behavior, logs, metrics, terminal sessions, backups, snapshots,
pipelines, cron, domains, and the absence of legacy runtime references.
