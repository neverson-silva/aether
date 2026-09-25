#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

ENV_FILE="${AETHER_ENV_FILE:-$PWD/.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  . "$ENV_FILE"
  set +a
fi

# Localiza o Go em locais comuns e expõe o GOBIN (air) no PATH
if ! command -v go >/dev/null 2>&1; then
  for g in "$HOME/.local/go/bin/go" "/usr/local/go/bin/go" "$HOME/go/bin/go"; do
    if [[ -x "$g" ]]; then
      export PATH="$(dirname "$g"):$PATH"
      break
    fi
  done
fi
export PATH="$(go env GOPATH)/bin:$PATH"

# Credenciais do banco (geradas pelo install.sh)
CRED_FILE="${AETHER_STATE:-$HOME/.aether}/.aether-db"
if [[ -f "$CRED_FILE" ]]; then
  # shellcheck disable=SC1090
  . "$CRED_FILE"
fi
export DATABASE_NAME="${DATABASE_NAME:-${DB_NAME:-aether_dev}}"
export DATABASE_USER="${DATABASE_USER:-${DB_USER:-aether}}"
export DATABASE_PASSWORD="${DATABASE_PASSWORD:-${DB_PASSWORD:-}}"
if [[ -z "$DATABASE_PASSWORD" ]]; then
  echo "error: DATABASE_PASSWORD must be set or provided by $CRED_FILE" >&2
  exit 1
fi

export AETHER_STATE="${AETHER_STATE:-$HOME/.aether}"
export DEV_MODE="${DEV_MODE:-true}"
export AETHER_API_ADDR="${AETHER_API_ADDR:-127.0.0.1:8090}"
if [[ "${DEV_MODE}" == "true" || "${DEV_MODE}" == "1" || "${DEV_MODE}" == "TRUE" || "${DEV_MODE}" == "yes" ]]; then
	export AETHER_PUBLIC_URL="${AETHER_PUBLIC_URL:-http://localhost:5173}"
else
  if [[ -z "${AETHER_PUBLIC_URL:-}" ]]; then
    PUBLIC_HOST=""
    for service in https://api.ipify.org https://icanhazip.com https://ifconfig.me/ip; do
      PUBLIC_HOST="$(curl -fsS --max-time 3 "$service" 2>/dev/null | tr -d '[:space:]' || true)"
      [[ -n "$PUBLIC_HOST" ]] && break
    done
    export AETHER_PUBLIC_URL="${PUBLIC_HOST:+http://${PUBLIC_HOST}:5173}"
    export AETHER_PUBLIC_URL="${AETHER_PUBLIC_URL:-http://localhost:5173}"
  fi
fi
export DATABASE_HOST="${DATABASE_HOST:-127.0.0.1}"
export DATABASE_PORT="${DATABASE_PORT:-${PG_PORT:-5432}}"
export DATABASE_SSL_MODE="${DATABASE_SSL_MODE:-disable}"
export DATABASE_MIGRATE_ON_START="${DATABASE_MIGRATE_ON_START:-true}"
export AETHER_MODE="${AETHER_MODE:-dev}"
COOKIE_SECURE_DEFAULT=true
if [[ "$DEV_MODE" == "1" || "$DEV_MODE" == "true" || "$DEV_MODE" == "TRUE" || "$DEV_MODE" == "yes" ]]; then
  COOKIE_SECURE_DEFAULT=false
fi
export AETHER_COOKIE_SECURE="${AETHER_COOKIE_SECURE:-$COOKIE_SECURE_DEFAULT}"

# Credenciais NATS (geradas/persistidas pelo install.sh/install-dev.sh)
export AETHER_NATS_URL="${AETHER_NATS_URL:-nats://127.0.0.1:4222}"
NATS_AUTH_FILE="${AETHER_STATE:-$HOME/.aether}/keys/nats.auth"
if [[ -z "${AETHER_NATS_USER:-}" && -z "${AETHER_NATS_PASSWORD:-}" && -f "$NATS_AUTH_FILE" ]]; then
  AETHER_NATS_USER="$(sed -n '1p' "$NATS_AUTH_FILE")"
  AETHER_NATS_PASSWORD="$(sed -n '2p' "$NATS_AUTH_FILE")"
fi
export AETHER_NATS_USER
export AETHER_NATS_PASSWORD

# Provider de free-domain: nip.io (default) | sslip.io | traefik.me | ngrok
export AETHER_FREE_DOMAIN_PROVIDER="${AETHER_FREE_DOMAIN_PROVIDER:-nip.io}"

# ngrok: expõe API (8090) e frontend (5173) e usa a URL gerada como base
# do free-domain. O ngrok roda em background e a API lê a base via env.
if [[ "${AETHER_FREE_DOMAIN_PROVIDER}" == "ngrok" ]]; then
  if ! command -v ngrok >/dev/null 2>&1; then
    echo "error: AETHER_FREE_DOMAIN_PROVIDER=ngrok requires the ngrok CLI (not found)" >&2
    exit 1
  fi
  echo "Starting ngrok tunnels (api:8090, web:5173)..."
  nohup ngrok http 8090 >"$HOME/.aether/ngrok-api.log" 2>&1 &
  nohup ngrok http 5173 >"$HOME/.aether/ngrok-web.log" 2>&1 &
  for _ in $(seq 1 30); do
    if curl -sf http://127.0.0.1:4040/api/tunnels >"$HOME/.aether/ngrok-tunnels.json" 2>/dev/null; then
      NGROK_HOST="$(grep -oE 'https://[a-z0-9-]+\.(ngrok-free\.app|ngrok\.app|ngrok\.io)' "$HOME/.aether/ngrok-tunnels.json" | head -1 | sed 's#https://##')"
      if [[ -n "$NGROK_HOST" ]]; then
        export AETHER_FREE_DOMAIN_BASE="$NGROK_HOST"
        echo "ngrok free-domain base: $NGROK_HOST"
        break
      fi
    fi
    sleep 1
  done
  if [[ -z "${AETHER_FREE_DOMAIN_BASE:-}" ]]; then
    echo "warning: could not fetch ngrok URL; free-domain base not set (set AETHER_FREE_DOMAIN_BASE manually)" >&2
  fi
fi

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  if [[ -z "${AETHER_BUILD_DOCKER_HOST:-}" ]]; then
    if [[ -S /var/run/docker.sock ]]; then
      export AETHER_BUILD_DOCKER_HOST=unix:///var/run/docker.sock
    elif [[ -S "$HOME/.docker/run/docker.sock" ]]; then
      export AETHER_BUILD_DOCKER_HOST="unix://$HOME/.docker/run/docker.sock"
    fi
  fi
  if [[ -n "${AETHER_BUILD_DOCKER_HOST:-}" ]]; then
    echo "AETHER_BUILD_DOCKER_HOST set for SmartBuild"
  else
    echo "warning: Docker socket path is unavailable; SmartBuild app deploys may fail." >&2
  fi
else
  echo "warning: Docker Engine is unavailable — SmartBuild app deploys will fail." >&2
fi

HOST_WATCHDOG_STATE_DIR="${AETHER_STATE:-$HOME/.aether}"
HOST_WATCHDOG_PID_FILE="$HOST_WATCHDOG_STATE_DIR/host-agent.pid"
HOST_WATCHDOG_OWNED=0
HOST_WATCHDOG_RUNNING=0
if [[ -f "$HOST_WATCHDOG_PID_FILE" ]] && kill -0 "$(cat "$HOST_WATCHDOG_PID_FILE" 2>/dev/null)" 2>/dev/null; then
  HOST_WATCHDOG_RUNNING=1
fi
if [[ "$HOST_WATCHDOG_RUNNING" -eq 1 ]] && [[ -z "$(find "$HOST_WATCHDOG_STATE_DIR/host-stats.json" -mmin -1 -print -quit 2>/dev/null)" ]]; then
  kill "$(cat "$HOST_WATCHDOG_PID_FILE" 2>/dev/null)" >/dev/null 2>&1 || true
  rm -f "$HOST_WATCHDOG_PID_FILE"
  HOST_WATCHDOG_RUNNING=0
fi
if [[ "$HOST_WATCHDOG_RUNNING" -eq 1 ]]; then
  echo "Host metrics watchdog already running."
else
  mkdir -p "$HOST_WATCHDOG_STATE_DIR/logs"
  rm -f "$HOST_WATCHDOG_PID_FILE"
  AETHER_API_URL="http://127.0.0.1:8090/api/v1/ready" AETHER_HOST_AGENT="$PWD/infra/scripts/host-agent.sh" nohup bash "$PWD/infra/scripts/host-watchdog.sh" >> "$HOST_WATCHDOG_STATE_DIR/logs/host-agent.log" 2>&1 &
  echo "$!" > "$HOST_WATCHDOG_PID_FILE"
  HOST_WATCHDOG_OWNED=1
fi

cleanup_host_watchdog() {
  if [[ "$HOST_WATCHDOG_OWNED" -eq 1 ]] && [[ -f "$HOST_WATCHDOG_PID_FILE" ]]; then
    local watchdog_pid
    watchdog_pid="$(cat "$HOST_WATCHDOG_PID_FILE" 2>/dev/null || true)"
    if [[ -n "$watchdog_pid" ]] && kill -0 "$watchdog_pid" 2>/dev/null; then
      kill "$watchdog_pid" >/dev/null 2>&1 || true
    fi
    rm -f "$HOST_WATCHDOG_PID_FILE"
  fi
}

trap cleanup_host_watchdog EXIT INT TERM

if command -v docker >/dev/null 2>&1; then
  LIFECYCLE_IMAGE="docker.io/buildpacksio/lifecycle:${AETHER_LIFECYCLE_VERSION:-0.21.17}"
  if ! docker image inspect "$LIFECYCLE_IMAGE" >/dev/null 2>&1; then
    docker pull "$LIFECYCLE_IMAGE"
  fi
fi

echo "Applying database migrations..."
go run ./api/cmd/api -migrate || echo "warning: migrations failed, the API will retry on start"

exec air
