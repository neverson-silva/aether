#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

BUILDER_NAME="${AETHER_CNB_BUILDER:-aether/builder:node-spa}"
LIFECYCLE_VER="${LIFECYCLE_VER:-0.19.6}"
SPA_BP_VER="${SPA_BP_VER:-0.1.0}"
NODE_BP_VER="${NODE_BP_VER:-0.1.0}"
WORK="${WORK:-$HOME/.aether/builds/builder-cache}"
mkdir -p "$WORK"

info() { echo "[aether-builder] $*"; }

command -v podman >/dev/null 2>&1 || { info "podman não encontrado."; exit 1; }

HOST_ARCH="$(podman info --format '{{.Host.Arch}}' 2>/dev/null || uname -m)"
case "$HOST_ARCH" in
  aarch64|arm64)
    LIFECYCLE_ARCH="linux.arm64"
    ;;
  x86_64|amd64)
    LIFECYCLE_ARCH="linux.x86-64"
    ;;
  *)
    info "arquitetura desconhecida: $HOST_ARCH"
    exit 1
    ;;
esac
info "arquitetura do runtime: $HOST_ARCH (lifecycle $LIFECYCLE_ARCH)"

ctx="$WORK/cnb-buildcontext"
rm -rf "$ctx"
mkdir -p "$ctx"

cp -r "$(pwd)/../buildpacks/spa-static" "$ctx/spa-static"
cp -r "$(pwd)/../buildpacks/node-server" "$ctx/node-server"

cat > "$ctx/buildpack.toml" <<EOF
[[buildpacks]]
  id = "aether/spa-static"
  version = "$SPA_BP_VER"
  path = "/cnb/buildpacks/aether_spa-static/$SPA_BP_VER"

[[buildpacks]]
  id = "aether/node-server"
  version = "$NODE_BP_VER"
  path = "/cnb/buildpacks/aether_node-server/$NODE_BP_VER"
EOF

cat > "$ctx/order.toml" <<'EOF'
[[order]]
  [[order.group]]
    id = "aether/spa-static"
    version = "0.1.0"

[[order]]
  [[order.group]]
    id = "aether/node-server"
    version = "0.1.0"
EOF

python3 - "$ctx" <<'PYEOF'
import json, sys
ctx = sys.argv[1]
bp_list = [
    {"id":"aether/spa-static","name":"Aether Static SPA Server","version":"0.1.0","homepage":"","licenses":[{"Type":"MIT","URI":"https://opensource.org/license/mit"}]},
    {"id":"aether/node-server","name":"Aether Node Server","version":"0.1.0","homepage":"","licenses":[{"Type":"MIT","URI":"https://opensource.org/license/mit"}]},
]
meta = {
    "description":"","buildpacks":bp_list,"extensions":[],
    "stack":{"runImage":{"image":"paketobuildpacks/run-jammy-base","mirrors":None}},
    "lifecycle":{"version":"0.19.6","api":{"buildpack":"0.7","platform":"0.7"}},
    "createdBy":{"name":"Aether","version":"1.0"},
    "images":[{"image":"paketobuildpacks/run-jammy-base","mirrors":None}],
}
order = [
    {"group":[{"id":"aether/spa-static","version":"0.1.0"}]},
    {"group":[{"id":"aether/node-server","version":"0.1.0"}]},
]
open(f"{ctx}/builder-meta.json","w").write(json.dumps(meta, separators=(",",":")))
open(f"{ctx}/order-label.json","w").write(json.dumps(order, separators=(",",":")))
PYEOF

cat > "$ctx/Dockerfile" <<EOF
FROM paketobuildpacks/build-jammy-base

USER root

ADD https://github.com/buildpacks/lifecycle/releases/download/v$LIFECYCLE_VER/lifecycle-v$LIFECYCLE_VER+$LIFECYCLE_ARCH.tgz /tmp/lifecycle.tgz

RUN mkdir -p /cnb/lifecycle /cnb/buildpacks \
  && tar -xzf /tmp/lifecycle.tgz -C /cnb/lifecycle --strip-components=1 \
  && rm -f /tmp/lifecycle.tgz \
  && chown -R 1001:1000 /cnb/lifecycle

COPY spa-static /cnb/buildpacks/aether_spa-static/$SPA_BP_VER
COPY node-server /cnb/buildpacks/aether_node-server/$NODE_BP_VER
COPY buildpack.toml /cnb/buildpack.toml
COPY order.toml /cnb/order.toml

ARG BUILDER_METADATA
ARG BUILDER_ORDER
RUN mkdir -p /cnb && chown 1001:1000 /cnb \
  && printf '%s' "\$BUILDER_METADATA" > /cnb/builder-metadata.json \
  && printf '%s' "\$BUILDER_ORDER" > /cnb/builder-order-label.json

USER 1001:1000

LABEL io.buildpacks.builder.metadata="\$BUILDER_METADATA"
LABEL io.buildpacks.buildpack.order="\$BUILDER_ORDER"
EOF

podman build \
  --build-arg BUILDER_METADATA="$(cat "$ctx/builder-meta.json")" \
  --build-arg BUILDER_ORDER="$(cat "$ctx/order-label.json")" \
  -t "$BUILDER_NAME" "$ctx"
if [[ "$BUILDER_NAME" == */*.*:*/* || "$BUILDER_NAME" == *:*/* ]]; then
  podman push --remove-signatures --tls-verify=false "$BUILDER_NAME"
  info "builder publicado: $BUILDER_NAME"
else
  info "builder local disponível: $BUILDER_NAME"
fi
