#!/usr/bin/env bash
# Roda todos os spikes que rodam neste host (macOS/Linux).
# podman-linux é excluído: exige execução no alvo Linux (ver harness.sh).
set -euo pipefail
cd "$(dirname "$0")"

echo "=== F0: rodando spikes locais ==="
for d in sql-01-wal sql-02-eventlog; do
  echo
  echo "──── $d ────"
  ( cd "$d" && python3 bench.py )
done

echo
echo "──── runtime-driver ────"
( cd runtime-driver && go run . )

echo
echo "──── traefik-dynamic ────"
( cd traefik-dynamic && bash run.sh ) || echo "traefik-dynamic: verificar portas em uso"

echo
echo "──── lang-go-rust ────"
( cd lang-go-rust && python3 measure.py go-minicore/minicore rust-minicore/target/release/minicore 2>/dev/null || echo "compile primeiro: ver README do spike" )

echo
echo "=== FIM: spikes locais (podman-linux exige host Linux) ==="
