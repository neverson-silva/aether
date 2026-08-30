#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
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
    fail "Root privileges are required to purge Docker packages."
  fi
}

if [[ "$(uname -s)" != "Linux" ]]; then
  fail "The production uninstaller supports Linux servers only."
fi
command_exists docker || fail "Docker CLI is not installed."

STATE_DIR="${AETHER_STATE:-$(if [[ "$(id -u)" -eq 0 ]]; then printf '/var/lib/aether'; else printf '%s/.aether' "$HOME"; fi)}"
INSTALL_DIR="${AETHER_INSTALL_DIR:-$(if [[ "$(id -u)" -eq 0 ]]; then printf '/opt/aether'; else printf '%s/.local/share/aether' "$HOME"; fi)}"
YES=0
PURGE_RUNTIME=0

for argument in "$@"; do
  case "$argument" in
    --yes) YES=1 ;;
    --purge-runtime) PURGE_RUNTIME=1 ;;
    *) fail "Usage: uninstall.sh [--yes] [--purge-runtime]" ;;
  esac
done

if [[ "$YES" -ne 1 ]]; then
  printf 'This removes Aether containers, volumes, images, state, and checkout. Type REMOVE to continue: '
  read -r confirmation
  [[ "$confirmation" == "REMOVE" ]] || fail "Uninstall cancelled."
fi

remove_container() {
  local container="$1"
  if docker container inspect "$container" >/dev/null 2>&1; then
    info "Removing container $container."
    docker rm -f "$container" >/dev/null
  fi
}

remove_volume() {
  local volume="$1"
  if docker volume inspect "$volume" >/dev/null 2>&1; then
    info "Removing volume $volume."
    docker volume rm -f "$volume" >/dev/null
  fi
}

remove_image() {
  local image="$1"
  if docker image inspect "$image" >/dev/null 2>&1; then
    info "Removing image $image."
    docker image rm "$image" >/dev/null || warn "Could not remove image $image."
  fi
}

for container in aether-api aether-worker aether-monitoring aether-web aether-postgres aether-nats aether-registry aether-traefik; do
  remove_container "$container"
done

for volume in aether-pg-data aether-nats-data aether-traefik aether-pack-cache; do
  remove_volume "$volume"
done

for image in aether.local/api:1 aether.local/web:1 aether.local/builder:node-spa 127.0.0.1:1500/builder:node-spa localhost:1500/builder:node-spa; do
  remove_image "$image"
done

for network in aether-net aether-ingress aether-internal; do
  docker network inspect "$network" >/dev/null 2>&1 && docker network rm "$network" >/dev/null 2>&1 || true
done

if [[ -d "$STATE_DIR" && "$STATE_DIR" != "/" && "$STATE_DIR" != "$HOME" ]]; then
  info "Removing Aether state from $STATE_DIR."
  rm -rf -- "$STATE_DIR"
fi

if [[ -d "$INSTALL_DIR/.git" && "$INSTALL_DIR" != "/" && "$INSTALL_DIR" != "$HOME" ]]; then
  info "Removing Aether checkout from $INSTALL_DIR."
  rm -rf -- "$INSTALL_DIR"
fi

if [[ "$PURGE_RUNTIME" -eq 1 ]]; then
  if command_exists apt-get; then
    run_root apt-get purge -y docker.io docker-ce docker-ce-cli docker-compose-plugin
  elif command_exists dnf; then
    run_root dnf remove -y docker docker-ce docker-ce-cli docker-compose-plugin
  elif command_exists yum; then
    run_root yum remove -y docker docker-ce docker-ce-cli docker-compose-plugin
  elif command_exists pacman; then
    run_root pacman -Rns --noconfirm docker docker-compose
  elif command_exists zypper; then
    run_root zypper --non-interactive remove docker docker-compose
  elif command_exists apk; then
    run_root apk del docker docker-cli-compose
  else
    warn "Could not identify the package manager; Docker packages were not removed."
  fi
fi

info "Aether uninstall completed."
