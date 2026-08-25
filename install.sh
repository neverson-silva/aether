#!/usr/bin/env bash
set -euo pipefail

readonly REPOSITORY_URL="${AETHER_REPOSITORY_URL:-https://github.com/neverson-silva/aether.git}"
readonly REPOSITORY_REF="${AETHER_REPOSITORY_REF:-main}"
readonly INSTALL_DIR="${AETHER_INSTALL_DIR:-$(if [[ "$(id -u)" -eq 0 ]]; then printf '/opt/aether'; else printf '%s/.local/share/aether' "$HOME"; fi)}"
readonly DEFAULT_STATE_DIR="${AETHER_STATE:-$(if [[ "$(id -u)" -eq 0 ]]; then printf '/var/lib/aether'; else printf '%s/.aether' "$HOME"; fi)}"
readonly ENV_FILE="${AETHER_ENV_FILE:-$DEFAULT_STATE_DIR/.env}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

info() { echo -e "${GREEN}[aether]${NC} $*" >&2; }
warn() { echo -e "${YELLOW}[aether]${NC} $*" >&2; }
fail() { echo -e "${RED}[aether]${NC} $*" >&2; exit 1; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

run_root() {
  if [[ "$(id -u)" -eq 0 ]]; then
    "$@"
  elif command_exists sudo; then
    sudo "$@"
  else
    fail "Root privileges are required to install system dependencies. Install sudo or run this script as root."
  fi
}

distribution() {
  if [[ -r /etc/os-release ]]; then
    . /etc/os-release
    printf '%s' "${ID:-linux}"
    return
  fi
  printf 'linux'
}

install_bootstrap_dependencies() {
  local distro
  distro="$(distribution)"
  if command_exists git && command_exists curl; then
    return 0
  fi

  info "Installing bootstrap dependencies for ${distro}."
  case "$distro" in
    ubuntu|debian|linuxmint|pop)
      run_root apt-get update
      run_root apt-get install -y git curl ca-certificates
      ;;
    fedora|rhel|centos|rocky|almalinux|ol|amzn)
      if command_exists dnf; then
        run_root dnf install -y git curl ca-certificates
      elif command_exists yum; then
        run_root yum install -y git curl ca-certificates
      elif command_exists microdnf; then
        run_root microdnf install -y git curl ca-certificates
      else
        fail "No supported RPM package manager was found on ${distro}."
      fi
      ;;
    opensuse*|sles)
      run_root zypper --non-interactive install git curl ca-certificates
      ;;
    arch|manjaro)
      run_root pacman -Sy --noconfirm --needed git curl ca-certificates
      ;;
    alpine)
      run_root apk add --no-cache bash git curl ca-certificates
      ;;
    *)
      fail "Unsupported distribution '${distro}'. Install git and curl, then run this script again."
      ;;
  esac
}

configure_runtime_socket() {
  if [[ -n "${AETHER_PODMAN_SOCKET:-}" && -S "$AETHER_PODMAN_SOCKET" ]]; then
    export AETHER_PODMAN_SOCKET
    return 0
  fi
  if [[ -S /run/podman/podman.sock ]]; then
    export AETHER_PODMAN_SOCKET=/run/podman/podman.sock
  elif [[ -S "/run/user/$(id -u)/podman/podman.sock" ]]; then
    export AETHER_PODMAN_SOCKET="/run/user/$(id -u)/podman/podman.sock"
  elif [[ -n "${XDG_RUNTIME_DIR:-}" && -S "$XDG_RUNTIME_DIR/podman/podman.sock" ]]; then
    export AETHER_PODMAN_SOCKET="$XDG_RUNTIME_DIR/podman/podman.sock"
  else
    unset AETHER_PODMAN_SOCKET
    warn "Podman socket was not found yet; install-dev.sh will retry its runtime detection."
  fi
}

validate_host() {
  [[ "$(uname -s)" == "Linux" ]] || fail "The production installer supports Linux servers only. Use install-dev.sh on macOS."
  case "$(uname -m)" in
    x86_64|amd64|aarch64|arm64)
      ;;
    *)
      fail "Unsupported architecture '$(uname -m)'. Supported architectures: amd64 and arm64."
      ;;
  esac
}

prepare_install_dir() {
  if [[ -e "$INSTALL_DIR" && ! -d "$INSTALL_DIR" ]]; then
    fail "Install path exists and is not a directory: $INSTALL_DIR"
  fi
  mkdir -p "$(dirname "$INSTALL_DIR")"
  if [[ -d "$INSTALL_DIR/.git" ]]; then
    local remote
    remote="$(git -C "$INSTALL_DIR" remote get-url origin 2>/dev/null || true)"
    [[ "$remote" == "$REPOSITORY_URL" || "$remote" == "${REPOSITORY_URL%.git}" ]] || fail "Existing directory has a different Git remote: $remote"
    [[ -z "$(git -C "$INSTALL_DIR" status --porcelain)" ]] || fail "Existing Aether checkout has local changes: $INSTALL_DIR"
    info "Updating Aether checkout in $INSTALL_DIR."
    git -C "$INSTALL_DIR" fetch --depth 1 origin "$REPOSITORY_REF"
    git -C "$INSTALL_DIR" checkout -B "$REPOSITORY_REF" "origin/$REPOSITORY_REF"
  elif [[ -n "$(find "$INSTALL_DIR" -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    fail "Install directory is not an Aether Git checkout and is not empty: $INSTALL_DIR"
  else
    info "Cloning Aether from $REPOSITORY_URL into $INSTALL_DIR."
    git clone --depth 1 --branch "$REPOSITORY_REF" "$REPOSITORY_URL" "$INSTALL_DIR"
  fi
}

preserve_install_configuration() {
  local checkout_env="$INSTALL_DIR/.env"
  if [[ "$ENV_FILE" != "$checkout_env" && -f "$checkout_env" && ! -e "$ENV_FILE" ]]; then
    mkdir -p "$(dirname "$ENV_FILE")"
    cp -p "$checkout_env" "$ENV_FILE"
    chmod 600 "$ENV_FILE"
    info "Preserved installation configuration in $ENV_FILE."
  fi
}

validate_checkout() {
  [[ -x "$INSTALL_DIR/install-dev.sh" ]] || fail "The cloned repository does not contain executable install-dev.sh."
  [[ -f "$INSTALL_DIR/infra/Dockerfile" ]] || fail "The cloned repository is missing infra/Dockerfile."
  [[ -f "$INSTALL_DIR/frontend/aether_ds/package.json" ]] || fail "The cloned repository is missing frontend/aether_ds/package.json required by the web image build."
  chmod +x "$INSTALL_DIR/install-dev.sh"
}

run_development_installer() {
  export AETHER_MODE="${AETHER_MODE:-prod}"
  export DEV_MODE="${DEV_MODE:-false}"
  if [[ ! -S /run/dbus/system_bus_socket && ! -S /run/systemd/private ]]; then
    export AETHER_SKIP_SYSTEMD_SETUP=true
  fi
  export AETHER_PG_PORT="${AETHER_PG_PORT:-${DATABASE_PORT:-15432}}"
  if [[ -z "${AETHER_STATE:-}" ]]; then
    if [[ "$(id -u)" -eq 0 ]]; then
      export AETHER_STATE="$DEFAULT_STATE_DIR"
    else
      export AETHER_STATE="$HOME/.aether"
    fi
  fi
  export AETHER_ENV_FILE="$ENV_FILE"
  configure_runtime_socket
  cd "$INSTALL_DIR"
  "$INSTALL_DIR/install-dev.sh" "${1:-install}"
}

cleanup_install_dir() {
  [[ -n "$INSTALL_DIR" && "$INSTALL_DIR" != "/" && "$INSTALL_DIR" != "$HOME" ]] || fail "Refusing to remove an unsafe install path: $INSTALL_DIR"
  [[ -d "$INSTALL_DIR/.git" ]] || return 0
  if [[ "$ENV_FILE" == "$INSTALL_DIR/"* && -f "$ENV_FILE" && ! -e "$DEFAULT_STATE_DIR/.env" ]]; then
    mkdir -p "$DEFAULT_STATE_DIR"
    cp -p "$ENV_FILE" "$DEFAULT_STATE_DIR/.env"
    chmod 600 "$DEFAULT_STATE_DIR/.env"
    info "Moved installation configuration to $DEFAULT_STATE_DIR/.env."
  fi
  info "Removing the temporary Aether checkout from $INSTALL_DIR."
  rm -rf -- "$INSTALL_DIR"
}

main() {
  local command="${1:-install}"
  case "$command" in
    install|update|start|stop|status|logs)
      ;;
    *)
      fail "Usage: install.sh [install|update|start|stop|status|logs]"
      ;;
  esac

  validate_host
  install_bootstrap_dependencies
  configure_runtime_socket
  prepare_install_dir
  validate_checkout
  preserve_install_configuration

  if [[ "$command" == "update" ]]; then
    command="install"
  fi
  info "Running Aether production installer from $INSTALL_DIR."
  info "State is kept outside the checkout at $DEFAULT_STATE_DIR."
  run_development_installer "$command"
  if [[ "$command" == "install" || "$command" == "start" ]]; then
    cleanup_install_dir
  fi
}

main "$@"
