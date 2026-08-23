#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

export PATH="$PATH:$HOME/.local/bin"
export DOCKER_HOST="${DOCKER_HOST:-unix://${XDG_RUNTIME_DIR:-/run/user/$(id -u)}/podman/podman.sock}"
BUILDER="${AETHER_CNB_BUILDER:-${AETHER_REGISTRY_ADDR:-127.0.0.1:5000}/builder:node-spa}"

command -v pack >/dev/null 2>&1 || { echo "pack não encontrado"; exit 1; }
command -v podman >/dev/null 2>&1 || { echo "podman não encontrado"; exit 1; }

case "$(uname -m)" in
  x86_64|amd64) CNB_ARCH="amd64" ;;
  aarch64|arm64) CNB_ARCH="arm64" ;;
  *) echo "arch não suportada"; exit 1 ;;
esac
PLATFORM="linux/$CNB_ARCH"

bash infra/builders/build-builder.sh >/dev/null 2>&1 || true

pass=0
fail=0

run_case() {
  local name="$1" dir="$2" port="$3" expect="$4" fallback="${5:-0}"
  local img="cnb-e2e-$name"
  local log="/tmp/cnb-e2e-$name.log"
  if ! pack build "$img" -p "$dir" -B "$BUILDER" --docker-host=inherit --platform "$PLATFORM" >"$log" 2>&1; then
    echo "FAIL $name (build)"
    tail -6 "$log"
    fail=$((fail + 1))
    return
  fi
  podman rm -f "cnb-e2e-$name-run" >/dev/null 2>&1 || true
  if ! podman run -d --name "cnb-e2e-$name-run" -e "PORT=$port" -p "127.0.0.1:$port:$port" "$img" >/dev/null; then
    echo "FAIL $name (run)"
    fail=$((fail + 1))
    return
  fi
  sleep 2
  local body code ok=0
  body="$(curl -fsS "http://127.0.0.1:$port/" 2>/dev/null || true)"
  case "$body" in *"$expect"*) ok=1 ;; esac
  if [ "$fallback" = "1" ]; then
    code="$(curl -sS -o /dev/null -w '%{http_code}' "http://127.0.0.1:$port/qualquer-rota" 2>/dev/null || true)"
    [ "$code" = "200" ] || ok=0
  fi
  podman rm -f "cnb-e2e-$name-run" >/dev/null 2>&1 || true
  if [ "$ok" = "1" ]; then
    echo "PASS $name"
    pass=$((pass + 1))
  else
    echo "FAIL $name (http: body=${body:0:60})"
    fail=$((fail + 1))
  fi
}

run_case node-simple infra/testdata/cnb/node-simple 39001 "aether node fixture"
run_case nestjs infra/testdata/cnb/nestjs 39002 "aether nestjs fixture"
run_case react-vite infra/testdata/cnb/react-vite 39003 "react" 1
run_case vue-vite infra/testdata/cnb/vue-vite 39004 "vue" 1
run_case angular infra/testdata/cnb/angular 39005 "AngMin" 1
run_case pnpm-node infra/testdata/cnb/pnpm-node 39006 "aether pnpm fixture"

echo "---"
echo "PASS=$pass FAIL=$fail"
[ "$fail" -eq 0 ]
