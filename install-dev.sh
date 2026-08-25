#!/usr/bin/env bash

if [ -z "${BASH_VERSION:-}" ]; then
  exec bash "$0" "$@"
fi

set -euo pipefail

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
info()  { echo -e "${GREEN}[aether]${NC} $*" >&2; }
warn()  { echo -e "${YELLOW}[aether]${NC} $*" >&2; }
fail()  { echo -e "${RED}[aether]${NC} $*" >&2; exit 1; }

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${AETHER_ENV_FILE:-$PROJECT_ROOT/.env}"
if [[ -f "$ENV_FILE" ]]; then
  set -a
  . "$ENV_FILE"
  set +a
fi

# ---------------------------------------------------------------------------
# Configuration — everything runs in containers; podman is the ONLY host dep.
STATE_DIR="${AETHER_STATE:-$HOME/.aether}"
NET_NAME="${AETHER_NET:-aether-net}"

PG_CONTAINER="aether-postgres"
PG_IMAGE="${AETHER_PG_IMAGE:-docker.io/library/postgres:16-alpine}"
PG_PORT="${AETHER_PG_PORT:-${DATABASE_PORT:-15432}}"

NATS_CONTAINER="aether-nats"
NATS_IMAGE="${AETHER_NATS_IMAGE:-docker.io/library/nats:2.14.2-alpine}"
NATS_PORT="${AETHER_NATS_PORT:-4222}"
NATS_MONITOR_PORT="${AETHER_NATS_MONITOR_PORT:-8222}"
NATS_URL_EFFECTIVE="${AETHER_NATS_URL:-}"
NATS_USER="${AETHER_NATS_USER:-aether}"
NATS_PASSWORD="${AETHER_NATS_PASSWORD:-}"
NATS_AUTH_FILE="$STATE_DIR/keys/nats.auth"

API_CONTAINER="aether-api"
API_IMAGE="${AETHER_API_IMAGE:-aether.local/api:1}"
API_PORT="${AETHER_API_PORT:-8080}"
WORKER_CONTAINER="aether-worker"
MONITORING_CONTAINER="aether-monitoring"

WEB_CONTAINER="aether-web"
WEB_IMAGE="${AETHER_WEB_IMAGE:-aether.local/web:1}"
FRONTEND_DIR="$PROJECT_ROOT/frontend/web"
WEB_PORT=4000
DEV_MODE="${DEV_MODE:-false}"

MODE="${AETHER_MODE:-dev}"
CRED_FILE="$STATE_DIR/.aether-db"
HOST_LOG="$STATE_DIR/logs/host-setup.log"
FORCE_API_RECREATE=0

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
  printf '%s %s\n' "$(date -Is)" "$*" >> "$HOST_LOG"
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
# RUNTIME — podman (única dependência do host)
detect_runtime() {
  if command_exists podman; then
    echo "podman"
  else
    echo "none"
  fi
}

ensure_podman_machine() {
  command_exists podman || fail "podman is not available"
  if podman machine inspect --format '{{.State}}' >/dev/null 2>&1; then
    info "Podman machine already exists."
  else
    info "Creating Podman machine..."
    podman machine init || fail "podman machine init failed"
  fi
  local state
  state="$(podman machine inspect --format '{{.State}}' 2>/dev/null | tr '[:upper:]' '[:lower:]' || true)"
  if [[ "$state" != "running" ]]; then
    info "Starting Podman machine..."
    podman machine start || fail "podman machine start failed"
  fi
  info "Podman machine: $(podman machine inspect --format '{{.Name}} {{.State}}' 2>/dev/null || podman info --format '{{.Host.Arch}}')"

  # O buildah (via podman build) espera o profile de seccomp em
  # /etc/containers/seccomp.json. Em Fedora CoreOS ele fica em
  # /usr/share/containers/seccomp.json — cria o symlink na VM se faltar.
  if podman machine ssh -- "test -e /etc/containers/seccomp.json" 2>/dev/null | grep -q false; then
    podman machine ssh -- "sudo sh -c 'mkdir -p /etc/containers && ln -sf /usr/share/containers/seccomp.json /etc/containers/seccomp.json'" 2>/dev/null \
      || true
  fi

  # Permite o Traefik (rootless) expor as portas 80/443.
  podman machine ssh -- "sudo sh -c 'echo net.ipv4.ip_unprivileged_port_start=80 > /etc/sysctl.d/80-unpriv-ports.conf' && sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80" 2>/dev/null \
    || true
}

ensure_runtime() {
  local runtime
  runtime="$(detect_runtime)"
  if [[ "$runtime" == "none" ]]; then
    warn "podman is not installed — attempting to install it automatically..."
    if install_podman; then
      runtime="$(detect_runtime)"
    else
      fail "podman is required but could not be installed. Install it manually (https://podman.io/docs/installation) and run again."
    fi
  fi
  if [[ "$(uname -s)" == "Darwin" ]]; then
    ensure_podman_machine
  else
    ensure_linux_host
  fi
  RUNTIME="$runtime"
  info "Container runtime: podman"
}

# install_podman — instala o podman via gerenciador de pacotes da distro.
install_podman() {
  local pm=""
  if command_exists dnf; then
    pm="dnf install -y podman podman-docker"
  elif command_exists apt-get; then
    pm="apt-get update -qq && apt-get install -y podman"
  elif command_exists pacman; then
    pm="pacman -S --noconfirm --needed podman"
  elif command_exists zypper; then
    pm="zypper --non-interactive install podman"
  elif command_exists apk; then
    pm="apk add podman podman-compose"
  fi
  [[ -n "$pm" ]] || { host_log "fail: no package manager matched (dnf/apt/pacman/zypper/apk)"; return 1; }
  host_log "step: install podman -> $pm"
  local out
  out="$(run_sudo sh -c "$pm" 2>&1)"
  if [[ $? -eq 0 ]]; then
    host_log "ok: podman installed via $pm"
    info "  ✓ podman installed via package manager."
    return 0
  fi
  host_log "fail: install podman -> $pm ($out)"
  return 1
}

# ensure_linux_host — configura o host Linux nativo para o runtime rootless
# do podman. No macOS isso acontece dentro da VM (ensure_podman_machine);
# no Linux o host É a máquina, então fazemos os ajustes aqui, com sudo não
ensure_linux_host() {
  [[ "$(uname -s)" == "Linux" ]] || return 0
  host_log "=== host setup begin (user $(id -un), uid $(id -u)) ==="

  # 1) podman.socket — a API monta o socket para orquestrar deploys de apps.
  local sock
  sock="$(podman_socket)"
  if [[ -n "$sock" && -S "$sock" ]] && podman_socket_healthy "$sock"; then
    host_log "ok: podman.socket already present ($sock)"
  else
    host_log "step: enable podman.socket"
    if [[ "${AETHER_SKIP_SYSTEMD_SETUP:-false}" != "true" && "$(id -u)" -eq 0 && system_bus_available ]]; then
      if systemctl enable --now podman.socket >/dev/null 2>&1; then
        host_log "ok: podman.socket enabled (system)"
        info "  ✓ podman.socket enabled (system)"
      else
        host_log "fail: systemctl enable --now podman.socket"
        warn "  podman.socket could not be enabled — deploy orchestration will be limited."
      fi
    elif [[ "${AETHER_SKIP_SYSTEMD_SETUP:-false}" != "true" && "$(id -u)" -ne 0 ]]; then
      export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
      if user_bus_available && systemctl --user enable --now podman.socket >/dev/null 2>&1; then
        host_log "ok: podman.socket enabled (user)"
        info "  ✓ podman.socket enabled (user: $(id -un))"
      else
        host_log "fail: systemctl --user enable --now podman.socket"
        warn "  podman.socket could not be enabled — deploy orchestration will be limited."
      fi
    fi
    if [[ "$(uname -s)" == "Linux" ]]; then
      start_podman_socket_without_systemd
    fi
  fi

  # 2) Portas não-privilegiadas ≥ 80 — apps publicados na porta 80 (padrão)
  #    falham ao iniciar em rootless sem este sysctl (bind: permission denied).
  local cur
  cur="$(sysctl -n net.ipv4.ip_unprivileged_port_start 2>/dev/null || echo 1024)"
  if [[ "${cur:-1024}" -le 80 ]]; then
    host_log "ok: unprivileged ports start at $cur (<=80)"
  else
    host_log "step: set net.ipv4.ip_unprivileged_port_start=80 (currently $cur)"
    if can_sudo; then
      local out
      out="$(run_sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80 2>&1)"
      out+="$(run_sudo sh -c 'echo net.ipv4.ip_unprivileged_port_start=80 > /etc/sysctl.d/80-unpriv-ports.conf' 2>&1)"
      if [[ $? -eq 0 ]] && sysctl -n net.ipv4.ip_unprivileged_port_start | grep -qx '80'; then
        host_log "ok: unprivileged ports start at 80 (persisted)"
        info "  ✓ net.ipv4.ip_unprivileged_port_start=80 (persisted in /etc/sysctl.d)"
      else
        host_log "fail: set sysctl 80 -> $out"
        warn "  Could not set unprivileged ports to 80 — apps on port 80 will fail to start."
      fi
    else
      host_log "fail: sudo required for sysctl (current $cur)"
      warn "  Para apps na porta 80 (padrão) funcionarem, rode como root:"
      warn "    sudo sh -c 'echo net.ipv4.ip_unprivileged_port_start=80 > /etc/sysctl.d/80-unpriv-ports.conf && sysctl -w net.ipv4.ip_unprivileged_port_start=80'"
    fi
  fi

  # 3) Perfil seccomp do buildah — podman build falha sem
  #    /etc/containers/seccomp.json (em várias distros ele fica em
  #    /usr/share/containers/seccomp.json).
  if [[ -e /etc/containers/seccomp.json ]]; then
    host_log "ok: seccomp profile present"
  else
    local src=""
    for src in /usr/share/containers/seccomp.json /usr/local/share/containers/seccomp.json; do
      [[ -f "$src" ]] && break
      src=""
    done
    if [[ -n "$src" ]]; then
      if can_sudo; then
        local out
        out="$(run_sudo sh -c "mkdir -p /etc/containers && ln -sf '$src' /etc/containers/seccomp.json" 2>&1)"
        if [[ $? -eq 0 ]]; then
          host_log "ok: seccomp symlink $src -> /etc/containers/seccomp.json"
          info "  ✓ seccomp profile linked (/etc/containers/seccomp.json)"
        else
          host_log "fail: seccomp symlink -> $out"
          warn "  Could not create /etc/containers/seccomp.json — podman build may fail."
        fi
      else
        host_log "fail: sudo required for seccomp symlink"
        warn "  Para podman build funcionar, rode como root:"
        warn "    sudo ln -sf '$src' /etc/containers/seccomp.json"
      fi
    else
      host_log "warn: seccomp profile not found (/usr/share/containers, /usr/local/share/containers)"
      warn "  Perfil de seccomp não encontrado no host — podman build pode falhar."
    fi
  fi

  # 4) loginctl linger (rootless) — mantém o user manager vivo para que
  #    containers com restart=unless-stopped subam sozinhos após logout/reboot.
  if [[ "${AETHER_SKIP_SYSTEMD_SETUP:-false}" != "true" && "$(id -u)" -ne 0 ]] && command_exists loginctl && system_bus_available; then
    local linger
    linger="$(loginctl show-user "$(id -un)" -p Linger --value 2>/dev/null || true)"
    if [[ "$linger" == "yes" ]]; then
      host_log "ok: linger already enabled"
    elif can_sudo; then
      if run_sudo loginctl enable-linger "$(id -un)" >/dev/null 2>&1; then
        host_log "ok: linger enabled"
        info "  ✓ loginctl linger enabled (containers restauram após reboot)"
      else
        host_log "fail: loginctl enable-linger"
        warn "  loginctl enable-linger falhou — containers podem não subir sozinhos após reboot."
      fi
    else
      host_log "fail: sudo required for linger"
      warn "  Para containers subirem sozinhos após reboot, rode: sudo loginctl enable-linger $(id -un)"
    fi
  fi

  host_log "=== host setup end ==="
}

podman_socket_healthy() {
  local socket="$1"
  curl --unix-socket "$socket" -fsS http://d/_ping >/dev/null 2>&1 \
    || curl --unix-socket "$socket" -fsS http://d/libpod/_ping >/dev/null 2>&1
}

start_podman_socket_without_systemd() {
  local socket
  if [[ "$(id -u)" -eq 0 ]]; then
    socket="/run/podman/podman.sock"
    mkdir -p /run/podman
  else
    export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
    socket="$XDG_RUNTIME_DIR/podman/podman.sock"
    mkdir -p "$(dirname "$socket")"
  fi
  if podman_socket_healthy "$socket"; then
    export AETHER_PODMAN_SOCKET="$socket"
    return 0
  fi
  if [[ -S "$socket" ]]; then
    rm -f -- "$socket"
  fi
  info "  Starting Podman API service directly on $socket..."
  nohup podman system service --time=0 "unix://$socket" >/tmp/aether-podman-service.log 2>&1 &
  local attempt
  for attempt in {1..20}; do
    if podman_socket_healthy "$socket"; then
      export AETHER_PODMAN_SOCKET="$socket"
      host_log "ok: podman API service started directly ($socket)"
      info "  ✓ podman API service available ($socket)"
      return 0
    fi
    sleep 1
  done
  host_log "fail: podman API service did not become ready ($socket)"
  warn "  Podman API socket is unavailable — deploy orchestration will be limited."
  return 1
}

podman_socket() {
  if [[ -n "${AETHER_PODMAN_SOCKET:-}" && -S "$AETHER_PODMAN_SOCKET" ]]; then
    echo "$AETHER_PODMAN_SOCKET"
  elif [[ -S "/run/podman/podman.sock" ]]; then
    echo "/run/podman/podman.sock"
  elif [[ -n "${XDG_RUNTIME_DIR:-}" && -S "$XDG_RUNTIME_DIR/podman/podman.sock" ]]; then
    echo "$XDG_RUNTIME_DIR/podman/podman.sock"
  elif [[ -S "/run/user/$(id -u)/podman/podman.sock" ]]; then
    echo "/run/user/$(id -u)/podman/podman.sock"
  elif [[ "$(uname -s)" == "Darwin" ]]; then
    podman machine inspect --format '{{.ConnectionInfo.PodmanSocket.Path}}' 2>/dev/null || echo ""
  else
    echo ""
  fi
}

# podman_machine_socket — em podman machine (macOS), o daemon roda dentro da VM.
# O source de um `-v` é resolvido dentro da VM, então devolvemos o caminho do
podman_machine_socket() {
  local uri
  uri="$(podman system connection list --format '{{.Name}}|{{.URI}}|{{.Default}}' 2>/dev/null \
    | awk -F'|' '$3=="true"{print $2; exit}')"
  if [[ -n "$uri" && "$uri" == ssh://* ]]; then
    echo "$uri" | sed -E 's#^ssh://[^/]+(/.*)$#\1#'
  else
    echo ""
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
  info "Database credentials generated automatically (saved to $CRED_FILE, 0600)."
}

ensure_network() {
  if ! $RUNTIME network exists "$NET_NAME" 2>/dev/null; then
    info "Creating podman network '$NET_NAME'..."
    $RUNTIME network create "$NET_NAME" >/dev/null
  else
    info "Podman network '$NET_NAME' already exists."
  fi
}

ensure_postgres() {
  local runtime="$RUNTIME"
  load_db_credentials
  local password="$DB_PASSWORD"

  local exists
  exists="$($runtime ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$PG_CONTAINER" || true)"
  if [[ -n "$exists" ]]; then
    local published_port
    published_port="$($runtime port "$PG_CONTAINER" 5432/tcp 2>/dev/null | sed -nE 's/.*:([0-9]+)$/\1/p' | head -1)"
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
    $runtime exec "$PG_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "ALTER ROLE \"$DB_USER\" PASSWORD '$password';" >/dev/null 2>&1 \
      && info "Password synced in PostgreSQL." \
      || warn "Could not sync the password — check the container."
  else
    info "Creating PostgreSQL ($PG_IMAGE) with user '$DB_USER'..."
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
    info "PostgreSQL started on 127.0.0.1:$PG_PORT (database '$DB_NAME')."
  fi

  info "Waiting for PostgreSQL to become healthy..."
  local tries=0
  until $runtime exec "$PG_CONTAINER" pg_isready -U "$DB_USER" -d "$DB_NAME" >/dev/null 2>&1; do
    tries=$((tries + 1))
    [[ $tries -gt 60 ]] && fail "PostgreSQL did not become ready in 60s."
    sleep 2
  done
  info "PostgreSQL ready."
}

ensure_nats() {
	ensure_nats_auth
	if [[ -n "${AETHER_NATS_URL:-}" ]]; then
    info "NATS already configured at $AETHER_NATS_URL — reusing."
    NATS_URL_EFFECTIVE="$AETHER_NATS_URL"
    return 0
  fi
  local runtime="$RUNTIME"
  local exists
  exists="$($runtime ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$NATS_CONTAINER" || true)"
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
    info "NATS already exists ($NATS_CONTAINER) — using the existing one."
    local running
    running="$($runtime ps --format '{{.Names}}' 2>/dev/null | grep -cx "$NATS_CONTAINER" || true)"
    [[ "$running" -eq 0 ]] && $runtime start "$NATS_CONTAINER"
  else
    info "Creating NATS with JetStream ($NATS_IMAGE)..."
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
  info "NATS ready (container $NATS_CONTAINER)."
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
    info "Master key found at $keyfile — reusing (never overwrites)."
  else
    info "Generating master key (AES-256) at $keyfile..."
    mkdir -p "$keydir"
    umask 077
    if command_exists openssl; then
      openssl rand -hex 32 > "$keyfile"
    else
      head -c 32 /dev/urandom | xxd -p | tr -d '\n' > "$keyfile"
      printf '\n' >> "$keyfile"
    fi
    chmod 600 "$keyfile"
    info "Master key generated (0600). It is NEVER stored in the database."
  fi
}

# ---------------------------------------------------------------------------
# BUILD + RUN — API e frontend em containers
build_api_image() {
  [[ -f "$PROJECT_ROOT/infra/Dockerfile" ]] || fail "Dockerfile not found. Run from inside the project directory."
  [[ -f "$PROJECT_ROOT/api/cmd/api/main.go" ]] || fail "API source not found in ./api/cmd/api."
  info "Building the API image ($API_IMAGE)..."
  info "  (Go build + runtime — all inside a container)"
  $RUNTIME build -t "$API_IMAGE" -f "$PROJECT_ROOT/infra/Dockerfile" "$PROJECT_ROOT" \
    || fail "Image build failed."
  info "  ✓ image built: $API_IMAGE"
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
  info "Building the web image ($WEB_IMAGE)..."
  if ! $RUNTIME build -t "$WEB_IMAGE" -f "$PROJECT_ROOT/infra/web.Dockerfile" "$PROJECT_ROOT"; then
    cleanup_web_build_env
    fail "Web image build failed."
  fi
  cleanup_web_build_env
  info "  ✓ image built: $WEB_IMAGE"
}

api_exists() {
  $RUNTIME ps -a --format '{{.Names}}' 2>/dev/null | grep -qx "$API_CONTAINER"
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

  local socket
  socket="$(podman_socket || true)"

  mkdir -p "$STATE_DIR" "$STATE_DIR/data" "$STATE_DIR/certs" "$STATE_DIR/logs" \
    "$STATE_DIR/builds" "$STATE_DIR/cache" "$STATE_DIR/keys" "$STATE_DIR/logs/apps" \
    "$STATE_DIR/builds/sources" "$STATE_DIR/snapshots"

  # volume compartilhado para o ingress (config Traefik + acme), montado na API
  # e no container Traefik, evitando dependência de path do host.
  $RUNTIME volume create aether-traefik >/dev/null 2>&1 || true
  $RUNTIME volume create aether-pack-cache >/dev/null 2>&1 || true

  if api_exists; then
    $RUNTIME rm -f "$API_CONTAINER" >/dev/null 2>&1
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
    -e "AETHER_COOKIE_SECURE=${AETHER_COOKIE_SECURE:-false}"
    -e "AETHER_MODE=$MODE"
  )
  # Monta o socket do podman para a API orquestrar deploys de apps.
  # - Linux: socket unix do host (mountável diretamente).
  # - macOS (podman machine): o daemon roda na VM; o source do `-v` é resolvido
  #   dentro da VM, então usamos o caminho do socket rootless da VM.
  local sock_mount=0
  local sock_src=""
  if [[ -n "$socket" && -S "$socket" && ( "$socket" == "$HOME"/* || "$(uname -s)" == "Linux" ) ]]; then
    sock_src="$socket"
  elif [[ "$(uname -s)" == "Darwin" ]]; then
    sock_src="$(podman_machine_socket)"
  fi
  if [[ -n "$sock_src" ]]; then
    # Monta o socket no MESMO path do host/VM para que o podman dentro do
    # container consiga re-exportar o socket em containers filhos (builds
    # pack/buildpacks), já que o source de um -v é resolvido
    # dentro da VM, não no container.
    args+=( -v "$sock_src:$sock_src:ro" )
    args+=( -e "CONTAINER_HOST=unix://$sock_src" -e "DOCKER_HOST=unix://$sock_src" )
    sock_mount=1
    info "  ✓ podman socket mounted (for app deployments): $sock_src"
  fi
  if [[ "$sock_mount" -eq 0 ]]; then
    warn "  podman socket not mountable on this host — app deployment orchestration will be limited (core platform is unaffected)."
  fi

  if ! $RUNTIME "${args[@]}" "$API_IMAGE" >/dev/null 2>&1; then
    if [[ "$sock_mount" -eq 1 ]]; then
      warn "  Could not mount the podman socket — retrying without it."
      if api_exists; then
        $RUNTIME rm -f "$API_CONTAINER" >/dev/null 2>&1
      fi
      local clean_args=()
      for a in "${args[@]}"; do
        [[ "$a" == *"podman.sock:ro" ]] && continue
        [[ "$a" == "CONTAINER_HOST=unix://"* ]] && continue
        [[ "$a" == "DOCKER_HOST=unix://"* ]] && continue
        clean_args+=( "$a" )
      done
      $RUNTIME "${clean_args[@]}" "$API_IMAGE" >/dev/null || fail "Failed to start the API container."
    else
      fail "Failed to start the API container."
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
  local socket
  socket="$(podman_socket || true)"
  local sock_src=""
  if [[ -n "$socket" && -S "$socket" && ( "$socket" == "$HOME"/* || "$(uname -s)" == "Linux" ) ]]; then
    sock_src="$socket"
  elif [[ "$(uname -s)" == "Darwin" ]]; then
    sock_src="$(podman_machine_socket)"
  fi
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
    args+=( -e "AETHER_WORKER_HEALTH_ADDR=0.0.0.0:8081" -p "127.0.0.1:8081:8081" )
  elif [[ "$binary" == "aether-monitoring" ]]; then
    args+=( -e "AETHER_MONITORING_HEALTH_ADDR=0.0.0.0:8082" -p "127.0.0.1:8082:8082" )
  fi
  if [[ -n "$sock_src" ]]; then
    args+=( -v "$sock_src:$sock_src:ro" )
    args+=( -e "CONTAINER_HOST=unix://$sock_src" -e "DOCKER_HOST=unix://$sock_src" )
  fi
  info "Starting $container..."
  $RUNTIME "${args[@]}" "$API_IMAGE" >/dev/null || fail "Failed to start $container."
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
  info "nginx config generated: $conf"
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
# REGISTRY + CNB BUILDER — registry local (aether-registry:5000) e a imagem
# do builder CNB (Paketo node + aether/spa-static), construída via podman
# build (pack builder create não exporta corretamente para o podman).
REGISTRY_IMAGE="${AETHER_REGISTRY_IMAGE:-docker.io/library/registry:2}"
REGISTRY_ADDR="${AETHER_REGISTRY_ADDR:-127.0.0.1:5000}"
REGISTRY_CONTAINER="aether-registry"
CNB_BUILDER="${AETHER_CNB_BUILDER:-${AETHER_REGISTRY_ADDR:-127.0.0.1:5000}/builder:node-spa}"
if [[ "$CNB_BUILDER" == "aether/builder:node-spa" || "$CNB_BUILDER" == "localhost/aether/builder:node-spa" ]]; then
  CNB_BUILDER="${AETHER_REGISTRY_ADDR:-127.0.0.1:5000}/builder:node-spa"
fi

ensure_insecure_registry() {
  local conf="$HOME/.config/containers/registries.conf"
  if [[ -f "$conf" ]] && grep -q "127.0.0.1:5000" "$conf"; then
    return 0
  fi
  mkdir -p "$(dirname "$conf")"
  {
    [[ -f "$conf" ]] && cat "$conf"
    printf '\n[[registry]]\nlocation = "127.0.0.1:5000"\ninsecure = true\n\n[[registry]]\nlocation = "localhost:5000"\ninsecure = true\n'
  } > "$conf.tmp" && mv "$conf.tmp" "$conf"
  info "registries.conf: registry local (127.0.0.1:5000) marcado como insecure."
}

ensure_registry() {
  ensure_insecure_registry
  local exists
  exists="$($RUNTIME ps -a --format '{{.Names}}' 2>/dev/null | grep -x "$REGISTRY_CONTAINER" || true)"
  if [[ -n "$exists" ]]; then
    local running
    running="$($RUNTIME ps --format '{{.Names}}' 2>/dev/null | grep -cx "$REGISTRY_CONTAINER" || true)"
    [[ "$running" -eq 0 ]] && $RUNTIME start "$REGISTRY_CONTAINER" >/dev/null
    info "Registry already exists ($REGISTRY_CONTAINER) — using the existing one."
  else
    info "Creating registry ($REGISTRY_IMAGE) on $REGISTRY_ADDR..."
    $RUNTIME run -d \
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
  $RUNTIME pull "docker.io/buildpacksio/lifecycle:${AETHER_LIFECYCLE_VERSION:-0.21.17}" >/dev/null 2>&1 || true
  $RUNTIME pull "${AETHER_CNB_RUN_IMAGE:-docker.io/library/ubuntu:24.04}" >/dev/null 2>&1 || true
  local source_stamp builder_stamp
  source_stamp="$(sha256sum "$PROJECT_ROOT/infra/buildpacks/node-server/bin/build" "$PROJECT_ROOT/infra/buildpacks/node-server/bin/detect" "$PROJECT_ROOT/infra/buildpacks/spa-static/bin/build" "$PROJECT_ROOT/infra/buildpacks/spa-static/bin/detect" "$PROJECT_ROOT/infra/buildpacks/builders/build-builder.sh" | sha256sum | awk '{print $1}')"
  builder_stamp="$STATE_DIR/cnb-builder.stamp"
  if $RUNTIME image exists "$CNB_BUILDER" >/dev/null 2>&1 && [[ -f "$builder_stamp" ]] && grep -qx "$source_stamp" "$builder_stamp"; then
    info "CNB builder already present ($CNB_BUILDER)."
    return 0
  fi
  info "Building CNB builder ($CNB_BUILDER) — aether buildpacks + ubuntu run image..."
  bash "$PROJECT_ROOT/infra/buildpacks/builders/build-builder.sh" || fail "CNB builder build failed."
  $RUNTIME image exists "$CNB_BUILDER" >/dev/null 2>&1 || fail "CNB builder image is not available as $CNB_BUILDER."
  mkdir -p "$STATE_DIR"
  printf '%s\n' "$source_stamp" > "$builder_stamp"
  info "CNB builder ready."
}

# ---------------------------------------------------------------------------
# HOST AGENT (host machine — macOS or Linux)
# The API runs inside a container; the host agent runs natively on the host
# machine and writes the real host metrics to $STATE_DIR/host-stats.json
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

  info "1/6 Checking the system..."
  ensure_runtime

  info "2/6 Preparing directories and keys..."
  ensure_master_key

  info "3/6 Setting up containers (network, postgres, NATS)..."
  ensure_network
  ensure_postgres
  ensure_nats

  info "3.5/6 Setting up registry + CNB builder..."
  ensure_registry
  ensure_builder

  info "4/6 Building the API and frontend images..."
  build_api_image
  ensure_web_image

  info "5/6 Starting the host agent + API container..."
  start_agent
  FORCE_API_RECREATE=1
  start_api
  start_workers
  start_web

  info "6/6 Verifying the installation..."
  status_cmd

  info "Finalizing..."
  echo
  echo -e "${GREEN}┌──────────────────────────────────────────────────────────────┐${NC}"
  echo -e "${GREEN}│  ✓ Aether installed and running successfully!                    │${NC}"
  echo -e "${GREEN}└──────────────────────────────────────────────────────────────┘${NC}"
  echo
  echo -e "  API:      ${CYAN}$AETHER_API_PUBLIC_URL${NC}"
  echo -e "  Web:      ${CYAN}$AETHER_PUBLIC_URL${NC} (nginx container)"
  echo -e "  State:    $STATE_DIR"
  echo -e "  Logs:     ${CYAN}./install-dev.sh logs${NC}"
  echo -e "  Host log: $HOST_LOG (instalação/configuração do host)"
  echo
  echo -e "  Commands:"
  echo -e "    ${CYAN}./install-dev.sh start${NC}   — start everything (containers)"
  echo -e "    ${CYAN}./install-dev.sh stop${NC}    — stop the API"
  echo -e "    ${CYAN}./install-dev.sh status${NC}  — check the status"
  echo -e "    ${CYAN}./install-dev.sh logs${NC}    — follow the API logs"
  echo
  echo -e "  Everything (API, frontend, postgres, NATS) runs in podman containers."
  echo -e "  podman is the only tool required on this host."
}

main "$@"
