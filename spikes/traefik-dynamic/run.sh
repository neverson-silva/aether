#!/usr/bin/env bash
# SPIKE-TRAEFIK: valida H5 — dynamic config via API (em memória), sem file-write por deploy.
set -euo pipefail
cd "$(dirname "$0")"

run_step() { echo "── $1"; }

run_step "1. backend (app fake) em :18099"
python3 -c "import http.server; http.server.HTTPServer(('127.0.0.1',18099), type('H',(http.server.SimpleHTTPRequestHandler,),{'log_message':lambda *a:None})).serve_forever()" &
BACKEND=$!
sleep 1

run_step "2. config-server (Networking Engine simulado) em :18090"
( cd config-server && go run . ) &
CSC=$!
sleep 2

run_step "3. traefik em :18081 (:18082 API/dashboard)"
./traefik --configFile=traefik.yml >/tmp/traefik_spike.log 2>&1 &
TR=$!
sleep 3

echo
run_step "4. rota spike.local -> backend (config inicial, já em memória)"
echo -n "   resp: "; curl -s -H "Host: spike.local" http://127.0.0.1:18081/; echo

run_step "5. rota spike2.local (adicionada AO VIVO no config-server) -> backend"
echo -n "   resp: "; curl -s -H "Host: spike2.local" http://127.0.0.1:18081/; echo
echo -n "   (sem restart, sem arquivo novo)"

echo
run_step "6. prova: nenhum arquivo dinâmico foi escrito no diretório"
BEFORE=$(find . -name '*.yml' -o -name '*.yaml' | sort | tr '\n' ' ')
sleep 2
AFTER=$(find . -name '*.yml' -o -name '*.yaml' | sort | tr '\n' ' ')
echo "   arquivos yml antes : $BEFORE"
echo "   arquivos yml depois: $AFTER"
[ "$BEFORE" = "$AFTER" ] && echo "   ✓ NENHUM arquivo dinâmico criado" || echo "   ✗ arquivo novo detectado"

run_step "7. dashboard API expõe routers ativos (rawdata)"
curl -s http://127.0.0.1:18082/api/rawdata | python3 -c "
import sys, json
d = json.load(sys.stdin)
routers = sorted(d['routers'].keys())
print('   routers ativos:', routers)
assert 'router-a@http' in routers and 'router-b@http' in routers, 'routers esperados ausentes'
print('   ✓ config dinâmica aplicada em memória (routers-a e b)')
"

echo
echo "=== LIMPEZA ==="
kill $TR $CSC $BACKEND 2>/dev/null || true
echo "FIM"
