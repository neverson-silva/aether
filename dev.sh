#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

# Localiza o Go em locais comuns e expõe o GOBIN (air) no PATH
if ! command -v go >/dev/null 2>&1; then
  for g in "$HOME/.local/go/bin/go" "/usr/local/go/bin/go" "$HOME/go/bin/go"; do
    if [[ -x "$g" ]]; then
      export PATH="$(dirname "$g"):$PATH"
      break
    fi
  done
fi
export PATH="$PATH:$(go env GOPATH)/bin"

# Credenciais do banco (geradas pelo install.sh)
CRED_FILE="$HOME/.aether/.aether-db"
if [[ -f "$CRED_FILE" ]]; then
  # shellcheck disable=SC1090
  . "$CRED_FILE"
fi
export DATABASE_NAME="${DATABASE_NAME:-${DB_NAME:-aether_dev}}"
export DATABASE_USER="${DATABASE_USER:-${DB_USER:-aether}}"
export DATABASE_PASSWORD="${DATABASE_PASSWORD:-${DB_PASSWORD:-aether_dev_pass}}"

export AETHER_STATE="$HOME/.aether"
export AETHER_API_ADDR="127.0.0.1:8090"
export DATABASE_HOST="127.0.0.1"
export DATABASE_PORT="${PG_PORT:-15432}"
export DATABASE_SSL_MODE="disable"
export DATABASE_MIGRATE_ON_START="true"
export AETHER_REDIS_ADDR="127.0.0.1:16379"
export AETHER_MODE="dev"
export AETHER_COOKIE_SECURE="false"

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

# Podman socket — pack/buildpacks (SmartBuild) precisa de um daemon docker-compatível
# (DOCKER_HOST), senão o pack cai no /var/run/docker.sock e o build falha.
PODMAN_SOCK=""
for s in "/run/podman/podman.sock" "${XDG_RUNTIME_DIR:-}/podman/podman.sock" "/run/user/$(id -u)/podman/podman.sock"; do
  if [[ -S "$s" ]]; then
    PODMAN_SOCK="$s"
    break
  fi
done
if [[ -n "$PODMAN_SOCK" ]]; then
  export DOCKER_HOST="unix://$PODMAN_SOCK"
  export CONTAINER_HOST="unix://$PODMAN_SOCK"
  echo "podman socket: $PODMAN_SOCK (DOCKER_HOST set for SmartBuild)"
else
  echo "warning: podman.socket not found — SmartBuild app deploys will fail." >&2
fi

echo "Applying database migrations..."
go run ./cmd/api -migrate || echo "warning: migrations failed, the API will retry on start"

exec air
