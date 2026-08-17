set -e

cd "$(dirname "$0")"

info()  { echo "[aether-dev] $*"; }
warn()  { echo "[aether-dev] WARN: $*" >&2; }

# Localiza o Go em qualquer local de instalação comum e expõe no PATH
if ! command -v go >/dev/null 2>&1; then
  for g in "$HOME/.local/go/bin/go" "/usr/local/go/bin/go" "$HOME/go/bin/go"; do
    if [[ -x "$g" ]]; then
      export PATH="$(dirname "$g"):$PATH"
      break
    fi
  done
fi

mkdir -p "$HOME/.local/bin"
export PATH="$PATH:$(go env GOPATH)/bin:$HOME/.local/bin"

# ---------------------------------------------------------------------------
# Dependências de dev: pack (SmartBuild/Paketo). Instalado em ~/.local/bin
# se faltar, com a MESMA versão da imagem do container.
PACK_VERSION="${PACK_VERSION:-v0.40.9}"
MACHINE="$(uname -m)"

need_bin() { command -v "$1" >/dev/null 2>&1 || [[ -x "$HOME/.local/bin/$1" ]]; }
install_pack() {
  if need_bin pack; then return 0; fi
  info "installing pack (SmartBuild/Paketo) $PACK_VERSION..."
  local pack_arch
  case "$MACHINE" in x86_64|amd64) pack_arch="linux";; arm64|aarch64) pack_arch="linux-arm64";; *) warn "pack: arch $MACHINE not supported"; return 1;; esac
  local tmp
  tmp="$(mktemp)"
  curl -fsSLo "$tmp" "https://github.com/buildpacks/pack/releases/download/$PACK_VERSION/pack-$PACK_VERSION-$pack_arch.tgz"
  tar -xzf "$tmp" -C "$HOME/.local/bin" pack
  chmod +x "$HOME/.local/bin/pack"
  rm -f "$tmp"
  info "pack installed: $($HOME/.local/bin/pack --version)"
}

install_pack

# ---------------------------------------------------------------------------
# Credenciais do banco: usa as do install.sh (~/.aether/.aether-db) quando
# existirem; senão, os padrões abaixo (usados para CRIAR o container).
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
export AETHER_REDIS_ADDR="127.0.0.1:6379"
export AETHER_RUNTIME_BACKEND="redis"
export AETHER_MODE="dev"
export AETHER_COOKIE_SECURE="false"

export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
export DOCKER_HOST="unix://$XDG_RUNTIME_DIR/podman/podman.sock"

# ---------------------------------------------------------------------------
# Infra: podman + postgres + redis (mesmos containers do install.sh).
# Cria os containers se não existirem; depois garante que estão rodando.
DEV_PG_IMAGE="${DEV_PG_IMAGE:-docker.io/library/postgres:16-alpine}"
DEV_REDIS_IMAGE="${DEV_REDIS_IMAGE:-docker.io/library/redis:7-alpine}"
DEV_REGISTRY_IMAGE="${DEV_REGISTRY_IMAGE:-docker.io/library/registry:2}"
DEV_CNB_BUILDER="${DEV_CNB_BUILDER:-127.0.0.1:5000/builder:node-spa}"

ensure_podman() {
  command -v podman >/dev/null 2>&1 || {
    warn "podman não encontrado. Instale-o ou rode ./install.sh uma vez."
    return 1
  }
  if [[ "$(uname -s)" == "Darwin" ]]; then
    podman machine inspect --format '{{.State}}' >/dev/null 2>&1 || podman machine init
    local state
    state="$(podman machine inspect --format '{{.State}}' 2>/dev/null | tr '[:upper:]' '[:lower:]' || true)"
    [[ "$state" != "running" ]] && podman machine start
  else
    systemctl --user start podman.socket >/dev/null 2>&1 || true
  fi
  info "podman ready"
}

container_exists() { podman ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "$1"; }
container_running() { podman ps --format '{{.Names}}' 2>/dev/null | grep -qx "$1"; }

ensure_postgres() {
  local name="aether-postgres"
  if container_exists "$name"; then
    container_running "$name" || podman start "$name" >/dev/null
    info "postgres container ok ($name) -> 127.0.0.1:$DATABASE_PORT"
  else
    info "creating postgres container ($name, db=$DATABASE_NAME)..."
    podman run -d --name "$name" \
      -e "POSTGRES_USER=$DATABASE_USER" \
      -e "POSTGRES_PASSWORD=$DATABASE_PASSWORD" \
      -e "POSTGRES_DB=$DATABASE_NAME" \
      -p "127.0.0.1:$DATABASE_PORT:5432" \
      --restart unless-stopped \
      "$DEV_PG_IMAGE" >/dev/null
  fi
  local tries=0
  until podman exec "$name" pg_isready -U "$DATABASE_USER" -d "$DATABASE_NAME" >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 60 ]] && { warn "postgres did not become ready in 60s."; return 1; }
    sleep 1
  done
  info "postgres ready"
}

ensure_redis() {
  local name="aether-redis"
  if container_exists "$name"; then
    container_running "$name" || podman start "$name" >/dev/null
    info "redis container ok ($name) -> 127.0.0.1:6379"
  else
    info "creating redis container ($name)..."
    podman run -d --name "$name" \
      -p "127.0.0.1:6379:6379" \
      --restart unless-stopped \
      "$DEV_REDIS_IMAGE" >/dev/null
  fi
  local tries=0
  until podman exec "$name" redis-cli ping >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 30 ]] && { warn "redis did not become ready in 30s."; return 1; }
    sleep 1
  done
  info "redis ready"
}

ensure_registry() {
  local conf="$HOME/.config/containers/registries.conf"
  if [ ! -f "$conf" ] || ! grep -q "127.0.0.1:5000" "$conf"; then
    mkdir -p "$(dirname "$conf")"
    {
      [ -f "$conf" ] && cat "$conf"
      printf '\n[[registry]]\nlocation = "127.0.0.1:5000"\ninsecure = true\n\n[[registry]]\nlocation = "localhost:5000"\ninsecure = true\n'
    } > "$conf.tmp" && mv "$conf.tmp" "$conf"
  fi
  local name="aether-registry"
  if container_exists "$name"; then
    container_running "$name" || podman start "$name" >/dev/null
    info "registry container ok ($name) -> 127.0.0.1:5000"
  else
    info "creating registry container ($name)..."
    podman run -d --name "$name" \
      -p "127.0.0.1:5000:5000" \
      --restart unless-stopped \
      "$DEV_REGISTRY_IMAGE" >/dev/null
  fi
  local tries=0
  until curl -fsS http://127.0.0.1:5000/v2/ >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 30 ]] && { warn "registry did not become ready in 30s."; return 1; }
    sleep 1
  done
  info "registry ready"
}

ensure_builder() {
  podman pull "docker.io/buildpacksio/lifecycle:${AETHER_LIFECYCLE_VERSION:-0.19.6}" >/dev/null 2>&1 || true
  if podman image exists "$DEV_CNB_BUILDER" 2>/dev/null; then
    info "CNB builder ok ($DEV_CNB_BUILDER)"
    return 0
  fi
  info "building CNB builder (Paketo + aether/spa-static)..."
  bash "$(dirname "$0")/builders/build-builder.sh" || { warn "CNB builder build falhou (deploys CNB ficam limitados)."; return 1; }
  info "CNB builder ok"
}

ensure_podman || exit 1
ensure_postgres || exit 1
ensure_redis || exit 1
ensure_registry || exit 1
ensure_builder || true

exec air
