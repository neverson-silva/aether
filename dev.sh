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
export DATABASE_PORT="5432"
export DATABASE_SSL_MODE="disable"
export DATABASE_MIGRATE_ON_START="true"
export AETHER_REDIS_ADDR="127.0.0.1:16379"
export AETHER_MODE="dev"
export AETHER_COOKIE_SECURE="false"

exec air
