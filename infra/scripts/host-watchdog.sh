#!/usr/bin/env bash
# Aether host watchdog — couples the host agent to the API lifecycle.
#
# Guarantees:
#   - whenever the API is up, the host agent is running (started on demand);
#   - whenever the API is down (crash, stop, failed boot, any reason), the
#     host agent is terminated immediately.
#
# The watchdog itself is a host process managed by install.sh (pidfile at
# $STATE_DIR/host-agent.pid). It polls the API readiness endpoint and spawns
# infra/scripts/host-agent.sh as a child, killing it whenever the API is unreachable.
set -u

STATE_DIR="${AETHER_STATE:-$HOME/.aether}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_SCRIPT="${AETHER_HOST_AGENT:-$SCRIPT_DIR/host-agent.sh}"
API_URL="${AETHER_API_URL:-http://127.0.0.1:8080/api/v1/ready}"
CHECK_INTERVAL="${AETHER_HOST_WATCHDOG_INTERVAL:-2}"

AGENT_PID=0

stop_agent() {
  if [[ "$AGENT_PID" -gt 0 ]] && kill -0 "$AGENT_PID" 2>/dev/null; then
    kill "$AGENT_PID" >/dev/null 2>&1
    wait "$AGENT_PID" 2>/dev/null
  fi
  AGENT_PID=0
}

api_ready() {
  curl -fsS -m 2 "$API_URL" >/dev/null 2>&1
}

shutdown() {
  stop_agent
  exit 0
}
trap shutdown TERM INT

while true; do
  if api_ready; then
    if [[ "$AGENT_PID" -eq 0 ]] || ! kill -0 "$AGENT_PID" 2>/dev/null; then
      bash "$AGENT_SCRIPT" &
      AGENT_PID=$!
    fi
  else
    stop_agent
  fi
  sleep "$CHECK_INTERVAL"
done
