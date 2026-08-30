#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
  exec bash "$0" "$@"
fi

set -euo pipefail

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[aether]${NC} $*"; }
warn()  { echo -e "${YELLOW}[aether]${NC} $*"; }
fail()  { progress_stop; echo -e "${RED}[aether]${NC} $*"; exit 1; }

PROGRESS_ENABLED=0
PROGRESS_CURRENT=0
PROGRESS_TOTAL=1
PROGRESS_STARTED=0
PROGRESS_LABEL=""

progress_start() {
  PROGRESS_STARTED="$(date +%s)"
  if [[ -t 1 && -z "${AETHER_PLAIN_OUTPUT:-}" ]]; then
    PROGRESS_ENABLED=1
  fi
}

progress_render() {
  [[ "$PROGRESS_ENABLED" -eq 1 ]] || return 0
  local width=28 filled remaining percent elapsed
  filled=$((PROGRESS_CURRENT * width / PROGRESS_TOTAL))
  remaining=$((width - filled))
  percent=$((PROGRESS_CURRENT * 100 / PROGRESS_TOTAL))
  elapsed=$(( $(date +%s) - PROGRESS_STARTED ))
  printf '\r\033[2K  [%3d%%] %s  (%02dm %02ds)' "$percent" "$PROGRESS_LABEL" "$((elapsed / 60))" "$((elapsed % 60))"
}

progress_step() {
  PROGRESS_CURRENT="$1"
  PROGRESS_LABEL="$2"
  if [[ "$PROGRESS_ENABLED" -eq 1 ]]; then
    progress_render
  else
    info "[$((PROGRESS_CURRENT * 100 / PROGRESS_TOTAL))%] $PROGRESS_LABEL"
  fi
}

progress_finish() {
  progress_render
  if [[ "$PROGRESS_ENABLED" -eq 1 ]]; then
    printf '\n'
  fi
}

progress_stop() {
  if [[ "$PROGRESS_ENABLED" -eq 1 ]]; then
    printf '\n'
    PROGRESS_ENABLED=0
  fi
}

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${AETHER_ENV_FILE:-$PROJECT_ROOT/.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  . "$ENV_FILE"
  set +a
fi

# ---------------------------------------------------------------------------
# Configuration — everything runs through Docker Engine.
STATE_DIR="${AETHER_STATE:-$HOME/.aether}"
NET_NAME="${AETHER_NET:-aether-net}"
INGRESS_NET_NAME="${AETHER_INGRESS_NETWORK:-aether-ingress}"

PG_CONTAINER="aether-postgres"
PG_IMAGE="${AETHER_PG_IMAGE:-docker.io/library/postgres:16-alpine}"
PG_PORT="1543"

NATS_CONTAINER="aether-nats"
NATS_IMAGE="${AETHER_NATS_IMAGE:-docker.io/library/nats:2.14.2-alpine}"
NATS_PORT="1422"
NATS_MONITOR_PORT="1822"
NATS_URL_EFFECTIVE="${AETHER_NATS_URL:-}"
NATS_USER="${AETHER_NATS_USER:-aether}"
NATS_PASSWORD="${AETHER_NATS_PASSWORD:-}"
NATS_AUTH_FILE="$STATE_DIR/keys/nats.auth"
TRAEFIK_IMAGE="${AETHER_TRAEFIK_IMAGE:-docker.io/library/traefik:v3.2}"

API_CONTAINER="aether-api"
API_IMAGE="${AETHER_API_IMAGE:-aether.local/api:1}"
API_PORT="${AETHER_API_PORT:-8080}"
WORKER_CONTAINER="aether-worker"
MONITORING_CONTAINER="aether-monitoring"
WORKER_HEALTH_PORT="1801"
MONITORING_HEALTH_PORT="1802"

WEB_CONTAINER="aether-web"
WEB_IMAGE="${AETHER_WEB_IMAGE:-aether.local/web:1}"
FRONTEND_DIR="$PROJECT_ROOT/frontend/web"
WEB_PORT=4000
DEV_MODE="${DEV_MODE:-false}"

MODE="${AETHER_MODE:-dev}"
CRED_FILE="$STATE_DIR/.aether-db"
HOST_LOG="$STATE_DIR/logs/host-setup.log"
INSTALL_LOG="${AETHER_INSTALL_LOG:-/dev/stderr}"
FORCE_API_RECREATE=0

content_fingerprint() {
  if command -v sha256sum >/dev/null 2>&1; then
    find "$@" -type f \
      ! -path '*/node_modules/*' \
      ! -path '*/dist/*' \
      ! -path '*/.git/*' \
      -print0 | LC_ALL=C sort -z | xargs -0 sha256sum | sha256sum | awk '{print $1}'
    return
  fi
  find "$@" -type f \
    ! -path '*/node_modules/*' \
    ! -path '*/dist/*' \
    ! -path '*/.git/*' \
    -print | LC_ALL=C sort | while IFS= read -r file; do
      shasum -a 256 "$file"
    done | shasum -a 256 | awk '{print $1}'
}

is_true() {
  [[ "$1" == "1" || "$1" == "true" || "$1" == "TRUE" || "$1" == "yes" ]]
}

resolve_public_url() {
	if [[ -n "${AETHER_PUBLIC_URL:-}" ]]; then
		printf '%s' "${AETHER_PUBLIC_URL%/}"
		return
	fi
	if is_true "$DEV_MODE"; then
		printf 'http://localhost:%s' "$WEB_PORT"
		return
	fi
  local host="${AETHER_PUBLIC_HOST:-}"
  if [[ -z "$host" ]] && command_exists curl; then
    for service in https://api.ipify.org https://icanhazip.com https://ifconfig.me/ip; do
      host="$(curl -fsS --max-time 3 "$service" 2>/dev/null | tr -d '[:space:]' || true)"
      [[ -n "$host" ]] && break
    done
  fi
  [[ -n "$host" ]] || host="127.0.0.1"
  printf 'http://%s:%s' "$host" "$WEB_PORT"
}

resolve_public_host() {
  if [[ -n "${AETHER_PUBLIC_HOST:-}" ]]; then
    printf '%s' "$AETHER_PUBLIC_HOST"
    return
  fi
  if [[ -n "${AETHER_PUBLIC_URL:-}" ]]; then
    local configured_host="${AETHER_PUBLIC_URL#*://}"
    configured_host="${configured_host%%/*}"
    configured_host="${configured_host%%:*}"
    [[ -n "$configured_host" ]] && printf '%s' "$configured_host" && return
  fi
  local host=""
  if command_exists curl; then
    for service in https://api.ipify.org https://icanhazip.com https://ifconfig.me/ip; do
      host="$(curl -fsS --max-time 3 "$service" 2>/dev/null | tr -d '[:space:]' || true)"
      [[ -n "$host" ]] && break
    done
  fi
  printf '%s' "${host:-127.0.0.1}"
}

command_exists() { command -v "$1" >/dev/null 2>&1; }

# host_log — registro por passo da configuração de host (Linux nativo).
host_log() {
  mkdir -p "$(dirname "$HOST_LOG")"
  printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*" >> "$HOST_LOG"
}

can_sudo() {
  [[ "$(id -u)" -eq 0 ]] && return 0
  sudo -n true 2>/dev/null
}

# run_sudo — executa como root (se já é root), via sudo -n (sem interação),
run_sudo() {
  if [[ "$(id -u)" -eq 0 ]]; then
    "$@"
  elif sudo -n true 2>/dev/null; then
    sudo -n "$@"
  else
    return 127
  fi
}

system_bus_available() {
  [[ -S /run/dbus/system_bus_socket || -S /run/systemd/private ]]
}

user_bus_available() {
  [[ -n "${XDG_RUNTIME_DIR:-}" && -S "$XDG_RUNTIME_DIR/bus" ]]
}

# ---------------------------------------------------------------------------
# RUNTIME — Docker Engine
detect_runtime() {
  if command_exists docker; then
    echo "docker"
  else
    echo "none"
  fi
}

ensure_runtime() {
  local runtime
  runtime="$(detect_runtime)"
  if [[ "$runtime" == "none" ]]; then
    warn "Docker CLI is not installed — attempting to install it automatically..."
    if install_docker; then
      runtime="$(detect_runtime)"
    else
      fail "Docker CLI is required but could not be installed. Install Docker Engine or Docker Desktop and run again."
    fi
  fi
  docker info >/dev/null 2>&1 || fail "Docker Engine is unavailable. Start Docker Desktop or the Docker daemon and run again."
  docker compose version >/dev/null 2>&1 || fail "Docker Compose is unavailable. Install the Docker Compose plugin and run again."
  ensure_linux_host
  RUNTIME="$runtime"
  DOCKER_RUNTIME="$runtime"
  info "Container runtime: Docker Engine"
}

# install_docker installs Docker CLI and Engine through the host package manager.
install_docker() {
  local distro="linux"
  if [[ -r /etc/os-release ]]; then
    . /etc/os-release
    distro="${ID:-linux}"
  fi
  local out=""
  case "$distro" in
    fedora)
      if install_docker_fedora; then
        return 0
      fi
      ;;
    ubuntu|debian|linuxmint|pop)
      if install_docker_debian; then
        return 0
      fi
      ;;
    rhel|centos|rocky|almalinux|ol|amzn)
      if install_docker_rpm "$distro"; then
        return 0
      fi
      ;;
    pacman)
      if install_docker_command "pacman -S --noconfirm --needed docker docker-compose"; then
        return 0
      fi
      ;;
    arch|manjaro)
      if install_docker_command "pacman -S --noconfirm --needed docker docker-compose"; then
        return 0
      fi
      ;;
    opensuse*|sles)
      if install_docker_command "zypper --non-interactive install docker docker-compose"; then
        return 0
      fi
      ;;
    alpine)
      if install_docker_command "apk add docker docker-cli-compose"; then
        return 0
      fi
      ;;
  esac
  if command_exists curl; then
    local script
    script="$(mktemp)"
    host_log "step: install Docker using the official installer"
    if curl -fsSL https://get.docker.com -o "$script" && run_sudo sh "$script" > /tmp/aether-docker-install.log 2>&1; then
      rm -f "$script"
      host_log "ok: Docker installed using the official installer"
      info "  Docker installed using the official installer."
      start_docker_service
      return 0
    fi
    out="$(tail -40 /tmp/aether-docker-install.log 2>/dev/null || true)"
    rm -f "$script"
  fi
  host_log "fail: Docker installation on $distro ($out)"
  return 1
}

install_docker_command() {
  local command="$1"
  host_log "step: install Docker -> $command"
  local out
  if out="$(run_sudo sh -c "$command" 2>&1)"; then
    host_log "ok: Docker installed via package manager"
    info "  Docker installed via package manager."
    start_docker_service
    return 0
  fi
  host_log "fail: Docker package installation -> $command ($out)"
  return 1
}

install_docker_fedora() {
  local repo_command=""
  if dnf config-manager addrepo --help >/dev/null 2>&1; then
    repo_command="dnf config-manager addrepo --from-repofile=https://download.docker.com/linux/fedora/docker-ce.repo"
  else
    repo_command="dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo"
  fi
  install_docker_command "dnf install -y dnf-plugins-core ca-certificates && $repo_command && dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"
}

install_docker_rpm() {
  local distro="$1"
  local repo_base="centos"
  [[ "$distro" == "amzn" ]] && repo_base="amazon"
  local package_manager="dnf"
  local plugins_package="dnf-plugins-core"
  [[ "$distro" == "amzn" ]] && package_manager="yum"
  [[ "$package_manager" == "yum" ]] && plugins_package="yum-utils"
  local repo_command=""
  if command_exists dnf && dnf config-manager addrepo --help >/dev/null 2>&1; then
    repo_command="dnf config-manager addrepo --from-repofile=https://download.docker.com/linux/$repo_base/docker-ce.repo"
  elif command_exists dnf; then
    repo_command="dnf config-manager --add-repo https://download.docker.com/linux/$repo_base/docker-ce.repo"
  else
    repo_command="yum-config-manager --add-repo https://download.docker.com/linux/$repo_base/docker-ce.repo"
  fi
  install_docker_command "$package_manager install -y $plugins_package ca-certificates && $repo_command && $package_manager install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"
}

install_docker_debian() {
  install_docker_command "apt-get update -qq && apt-get install -y ca-certificates curl gnupg && install -m 0755 -d /etc/apt/keyrings /etc/apt/sources.list.d && curl -fsSL https://download.docker.com/linux/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && chmod a+r /etc/apt/keyrings/docker.gpg && . /etc/os-release && printf 'deb [arch=%s signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/%s %s stable\\n' \"$(dpkg --print-architecture)\" \"\${ID}\" \"\${VERSION_CODENAME}\" > /etc/apt/sources.list.d/docker.list && apt-get update -qq && apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin"
}

start_docker_service() {
  if command_exists systemctl; then
    run_sudo systemctl enable --now docker >/dev/null 2>&1 || true
    run_sudo systemctl enable --now containerd >/dev/null 2>&1 || true
  elif command_exists service; then
    run_sudo service docker start >/dev/null 2>&1 || true
  fi
}

# ensure_linux_host validates Docker host prerequisites.
ensure_linux_host() {
  host_log "docker info verified"
  if [[ "$(uname -s)" == "Linux" && "$(id -u)" -ne 0 && ! -w /var/run/docker.sock ]]; then
    warn "Docker socket permissions may require membership in the docker group."
  fi
}

# ---------------------------------------------------------------------------
# DATABASE + NATS containers
strong_password() {
  if command_exists openssl; then
    openssl rand -base64 24 | tr -d '/+=' | head -c 24
  else
    head -c 24 /dev/urandom | base64 | tr -d '/+=' | head -c 24
  fi
}

# load_db_credentials — gera automaticamente user/password/database e persiste
load_db_credentials() {
  DB_USER="${DATABASE_USER:-}"
  DB_NAME="${DATABASE_NAME:-}"
  DB_PASSWORD="${DATABASE_PASSWORD:-}"

  if [[ -f "$CRED_FILE" ]]; then
    # shellcheck disable=SC1090
    . "$CRED_FILE"
    DB_USER="${DATABASE_USER:-$DB_USER}"
    DB_NAME="${DATABASE_NAME:-$DB_NAME}"
    DB_PASSWORD="${DATABASE_PASSWORD:-$DB_PASSWORD}"
  elif [[ -z "$DB_USER" && -z "$DB_PASSWORD" ]]; then
    local exists
    exists="$($RUNTIME ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$PG_CONTAINER" || true)"
    if [[ -n "$exists" ]]; then
      local env_vars
      env_vars="$($RUNTIME inspect "$PG_CONTAINER" --format '{{range .Config.Env}}{{println .}}{{end}}' 2>/dev/null || true)"
      DB_USER="$(printf '%s\n' "$env_vars" | sed -n 's/^POSTGRES_USER=//p' | head -1)"
      DB_NAME="$(printf '%s\n' "$env_vars" | sed -n 's/^POSTGRES_DB=//p' | head -1)"
      DB_PASSWORD="$(printf '%s\n' "$env_vars" | sed -n 's/^POSTGRES_PASSWORD=//p' | head -1)"
    fi
  fi

  if [[ -z "$DB_USER" ]]; then
    DB_USER="aether_$(head -c 4 /dev/urandom | base64 | tr -d '/+=' | head -c 6)"
  fi
  if [[ -z "$DB_NAME" ]]; then
    DB_NAME="aether_$(head -c 4 /dev/urandom | base64 | tr -d '/+=' | head -c 6)"
  fi
  if [[ -z "$DB_PASSWORD" ]]; then
    DB_PASSWORD="$(strong_password)"
  fi

  PG_PORT="1543"

  mkdir -p "$STATE_DIR"
  umask 077
  cat > "$CRED_FILE" <<EOF
DB_USER='$DB_USER'
DB_NAME='$DB_NAME'
DB_PASSWORD='$DB_PASSWORD'
PG_PORT='$PG_PORT'
NATS_PORT='$NATS_PORT'
EOF
  chmod 600 "$CRED_FILE"
  info "Application data credentials initialized securely."
}

ensure_network() {
  local network
  local missing=0
  for network in "$NET_NAME" "$INGRESS_NET_NAME"; do
    if $RUNTIME network inspect "$network" >/dev/null 2>&1; then
      continue
    fi
    missing=1
    info "Preparing the application network '$network'..."
    if ! $RUNTIME network create "$network" >/dev/null 2>&1 && ! $RUNTIME network inspect "$network" >/dev/null 2>&1; then
      fail "Could not prepare the application network '$network'."
    fi
  done
  if [[ "$missing" -eq 0 ]]; then
    info "Application networks already prepared."
  else
    info "Application networks prepared."
  fi
}

ensure_ingress_image() {
  if $RUNTIME image inspect "$TRAEFIK_IMAGE" >/dev/null 2>&1; then
    info "Ingress image already available."
    return 0
  fi
  info "Preparing the ingress image..."
  if ! $RUNTIME pull "$TRAEFIK_IMAGE" >>"$INSTALL_LOG" 2>&1; then
    fail "Could not download the ingress image '$TRAEFIK_IMAGE'."
  fi
  info "Ingress image prepared."
}

ensure_postgres() {
  local runtime="$RUNTIME"
  local expected_port="1543"
  PG_PORT="$expected_port"
  load_db_credentials
  local password="$DB_PASSWORD"

  local exists
  exists="$($runtime ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$PG_CONTAINER" || true)"
  if [[ -n "$exists" ]]; then
    local published_port
    published_port="$($runtime port "$PG_CONTAINER" 5432/tcp 2>/dev/null | sed -nE 's/.*:([0-9]+)$/\1/p' | head -1)"
    if [[ -n "$published_port" && "$published_port" != "$PG_PORT" ]]; then
      info "PostgreSQL is exposed on 127.0.0.1:$published_port — moving it to 127.0.0.1:$PG_PORT."
      published_port=""
    fi
    if [[ -n "$published_port" ]]; then
      PG_PORT="$published_port"
    else
      local volume_name
      volume_name="$($runtime inspect "$PG_CONTAINER" --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql/data"}}{{.Name}}{{end}}{{end}}' 2>/dev/null || true)"
      if [[ "$volume_name" != "aether-pg-data" ]]; then
        fail "PostgreSQL container '$PG_CONTAINER' has no host port and does not use the expected aether-pg-data volume. Refusing to recreate it automatically."
      fi
      info "PostgreSQL container '$PG_CONTAINER' has no host port — recreating it with the existing aether-pg-data volume."
      $runtime rm -f "$PG_CONTAINER" >/dev/null
      $runtime run -d \
        --name "$PG_CONTAINER" \
        --network "$NET_NAME" \
        --network-alias "$PG_CONTAINER" \
        -e "POSTGRES_USER=$DB_USER" \
        -e "POSTGRES_PASSWORD=$password" \
        -e "POSTGRES_DB=$DB_NAME" \
        -p "$PG_PORT:5432" \
        -v aether-pg-data:/var/lib/postgresql/data \
        --restart unless-stopped \
        "$PG_IMAGE" >/dev/null || fail "Failed to recreate the PostgreSQL container."
      info "PostgreSQL recreated on 127.0.0.1:$PG_PORT using the existing database volume."
    fi
    if [[ -n "$published_port" ]]; then
      info "PostgreSQL already exists ($PG_CONTAINER) — using the existing one."
      local running
      running="$($runtime ps --format '{{.Names}}' 2>/dev/null | grep -cx "$PG_CONTAINER" || true)"
      [[ "$running" -eq 0 ]] && $runtime start "$PG_CONTAINER"
    fi
    $runtime exec -e "PGPASSWORD=$password" "$PG_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "ALTER ROLE \"$DB_USER\" PASSWORD '$password';" >/dev/null 2>&1 \
      && info "Password synced in PostgreSQL." \
      || warn "Could not sync the password — check the container."
  else
    if $runtime volume inspect aether-pg-data >/dev/null 2>&1; then
      info "Existing PostgreSQL data volume found — reusing it without replacing data."
    else
      info "Initializing application data storage..."
    fi
    $runtime run -d \
      --name "$PG_CONTAINER" \
      --network "$NET_NAME" \
      --network-alias "$PG_CONTAINER" \
      -e "POSTGRES_USER=$DB_USER" \
      -e "POSTGRES_PASSWORD=$password" \
      -e "POSTGRES_DB=$DB_NAME" \
      -p "$PG_PORT:5432" \
      -v aether-pg-data:/var/lib/postgresql/data \
      --restart unless-stopped \
      "$PG_IMAGE" >/dev/null
    info "Application data storage started."
  fi

  info "Waiting for PostgreSQL to become healthy..."
  local tries=0
  until $runtime exec "$PG_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 60 ]] && fail "PostgreSQL did not become ready in 60s."
    sleep 2
  done
  if ! $runtime exec -e "PGPASSWORD=$password" "$PG_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -Atqc 'SELECT 1' >/dev/null 2>&1; then
    fail "PostgreSQL is running, but the configured credentials do not match its existing data volume. Refusing to start the API. Preserve the volume and recover or explicitly reset it before retrying."
  fi
  info "Application data storage ready."
}

ensure_nats() {
	ensure_nats_auth
	if [[ -n "${AETHER_NATS_URL:-}" ]]; then
    info "Application messaging already configured — reusing existing configuration."
    NATS_URL_EFFECTIVE="$AETHER_NATS_URL"
    return 0
  fi
  local runtime="$RUNTIME"
  local exists
  exists="$($runtime ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$NATS_CONTAINER" || true)"
  if [[ -n "$exists" ]]; then
    local configured_port configured_monitor_port
    configured_port="$($runtime port "$NATS_CONTAINER" 4222/tcp 2>/dev/null | sed -nE 's/.*:([0-9]+)$/\1/p' | head -1)"
    configured_monitor_port="$($runtime port "$NATS_CONTAINER" 8222/tcp 2>/dev/null | sed -nE 's/.*:([0-9]+)$/\1/p' | head -1)"
    if [[ "$configured_port" != "$NATS_PORT" || "$configured_monitor_port" != "$NATS_MONITOR_PORT" ]]; then
      info "Migrating NATS to the internal port range."
      $runtime rm -f "$NATS_CONTAINER" >/dev/null
      exists=""
    fi
  fi
  if [[ -n "$exists" ]]; then
    local configured_image
    configured_image="$($runtime inspect "$NATS_CONTAINER" --format '{{.Config.Image}}' 2>/dev/null || true)"
    if [[ "$configured_image" != "$NATS_IMAGE" ]]; then
      info "Upgrading NATS container from ${configured_image:-unknown} to $NATS_IMAGE..."
      $runtime rm -f "$NATS_CONTAINER" >/dev/null
      exists=""
    fi
    if [[ -n "$exists" ]]; then
      local configured_cmd
      configured_cmd="$($runtime inspect "$NATS_CONTAINER" --format '{{json .Config.Cmd}}' 2>/dev/null || true)"
      if [[ "$configured_cmd" != *"--user"* ]]; then
        info "Recreating NATS with authentication enabled..."
        $runtime rm -f "$NATS_CONTAINER" >/dev/null
        exists=""
      fi
    fi
  fi
  if [[ -n "$exists" ]]; then
    info "Application messaging already prepared — reusing the existing service."
    local running
    running="$($runtime ps --format '{{.Names}}' 2>/dev/null | grep -cx "$NATS_CONTAINER" || true)"
    [[ "$running" -eq 0 ]] && $runtime start "$NATS_CONTAINER"
  else
    info "Preparing application messaging..."
    $runtime run -d \
      --name "$NATS_CONTAINER" \
      --network "$NET_NAME" \
      --network-alias "$NATS_CONTAINER" \
      -p "$NATS_PORT:4222" \
      -p "$NATS_MONITOR_PORT:8222" \
      -v aether-nats-data:/data \
      --restart unless-stopped \
      "$NATS_IMAGE" -js -sd /data -m 8222 --user "$NATS_USER" --pass "$NATS_PASSWORD" >/dev/null
    info "NATS started on 127.0.0.1:$NATS_PORT."
  fi
  local tries=0
  until $runtime exec "$NATS_CONTAINER" nc -z 127.0.0.1 4222 >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 30 ]] && fail "NATS did not become ready in 30s."
    sleep 2
  done
  export AETHER_NATS_URL="nats://$NATS_CONTAINER:4222"
  NATS_URL_EFFECTIVE="$AETHER_NATS_URL"
  info "Application messaging ready."
}

ensure_nats_auth() {
  if [[ -z "$NATS_PASSWORD" && -f "$NATS_AUTH_FILE" ]]; then
    NATS_PASSWORD="$(sed -n '2p' "$NATS_AUTH_FILE")"
  fi
  if [[ -z "$NATS_PASSWORD" ]]; then
    mkdir -p "$(dirname "$NATS_AUTH_FILE")"
    umask 077
    if command_exists openssl; then
      NATS_PASSWORD="$(openssl rand -hex 24)"
    else
      NATS_PASSWORD="$(date +%s)-$(head -c 16 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 24)"
    fi
    printf '%s\n%s\n' "$NATS_USER" "$NATS_PASSWORD" > "$NATS_AUTH_FILE"
  fi
  export AETHER_NATS_USER="$NATS_USER"
  export AETHER_NATS_PASSWORD="$NATS_PASSWORD"
}

# ---------------------------------------------------------------------------
# MASTER KEY
ensure_master_key() {
  local keydir="$STATE_DIR/keys"
  local keyfile="$keydir/master.key"
  if [[ -f "$keyfile" ]]; then
    info "Security credentials already initialized — preserving existing credentials."
  else
    info "Initializing secure application credentials..."
    mkdir -p "$keydir"
    umask 077
    if command_exists openssl; then
      openssl rand -hex 32 > "$keyfile"
    else
      head -c 32 /dev/urandom | xxd -p | tr -d '\n' > "$keyfile"
      printf '\n' >> "$keyfile"
    fi
    chmod 600 "$keyfile"
    info "Secure application credentials initialized."
  fi
}

# ---------------------------------------------------------------------------
# BUILD + RUN — API e frontend em containers
build_api_image() {
  [[ -f "$PROJECT_ROOT/infra/Dockerfile" ]] || fail "Dockerfile not found. Run from inside the project directory."
  [[ -f "$PROJECT_ROOT/api/cmd/api/main.go" ]] || fail "API source not found in ./api/cmd/api."
  local image_stamp="$STATE_DIR/.api-image.stamp"
  local source_stamp
  source_stamp="$(content_fingerprint "$PROJECT_ROOT/api" "$PROJECT_ROOT/infra/Dockerfile" "$PROJECT_ROOT/.dockerignore")|$API_IMAGE"
  if $RUNTIME image inspect "$API_IMAGE" >/dev/null 2>&1 && [[ -f "$image_stamp" ]] && grep -Fxq "$source_stamp" "$image_stamp"; then
    info "Application services unchanged — reusing the cached image."
    return 0
  fi
  info "Preparing the application services..."
  mkdir -p "$(dirname "$INSTALL_LOG")"
  $RUNTIME build -t "$API_IMAGE" -f "$PROJECT_ROOT/infra/Dockerfile" "$PROJECT_ROOT" \
    >>"$INSTALL_LOG" 2>&1 || fail "The application services could not be prepared."
  mkdir -p "$STATE_DIR"
  printf '%s\n' "$source_stamp" > "$image_stamp"
  info "Application services prepared."
}

ensure_web_image() {
  [[ -f "$FRONTEND_DIR/package.json" ]] || fail "Frontend package.json not found."
  [[ -f "$PROJECT_ROOT/infra/web.Dockerfile" ]] || fail "Web Dockerfile not found."
  local build_env="$FRONTEND_DIR/.env.production.local"
  local backup_env=""
  if [[ -e "$build_env" ]]; then
    backup_env="$(mktemp "${TMPDIR:-/tmp}/aether-web-env.XXXXXX")"
    mv "$build_env" "$backup_env"
  fi
  printf 'VITE_API_TARGET="%s"\nVITE_AETHER_PUBLIC_URL="%s"\n' "$AETHER_API_PUBLIC_URL" "$AETHER_PUBLIC_URL" > "$build_env"
  cleanup_web_build_env() {
    rm -f "$build_env"
    if [[ -n "$backup_env" ]]; then
      mv "$backup_env" "$build_env"
    fi
  }
  local image_stamp="$STATE_DIR/.web-image.stamp"
  local source_stamp
  source_stamp="$(content_fingerprint "$PROJECT_ROOT/frontend/aether_ds" "$PROJECT_ROOT/frontend/web" "$PROJECT_ROOT/infra/web.Dockerfile" "$PROJECT_ROOT/infra/nginx.conf")|$WEB_IMAGE|$AETHER_API_PUBLIC_URL|$AETHER_PUBLIC_URL"
  if $RUNTIME image inspect "$WEB_IMAGE" >/dev/null 2>&1 && [[ -f "$image_stamp" ]] && grep -Fxq "$source_stamp" "$image_stamp"; then
    cleanup_web_build_env
    info "Application interface unchanged — reusing the cached image."
    return 0
  fi
  info "Preparing the application interface..."
  mkdir -p "$(dirname "$INSTALL_LOG")"
  if ! $RUNTIME build -t "$WEB_IMAGE" -f "$PROJECT_ROOT/infra/web.Dockerfile" "$PROJECT_ROOT" >>"$INSTALL_LOG" 2>&1; then
    cleanup_web_build_env
    fail "The application interface could not be prepared."
  fi
  cleanup_web_build_env
  mkdir -p "$STATE_DIR"
  printf '%s\n' "$source_stamp" > "$image_stamp"
  info "Application interface prepared."
}

api_exists() {
  $RUNTIME ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "$API_CONTAINER"
}

remove_api_container() {
  $RUNTIME rm -f "$API_CONTAINER" >/dev/null 2>&1 || true
  local attempts=0
  while api_exists; do
    attempts=$((attempts + 1))
    [[ "$attempts" -ge 20 ]] && fail "Could not release the API container name '$API_CONTAINER'."
    sleep 0.25
  done
}

api_running() {
  $RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -qx "$API_CONTAINER"
}

start_api() {
  if api_running && [[ "$FORCE_API_RECREATE" -eq 0 ]]; then
    info "API container already running."
    return 0
  fi

  build_api_image

  mkdir -p "$STATE_DIR" "$STATE_DIR/data" "$STATE_DIR/certs" "$STATE_DIR/logs" \
    "$STATE_DIR/builds" "$STATE_DIR/cache" "$STATE_DIR/keys" "$STATE_DIR/logs/apps" \
    "$STATE_DIR/builds/sources" "$STATE_DIR/snapshots"

  $RUNTIME volume create aether-traefik >/dev/null 2>&1 || true
  $RUNTIME volume create aether-pack-cache >/dev/null 2>&1 || true

  if api_exists; then
    remove_api_container
  fi

  info "Starting API container ($API_CONTAINER) on $AETHER_API_PUBLIC_URL..."
  local args=(run -d)
  if [[ -f "$ENV_FILE" ]]; then
    args+=(--env-file "$ENV_FILE")
  fi
  args+=(
    --name "$API_CONTAINER"
    --network "$NET_NAME"
    --network-alias "$API_CONTAINER"
    --security-opt label=disable
    -p "$API_PORT:8080"
    -v "$STATE_DIR:/var/lib/aether"
    -v "aether-traefik:/var/lib/aether/traefik"
    -v "aether-pack-cache:/root/.cache/pack"
    --restart unless-stopped
    -e "AETHER_STATE=/var/lib/aether"
    -e "AETHER_SNAPSHOT_HOST_DIR=$STATE_DIR/snapshots"
    -e "AETHER_API_ADDR=0.0.0.0:8080"
    -e "DATABASE_HOST=$PG_CONTAINER"
    -e "DATABASE_PORT=5432"
    -e "DATABASE_NAME=$DB_NAME"
    -e "DATABASE_USER=$DB_USER"
    -e "DATABASE_PASSWORD=$DB_PASSWORD"
    -e "DATABASE_SSL_MODE=disable"
    -e "DATABASE_MIGRATE_ON_START=true"
    -e "AETHER_NATS_URL=$NATS_URL_EFFECTIVE"
    -e "AETHER_NATS_USER=$NATS_USER"
    -e "AETHER_NATS_PASSWORD=$NATS_PASSWORD"
    -e "AETHER_RUNTIME_BACKEND=nats"
    -e "AETHER_CNB_BUILDER=$CNB_BUILDER"
    -e "AETHER_PUBLIC_URL=$AETHER_PUBLIC_URL"
    -e "DEV_MODE=$DEV_MODE"
    -e "AETHER_FREE_DOMAIN_PROVIDER=${AETHER_FREE_DOMAIN_PROVIDER:-nip.io}"
    -e "AETHER_TRAEFIK_IMAGE=$TRAEFIK_IMAGE"
    -e "AETHER_COOKIE_SECURE=${AETHER_COOKIE_SECURE:-false}"
    -e "AETHER_MODE=$MODE"
  )
  local sock_mount=0
  local docker_socket="${AETHER_DOCKER_SOCKET:-/var/run/docker.sock}"
  if [[ -S "$docker_socket" ]]; then
    args+=( -v "$docker_socket:/var/run/docker.sock:ro" )
    args+=( -e "DOCKER_HOST=unix:///var/run/docker.sock" -e "AETHER_BUILD_DOCKER_HOST=unix:///var/run/docker.sock" )
    sock_mount=1
    info "  Docker socket mounted for application deployments."
  fi
  if [[ "$sock_mount" -eq 0 ]]; then
    warn "  Docker socket is not mountable; application deployments will be unavailable."
  fi

  local api_started=0
  if $RUNTIME "${args[@]}" "$API_IMAGE" >/dev/null 2>&1; then
    api_started=1
  else
    remove_api_container
    sleep 1
    if $RUNTIME "${args[@]}" "$API_IMAGE" >/dev/null 2>&1; then
      api_started=1
    fi
  fi
  if [[ "$api_started" -eq 0 ]]; then
    if [[ "$sock_mount" -eq 1 ]]; then
      warn "  Docker socket mount failed — retrying without it."
      remove_api_container
      local clean_args=()
      local index=0
      while [[ "$index" -lt "${#args[@]}" ]]; do
        local argument="${args[$index]}"
        if [[ "$argument" == "-v" && "$((index + 1))" -lt "${#args[@]}" && "${args[$((index + 1))]}" == *"docker.sock:ro" ]]; then
          index=$((index + 2))
          continue
        fi
        if [[ "$argument" == "-e" && "$((index + 1))" -lt "${#args[@]}" && ( "${args[$((index + 1))]}" == "DOCKER_HOST=unix://"* || "${args[$((index + 1))]}" == "AETHER_BUILD_DOCKER_HOST=unix://"* ) ]]; then
          index=$((index + 2))
          continue
        fi
        clean_args+=( "$argument" )
        index=$((index + 1))
      done
      $RUNTIME "${clean_args[@]}" "$API_IMAGE" >/dev/null || fail "Failed to start the API container."
    else
      fail "Failed to start the API container."
    fi
  fi

  if [[ "$INGRESS_NET_NAME" != "$NET_NAME" ]]; then
    if ! $RUNTIME network connect "$INGRESS_NET_NAME" "$API_CONTAINER" >/dev/null 2>&1; then
      if ! $RUNTIME network inspect "$INGRESS_NET_NAME" >/dev/null 2>&1; then
        fail "The ingress network '$INGRESS_NET_NAME' is unavailable."
      fi
      if ! $RUNTIME inspect "$API_CONTAINER" --format '{{json .NetworkSettings.Networks}}' 2>/dev/null | grep -q "\"$INGRESS_NET_NAME\""; then
        fail "Could not connect the API container to the ingress network '$INGRESS_NET_NAME'."
      fi
    fi
  fi

  local tries=0
  until curl -fsS "http://127.0.0.1:$API_PORT/api/v1/ready" >/dev/null 2>&1; do
    tries=$((tries + 1))
    if ! api_running; then
      fail "The API container exited during boot. See: $RUNTIME logs $API_CONTAINER"
    fi
    [[ $tries -gt 90 ]] && fail "API did not respond in 180s. See: $RUNTIME logs $API_CONTAINER"
    sleep 2
  done
  info "  ✓ API healthy on http://127.0.0.1:$API_PORT"
}

start_auxiliary() {
  local container="$1"
  local binary="$2"
  $RUNTIME rm -f "$container" >/dev/null 2>&1 || true
  local args=(run -d --name "$container" --entrypoint "/usr/local/bin/$binary" --network "$NET_NAME" --security-opt label=disable)
  args+=(
    -v "$STATE_DIR:/var/lib/aether"
    -v "aether-traefik:/var/lib/aether/traefik"
    -e "AETHER_STATE=/var/lib/aether"
    -e "AETHER_SNAPSHOT_HOST_DIR=$STATE_DIR/snapshots"
    -e "DATABASE_HOST=$PG_CONTAINER"
    -e "DATABASE_PORT=5432"
    -e "DATABASE_NAME=$DB_NAME"
    -e "DATABASE_USER=$DB_USER"
    -e "DATABASE_PASSWORD=$DB_PASSWORD"
    -e "DATABASE_SSL_MODE=disable"
    -e "AETHER_NATS_URL=$NATS_URL_EFFECTIVE"
    -e "AETHER_NATS_USER=$NATS_USER"
    -e "AETHER_NATS_PASSWORD=$NATS_PASSWORD"
    -e "AETHER_RUNTIME_BACKEND=nats"
    -e "AETHER_CNB_BUILDER=$CNB_BUILDER"
    -e "AETHER_PUBLIC_URL=$AETHER_PUBLIC_URL"
    -e "DEV_MODE=$DEV_MODE"
    -e "AETHER_FREE_DOMAIN_PROVIDER=${AETHER_FREE_DOMAIN_PROVIDER:-nip.io}"
    -e "AETHER_MODE=$MODE"
    --restart unless-stopped
  )
  if [[ "$binary" == "aether-worker" ]]; then
    args+=( -e "AETHER_WORKER_HEALTH_ADDR=0.0.0.0:8081" -p "127.0.0.1:$WORKER_HEALTH_PORT:8081" )
  elif [[ "$binary" == "aether-monitoring" ]]; then
    args+=( -e "AETHER_MONITORING_HEALTH_ADDR=0.0.0.0:8082" -p "127.0.0.1:$MONITORING_HEALTH_PORT:8082" )
  fi
  if [[ "$binary" == "aether-worker" ]]; then
    local docker_socket="${AETHER_DOCKER_SOCKET:-/var/run/docker.sock}"
    if [[ -S "$docker_socket" ]]; then
      args+=( -v "$docker_socket:/var/run/docker.sock:ro" )
      args+=( -e "DOCKER_HOST=unix:///var/run/docker.sock" -e "AETHER_BUILD_DOCKER_HOST=unix:///var/run/docker.sock" )
    else
      warn "Docker Engine socket is unavailable; the deployment worker cannot build or run images."
    fi
  else
    local docker_socket="${AETHER_DOCKER_SOCKET:-/var/run/docker.sock}"
    if [[ -S "$docker_socket" ]]; then
      args+=( -v "$docker_socket:/var/run/docker.sock:ro" )
      args+=( -e "DOCKER_HOST=unix:///var/run/docker.sock" -e "AETHER_BUILD_DOCKER_HOST=unix:///var/run/docker.sock" )
    fi
  fi
  info "Starting $container..."
  $RUNTIME "${args[@]}" "$API_IMAGE" >/dev/null || fail "Failed to start $container."
  if [[ "$binary" == "aether-worker" && "$INGRESS_NET_NAME" != "$NET_NAME" ]]; then
    if ! $RUNTIME network connect "$INGRESS_NET_NAME" "$container" >/dev/null 2>&1; then
      if ! $RUNTIME inspect "$container" --format '{{json .NetworkSettings.Networks}}' 2>/dev/null | grep -q "\"$INGRESS_NET_NAME\""; then
        fail "Could not connect the deployment worker to the ingress network '$INGRESS_NET_NAME'."
      fi
    fi
  fi
}

start_workers() {
  start_auxiliary "$WORKER_CONTAINER" aether-worker
  start_auxiliary "$MONITORING_CONTAINER" aether-monitoring
}

# ---------------------------------------------------------------------------
# WEB CONTAINER (nginx serves the frontend on port 4000)
write_nginx_conf() {
  local conf="$STATE_DIR/nginx-aether.conf"
  cat > "$conf" <<EOF
server {
    listen 4000;
    server_name _;

    client_max_body_size 128m;

    location /api/ {
        proxy_pass http://$API_CONTAINER:8080;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 300s;
    }

    location / {
        root /usr/share/nginx/html;
        try_files \$uri \$uri/ /index.html;
    }
}
EOF
  printf '%s\n' "${GREEN}[aether]${NC} Application gateway configuration prepared." >&2
  echo "$conf"
}

start_web() {
  local conf
  conf="$(write_nginx_conf)"

  $RUNTIME rm -f "$WEB_CONTAINER" >/dev/null 2>&1 || true

  info "Starting web gateway ($WEB_CONTAINER) on 127.0.0.1:$WEB_PORT..."
  $RUNTIME run -d \
    --name "$WEB_CONTAINER" \
    --network "$NET_NAME" \
    -p "$WEB_PORT:4000" \
    -v "$conf:/etc/nginx/conf.d/default.conf:ro" \
    --restart unless-stopped \
    "$WEB_IMAGE" >/dev/null || fail "Failed to start the web container."

  local tries=0
  until curl -fsS "http://127.0.0.1:$WEB_PORT/" >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 30 ]] && fail "Web gateway did not become ready in 60s."
    sleep 2
  done
  info "  ✓ web gateway healthy on http://127.0.0.1:$WEB_PORT"
}

# ---------------------------------------------------------------------------
# REGISTRY + CNB BUILDER — local registry and Docker-backed CNB builder.
REGISTRY_IMAGE="${AETHER_REGISTRY_IMAGE:-docker.io/library/registry:2}"
REGISTRY_ADDR="127.0.0.1:1500"
REGISTRY_CONTAINER="aether-registry"
DOCKER_RUNTIME="${AETHER_DOCKER_CLI:-docker}"
CNB_BUILDER="127.0.0.1:1500/builder:node-spa"
if [[ "$CNB_BUILDER" == "aether/builder:node-spa" || "$CNB_BUILDER" == "localhost/aether/builder:node-spa" ]]; then
  CNB_BUILDER="127.0.0.1:1500/builder:node-spa"
fi

ensure_registry() {
  command -v "$DOCKER_RUNTIME" >/dev/null || fail "Docker CLI is required for the image registry."
  "$DOCKER_RUNTIME" info >/dev/null 2>&1 || fail "Docker Engine is unavailable for the image registry."
  local exists
  exists="$($DOCKER_RUNTIME ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$REGISTRY_CONTAINER" || true)"
  if [[ -n "$exists" ]]; then
    local configured_port
    configured_port="$($DOCKER_RUNTIME port "$REGISTRY_CONTAINER" 5000/tcp 2>/dev/null | sed -nE 's/.*:([0-9]+)$/\1/p' | head -1)"
    if [[ "$configured_port" != "1500" ]]; then
      info "Migrating the internal registry to port 1500."
      $DOCKER_RUNTIME rm -f "$REGISTRY_CONTAINER" >/dev/null
      exists=""
    fi
  fi
  if [[ -n "$exists" ]]; then
    local running
    running="$($DOCKER_RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -cx "$REGISTRY_CONTAINER" || true)"
    [[ "$running" -eq 0 ]] && $DOCKER_RUNTIME start "$REGISTRY_CONTAINER" >/dev/null
    info "Registry already exists ($REGISTRY_CONTAINER) — using the existing one."
  else
    info "Creating registry ($REGISTRY_IMAGE) on $REGISTRY_ADDR..."
    $DOCKER_RUNTIME run -d \
      --name "$REGISTRY_CONTAINER" \
      -p "$REGISTRY_ADDR:5000" \
      --restart unless-stopped \
      "$REGISTRY_IMAGE" >/dev/null
  fi
  local tries=0
  until curl -fsS "http://$REGISTRY_ADDR/v2/" >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 30 ]] && warn "Registry did not become ready in 30s."
    sleep 1
  done
  info "Registry ready on $REGISTRY_ADDR."
}

ensure_builder() {
  command -v "$DOCKER_RUNTIME" >/dev/null || fail "Docker CLI is required for the CNB builder."
  local lifecycle_image="docker.io/buildpacksio/lifecycle:${AETHER_LIFECYCLE_VERSION:-0.21.17}"
  local run_image="${AETHER_CNB_RUN_IMAGE:-docker.io/library/ubuntu:24.04}"
  if ! "$DOCKER_RUNTIME" image inspect "$lifecycle_image" >/dev/null 2>&1; then
    "$DOCKER_RUNTIME" pull "$lifecycle_image" >/dev/null 2>&1 || true
  fi
  if ! "$DOCKER_RUNTIME" image inspect "$run_image" >/dev/null 2>&1; then
    "$DOCKER_RUNTIME" pull "$run_image" >/dev/null 2>&1 || true
  fi
  local source_stamp builder_stamp
  source_stamp="$(content_fingerprint "$PROJECT_ROOT/infra/buildpacks")|$CNB_BUILDER|$run_image|$lifecycle_image|${AETHER_BUILDER_BASE_IMAGE:-}"
  builder_stamp="$STATE_DIR/cnb-builder.stamp"
  if "$DOCKER_RUNTIME" image inspect "$CNB_BUILDER" >/dev/null 2>&1 && [[ -f "$builder_stamp" ]] && grep -qx "$source_stamp" "$builder_stamp"; then
    info "CNB builder already present ($CNB_BUILDER)."
    return 0
  fi
  info "Building CNB builder ($CNB_BUILDER) — aether buildpacks + ubuntu run image..."
  mkdir -p "$(dirname "$INSTALL_LOG")"
  bash "$PROJECT_ROOT/infra/buildpacks/builders/build-builder.sh" >>"$INSTALL_LOG" 2>&1 || fail "The application build environment could not be prepared."
  "$DOCKER_RUNTIME" image inspect "$CNB_BUILDER" >/dev/null 2>&1 || fail "CNB builder image is not available as $CNB_BUILDER."
  mkdir -p "$STATE_DIR"
  printf '%s\n' "$source_stamp" > "$builder_stamp"
  info "CNB builder ready."
}

# ---------------------------------------------------------------------------
# HOST AGENT (host machine — macOS or Linux)
# The API runs inside a container; the host agent runs natively on the host
# and writes the real host metrics to $STATE_DIR/host-stats.json
# (mounted into the API container at /var/lib/aether). The watchdog couples
# the agent to the API lifecycle: agent starts when the API is up, and is
# terminated whenever the API goes down for any reason.

start_agent() {
  [[ -x "$PROJECT_ROOT/infra/scripts/host-watchdog.sh" ]] || return 0
  local pidfile="$STATE_DIR/host-agent.pid"
  if [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile" 2>/dev/null)" 2>/dev/null; then
    info "Host watchdog already running (pid $(cat "$pidfile"))."
    return 0
  fi
  mkdir -p "$STATE_DIR/logs"
  nohup bash "$PROJECT_ROOT/infra/scripts/host-watchdog.sh" >> "$STATE_DIR/logs/host-agent.log" 2>&1 &
  echo "$!" > "$pidfile"
  sleep 1
  if kill -0 "$(cat "$pidfile" 2>/dev/null)" 2>/dev/null; then
    info "Host watchdog started (pid $(cat "$pidfile")) → real host metrics while the API is up."
  else
    warn "Host watchdog failed to start — monitoring will report runtime metrics."
  fi
}

stop_agent() {
  local pidfile="$STATE_DIR/host-agent.pid"
  if [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile" 2>/dev/null)" 2>/dev/null; then
    kill "$(cat "$pidfile")" >/dev/null 2>&1
    rm -f "$pidfile"
    info "Host watchdog stopped (host agent terminated)."
  fi
}

# ---------------------------------------------------------------------------
# LIFECYCLE
stop_api() {
  local changed=0
  if api_running; then
    $RUNTIME stop "$API_CONTAINER" >/dev/null 2>&1 && changed=1
  fi
  if $RUNTIME ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "$WEB_CONTAINER"; then
    $RUNTIME stop "$WEB_CONTAINER" >/dev/null 2>&1
  fi
  $RUNTIME stop "$WORKER_CONTAINER" "$MONITORING_CONTAINER" >/dev/null 2>&1 || true
  if [[ "$changed" -eq 1 ]]; then
    info "API stopped."
  else
    warn "API is not running."
  fi
}

status_cmd() {
  if api_running; then
    info "API: RUNNING (container $API_CONTAINER) → http://127.0.0.1:$API_PORT"
    curl -fsS "http://127.0.0.1:$API_PORT/api/v1/ready" >/dev/null 2>&1 \
      && info "  readiness: ok" \
      || warn "  readiness: failing"
  else
    warn "API: STOPPED"
  fi
  local apidfile="$STATE_DIR/host-agent.pid"
  if [[ -f "$apidfile" ]] && kill -0 "$(cat "$apidfile" 2>/dev/null)" 2>/dev/null; then
    info "Host agent: RUNNING (pid $(cat "$apidfile")) → real host metrics"
  else
    warn "Host agent: STOPPED — monitoring will report runtime metrics"
  fi
  if $RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -qx "$PG_CONTAINER"; then
    info "PostgreSQL: RUNNING (container $PG_CONTAINER)"
  else
    warn "PostgreSQL: STOPPED"
  fi
  if $RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -qx "$NATS_CONTAINER"; then
    info "NATS: RUNNING (container $NATS_CONTAINER)"
  else
    warn "NATS: STOPPED"
  fi
  if $RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -qx "$WORKER_CONTAINER"; then
    info "Worker: RUNNING (container $WORKER_CONTAINER)"
  else
    warn "Worker: STOPPED"
  fi
  if $RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -qx "$MONITORING_CONTAINER"; then
    info "Monitoring: RUNNING (container $MONITORING_CONTAINER)"
  else
    warn "Monitoring: STOPPED"
  fi
  if $RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -qx "$WEB_CONTAINER"; then
    info "Web: RUNNING (nginx container $WEB_CONTAINER) → http://127.0.0.1:$WEB_PORT"
  else
    warn "Web: STOPPED"
  fi
}

logs_cmd() {
  $RUNTIME logs -f --tail 100 "$API_CONTAINER"
}

banner() {
  cat <<'EOF'

      █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗
     ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗
     ███████║█████╗     ██║   ███████║█████╗  ██████╔╝
     ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗
     ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║
     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
              PaaS Platform — self-hosted installer
EOF
}

main() {
  banner
  progress_start
  PROGRESS_TOTAL=7
  export AETHER_PUBLIC_HOST="$(resolve_public_host)"
  export AETHER_PUBLIC_URL="$(resolve_public_url)"
  export AETHER_API_PUBLIC_URL="http://$AETHER_PUBLIC_HOST:$API_PORT"
  if [[ "$AETHER_PUBLIC_HOST" == "127.0.0.1" || "$AETHER_PUBLIC_HOST" == "localhost" ]]; then
    warn "Could not detect a routable host IP. Set AETHER_PUBLIC_HOST before installing for remote access."
  fi
  info "Aether — self-hosted installer (web port: $WEB_PORT, API: $API_PORT)"
  echo

  case "${1:-install}" in
    stop)
      ensure_runtime
      stop_agent
      stop_api
      return 0
      ;;
    status)
      ensure_runtime
      status_cmd
      return 0
      ;;
    logs)
      ensure_runtime
      logs_cmd
      return 0
      ;;
    start)
      ensure_runtime
      start_agent
      ensure_network
      ensure_ingress_image
      ensure_postgres
      ensure_nats
      ensure_registry
      ensure_builder
      ensure_web_image
      FORCE_API_RECREATE=1
      start_api
      start_workers
      start_web
      info "Aether is running."
      return 0
      ;;
    install|update)
      ;;
  esac

  progress_step 1 "Checking your system"
  ensure_runtime

  progress_step 2 "Securing application data"
  ensure_master_key

  progress_step 3 "Preparing application services"
  ensure_network
  ensure_ingress_image
  ensure_postgres
  ensure_nats

  progress_step 4 "Preparing the application build environment"
  ensure_registry
  ensure_builder

  progress_step 5 "Preparing the application"
  build_api_image
  ensure_web_image

  progress_step 6 "Starting Aether"
  start_agent
  FORCE_API_RECREATE=1
  start_api
  start_workers
  start_web

  progress_step 7 "Verifying the installation"
  status_cmd

  progress_finish
  info "Finalizing..."
  echo
  echo -e "${GREEN}┌──────────────────────────────────────────────────────────────┐${NC}"
  echo -e "${GREEN}│  ✓ Aether installed and running successfully!                    │${NC}"
  echo -e "${GREEN}└──────────────────────────────────────────────────────────────┘${NC}"
  echo
  echo -e "  API:      ${CYAN}$AETHER_API_PUBLIC_URL${NC}"
  echo -e "  Web:      ${CYAN}$AETHER_PUBLIC_URL${NC}"
  echo -e "  Logs:     ${CYAN}./install-dev.sh logs${NC}"
  echo
  echo -e "  Commands:"
  echo -e "    ${CYAN}./install-dev.sh start${NC}   — start everything (containers)"
  echo -e "    ${CYAN}./install-dev.sh stop${NC}    — stop the API"
  echo -e "    ${CYAN}./install-dev.sh status${NC}  — check the status"
  echo -e "    ${CYAN}./install-dev.sh logs${NC}    — follow the API logs"
  echo
  echo -e "  Aether is ready to use."
}

mkdir -p "$(dirname "$INSTALL_LOG")"
main "$@" 2>>"$INSTALL_LOG"
