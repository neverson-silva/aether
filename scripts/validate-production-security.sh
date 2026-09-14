#!/usr/bin/env bash
set -euo pipefail

state_dir="${AETHER_STATE_DIR:-${HOME}/.aether}"
ingress_network="${AETHER_INGRESS_NETWORK:-aether-ingress}"
compose_file="${AETHER_COMPOSE_FILE:-infra/docker-compose.yml}"
failed=0

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'required command is unavailable: %s\n' "$1" >&2
    failed=1
  fi
}

require_command docker
require_command find

if command -v docker >/dev/null 2>&1; then
  if ! docker info >/dev/null 2>&1; then
    printf '%s\n' 'Docker Desktop daemon is unavailable' >&2
    failed=1
  else
    rootless="$(docker info --format '{{.SecurityOptions}}' 2>/dev/null || true)"
    printf 'Docker security options: %s\n' "$rootless"
    if docker inspect aether-api >/dev/null 2>&1; then
      api_state="$(docker inspect --format '{{.State.Status}}' aether-api)"
      if [[ "$api_state" == "running" ]]; then
        api_networks="$(docker inspect --format '{{json .NetworkSettings.Networks}}' aether-api)"
        if [[ "$api_networks" == *"$ingress_network"* ]]; then
          printf 'control plane must not attach to tenant ingress network: %s\n' "$ingress_network" >&2
          failed=1
        fi
      fi
    fi
    if docker network inspect "$ingress_network" >/dev/null 2>&1; then
      internal="$(docker network inspect --format '{{.Internal}}' "$ingress_network")"
      if [[ "$internal" != "true" ]]; then
        printf 'tenant ingress network is not internal: %s\n' "$ingress_network" >&2
        failed=1
      fi
    else
      printf 'tenant ingress network is unavailable: %s\n' "$ingress_network" >&2
      failed=1
    fi
    for container in aether-api aether-worker aether-monitoring aether-web; do
      if docker inspect "$container" >/dev/null 2>&1; then
        published="$(docker inspect --format '{{json .HostConfig.PortBindings}}' "$container")"
        if [[ "$published" == *'0.0.0.0'* || "$published" == *'::'* ]]; then
          printf 'container publishes a non-loopback port: %s\n' "$container" >&2
          failed=1
        fi
      fi
    done
  fi
fi

if [[ -f "$compose_file" ]] && command -v docker >/dev/null 2>&1; then
  if ! AETHER_NATS_PASSWORD=validation-nats-password AETHER_NATS_PASSWORD_HASH=validation-nats-password-hash AETHER_NATS_USER=validation-nats-user DATABASE_PASSWORD=validation-database-password docker compose -f "$compose_file" config --quiet; then
    printf 'Compose configuration is invalid: %s\n' "$compose_file" >&2
    failed=1
  fi
fi

if [[ -d "$state_dir/traefik" ]]; then
  while IFS= read -r -d '' item; do
    if [[ -L "$item" ]]; then
      printf 'Traefik state contains a symlink: %s\n' "$item" >&2
      failed=1
    fi
  done < <(find "$state_dir/traefik" -xdev -print0)
fi

if (( failed != 0 )); then
  exit 1
fi

printf '%s\n' 'Production security validation passed'
