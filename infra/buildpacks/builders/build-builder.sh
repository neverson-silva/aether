#!/usr/bin/env bash
# ============================================================================
# build-builder.sh — monta o builder CNB do Aether (todos os buildpacks)
#
# Estratégia: `pack builder create` não exporta corretamente para o podman
# ("duplicate paths"), então o builder é montado via `podman build` com a
# estrutura CNB canônica (/cnb/lifecycle, /cnb/buildpacks, order.toml,
# stack.toml + labels) e publicado no registry local `127.0.0.1:5000`.
#
# Uso:  ./build-builder.sh [arm64|x64]
# Depois: pack build my-app -B 127.0.0.1:5000/builder:node-spa --docker-host=inherit
# ============================================================================
set -euo pipefail
cd "$(dirname "$0")/../.."
ROOT="$(pwd)"

ARCH="${1:-$(uname -m)}"
case "$ARCH" in
  x86_64|amd64) LIFE_ARCH="x86-64"; REG_ARCH="" ;;
  aarch64|arm64) LIFE_ARCH="arm64"; REG_ARCH="" ;;
  *) echo "arch não suportada: $ARCH"; exit 1 ;;
esac

REGISTRY="${AETHER_REGISTRY_ADDR:-127.0.0.1:5000}"
BUILDER_TAG="$REGISTRY/builder:node-spa"
LIFECYCLE_VER="${LIFECYCLE_VER:-0.21.17}"
RUN_IMAGE="${RUN_IMAGE:-docker.io/library/ubuntu:24.04}"
STACK_ID="io.buildpacks.stacks.aether"
HOST_DISTRO="$(if [[ -r /etc/os-release ]]; then . /etc/os-release; printf '%s' "${ID:-linux}"; else printf 'linux'; fi)"
case "${AETHER_BUILDER_DISTRO:-$HOST_DISTRO}" in
  fedora|rhel|centos|rocky|almalinux|ol|amzn) BUILDER_BASE_IMAGE="${AETHER_BUILDER_BASE_IMAGE:-quay.io/fedora/fedora:latest}" ;;
  alpine) BUILDER_BASE_IMAGE="${AETHER_BUILDER_BASE_IMAGE:-docker.io/library/alpine:3.21}" ;;
  arch|manjaro) BUILDER_BASE_IMAGE="${AETHER_BUILDER_BASE_IMAGE:-docker.io/library/archlinux:base}" ;;
  opensuse*|sles) BUILDER_BASE_IMAGE="${AETHER_BUILDER_BASE_IMAGE:-registry.opensuse.org/opensuse/leap:15.6}" ;;
  *) BUILDER_BASE_IMAGE="${AETHER_BUILDER_BASE_IMAGE:-docker.io/library/ubuntu:24.04}" ;;
esac

# Ordem de detecção (marcadores fortes primeiro; node-server rejeita SPA; spa pega o resto)
ORDER_BPS=(php-server ruby-server dotnet-server go-server rust-server jvm-server node-server spa-static)

info() { printf '\033[0;32m[builder]\033[0m %s\n' "$*"; }

command -v podman >/dev/null || { echo "podman é necessário"; exit 1; }
info "host distro: $HOST_DISTRO; builder base: $BUILDER_BASE_IMAGE"

# ---------------------------------------------------------------------------
# 1. Registry local (podman machine: host network = VM localhost, visível aos
#    lifecycle containers via --docker-host=inherit)
# ---------------------------------------------------------------------------
if ! podman ps --format '{{.Names}}' | grep -qx aether-registry; then
  info "subindo aether-registry (host network, :5000)..."
  if podman ps -a --format '{{.Names}}' | grep -qx aether-registry; then
    podman rm -f aether-registry >/dev/null
  fi
  podman run -d --name aether-registry --network host docker.io/library/registry:2 >/dev/null
  sleep 3
fi
info "registry em http://127.0.0.1:5000/v2/ -> $(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:5000/v2/ || echo falha)"

# ---------------------------------------------------------------------------
# 2. Contexto de build
# ---------------------------------------------------------------------------
CTX="$(mktemp -d)"
trap 'rm -rf "$CTX"' EXIT
mkdir -p "$CTX/cnb/lifecycle" "$CTX/cnb/buildpacks"

# 2.1 lifecycle (CNB)
LIF_VER_FULL="${LIFECYCLE_VER#v}"
info "baixando lifecycle $LIF_VER_FULL ($LIFE_ARCH)..."
curl -fsSLo /tmp/lifecycle.tgz "https://github.com/buildpacks/lifecycle/releases/download/v$LIF_VER_FULL/lifecycle-v$LIF_VER_FULL+linux.$LIFE_ARCH.tgz"
tar -xzf /tmp/lifecycle.tgz -C "$CTX/cnb"
rm -f /tmp/lifecycle.tgz
chmod +x "$CTX/cnb/lifecycle"/*

# 2.2 buildpacks (aether/*)
BP_META_JSON="["
for i in "${!ORDER_BPS[@]}"; do
  bp="${ORDER_BPS[$i]}"
  ver="$(grep -oE 'version = "[^"]+"' "$ROOT/buildpacks/$bp/buildpack.toml" | head -1 | sed -E 's/.*"([^"]+)"/\1/')"
  mkdir -p "$CTX/cnb/buildpacks/aether_$bp/$ver"
  cp -a "$ROOT/buildpacks/$bp/." "$CTX/cnb/buildpacks/aether_$bp/$ver/"
  chmod +x "$CTX/cnb/buildpacks/aether_$bp/$ver/bin"/*
  [ "$i" -eq 0 ] || BP_META_JSON="$BP_META_JSON,"
  BP_META_JSON="$BP_META_JSON{\"id\":\"aether/$bp\",\"version\":\"$ver\",\"homepage\":\"https://github.com/aether\"}"
  info "buildpack aether/$bp@$ver"
done
BP_META_JSON="$BP_META_JSON]"

# 2.3 order.toml
: > "$CTX/cnb/order.toml"
for bp in "${ORDER_BPS[@]}"; do
  ver="$(grep -oE 'version = "[^"]+"' "$ROOT/buildpacks/$bp/buildpack.toml" | head -1 | sed -E 's/.*"([^"]+)"/\1/')"
  cat >> "$CTX/cnb/order.toml" <<EOF
[[order]]

  [[order.group]]
  id = "aether/$bp"
  version = "$ver"

EOF
done

# 2.4 stack.toml
cat > "$CTX/cnb/stack.toml" <<EOF
[run]
image = "$RUN_IMAGE"

[build]
image = "$RUN_IMAGE"
EOF

# 2.5 labels do builder
BUILDER_META="{\"description\":\"Aether CNB builder\",\"buildpacks\":$BP_META_JSON,\"stack\":{\"runImage\":{\"image\":\"$RUN_IMAGE\",\"mirrors\":[]},\"uid\":1001,\"gid\":1000},\"lifecycle\":{\"version\":\"$LIF_VER_FULL\",\"api\":{\"buildpack\":\"0.10\",\"platform\":\"0.12\"}}}"
STACK_META="{\"id\":\"$STACK_ID\"}"

# 2.6 Dockerfile
cat > "$CTX/Dockerfile" <<EOF
FROM $BUILDER_BASE_IMAGE

ENV CNB_STACK_ID=io.buildpacks.stacks.aether \
    CNB_USER_ID=1001 \
    CNB_GROUP_ID=1000 \
    CNB_PLATFORM_API=0.12

RUN set -eux; \
    if command -v apt-get >/dev/null 2>&1; then \
      export DEBIAN_FRONTEND=noninteractive; \
      apt-get update -qq; \
      apt-get install -y -qq --no-install-recommends bash curl unzip xz-utils git ca-certificates build-essential libssl-dev pkg-config passwd; \
      rm -rf /var/lib/apt/lists/*; \
    elif command -v dnf >/dev/null 2>&1; then \
      dnf install -y bash curl unzip xz git ca-certificates gcc gcc-c++ make openssl-devel pkgconf-pkg-config shadow-utils; \
      dnf clean all; \
      rm -rf /var/cache/dnf; \
    elif command -v apk >/dev/null 2>&1; then \
      apk add --no-cache bash curl unzip xz git ca-certificates build-base openssl-dev pkgconf shadow; \
    elif command -v pacman >/dev/null 2>&1; then \
      pacman -Sy --noconfirm --needed bash curl unzip xz git ca-certificates base-devel openssl pkgconf shadow; \
      pacman -Scc --noconfirm; \
    elif command -v zypper >/dev/null 2>&1; then \
      zypper --non-interactive refresh; \
      zypper --non-interactive install bash curl unzip xz git ca-certificates gcc gcc-c++ make libopenssl-devel pkg-config shadow; \
      zypper clean --all; \
    else \
      echo "No supported package manager found in builder base image" >&2; \
      exit 1; \
    fi; \
    groupadd -g 1000 cnb 2>/dev/null || true; \
    useradd -m -u 1001 -g 1000 -s /bin/bash cnb 2>/dev/null || true

COPY cnb /cnb

RUN chown -R 1001:1000 /cnb

ARG BUILDER_META
ARG STACK_META
LABEL io.buildpacks.builder.metadata="\$BUILDER_META" \
      io.buildpacks.stack.id="\$STACK_META"

WORKDIR /workspace
USER 1001:1000
EOF

# ---------------------------------------------------------------------------
# 3. Build + push
# ---------------------------------------------------------------------------
info "buildando $BUILDER_TAG..."
podman build \
  --build-arg BUILDER_META="$BUILDER_META" \
  --build-arg STACK_META="$STACK_META" \
  -t "$BUILDER_TAG" -f "$CTX/Dockerfile" "$CTX"
info "publicando no registry local..."
podman push "$BUILDER_TAG"

info "pronto: $BUILDER_TAG"
info "use: pack build my-app -B 127.0.0.1:5000/builder:node-spa --docker-host=inherit --platform linux/$(uname -m)"
