#!/usr/bin/env bash
# SPIKE-LANG: Go vs Rust — mini-core equivalente
# Métricas: tamanho do binário, tempo de startup, RSS idle, threads.

set -euo pipefail
ROOT="$(cd "$(dirname "$0")" && pwd)"

measure() {
  local name="$1"; local bin="$2"; shift 2
  local size threads
  size=$(stat -f%z "$bin")
  threads=$("$@" | grep -c . || true)

  # startup (time até imprimir READY)
  local t0 t1 pid rss
  t0=$(python3 -c 'import time;print(time.time())')
  "$bin" >/tmp/aether_spike_out 2>&1 &
  local bp=$!
  # espera READY
  for _ in $(seq 1 200); do
    if grep -q READY /tmp/aether_spike_out 2>/dev/null; then break; fi
    sleep 0.01
  done
  t1=$(python3 -c 'import time;print(time.time())')
  pid=$(awk '{print $2}' /tmp/aether_spike_out)
  sleep 0.3
  rss=$(ps -o rss= -p "$pid" | tr -d ' ')
  local bsize_mb startup_ms rss_mb
  bsize_mb=$(python3 -c "print(f'{$size/1048576:.1f}')")
  startup_ms=$(python3 -c "print(f'{($t1-$t0)*1000:.0f}')")
  rss_mb=$(python3 -c "print(f'{$rss/1024:.1f}')")
  threads=$(ps -M -p "$pid" -o nthreads= 2>/dev/null | head -1 | tr -d ' ' || echo "?")
  kill "$pid" 2>/dev/null || true
  echo "=== $name ==="
  echo "  binário:      ${bsize_mb} MB ($size bytes)"
  echo "  startup:      ${startup_ms} ms"
  echo "  RSS idle:     ${rss_mb} MB"
  echo "  threads:      ${threads}"
}

echo "== Build Rust =="
( cd "$ROOT/rust-minicore" && cargo build --release --quiet 2>/dev/null || cargo build --release )
RUST_BIN="$ROOT/rust-minicore/target/release/minicore"

echo "== Build Go =="
( cd "$ROOT/go-minicore" && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o minicore . )
GO_BIN="$ROOT/go-minicore/minicore"

measure "Go   (mini-core)" "$GO_BIN"
measure "Rust (mini-core)" "$RUST_BIN"
echo "FIM"
