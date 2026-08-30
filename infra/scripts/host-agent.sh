#!/usr/bin/env bash
# Aether host agent — collects REAL host metrics on macOS and Linux.
#
# The API runs inside a container; its own view of /proc reports
# the runtime, not the host. This agent runs natively on the host machine and
# writes $STATE_DIR/host-stats.json, which is mounted into the API container
# ($AETHER_STATE -> /var/lib/aether). The API reads it and falls back to
# runtime metrics when the file is missing or stale.
#
# Lifecycle is owned by infra/scripts/host-watchdog.sh (started by install.sh), which
# starts this agent only while the API is up and stops it whenever the API
# goes down for any reason.
set -u

STATE_DIR="${AETHER_STATE:-$HOME/.aether}"
OUT="${AETHER_HOST_STATS:-$STATE_DIR/host-stats.json}"
INTERVAL="${AETHER_HOST_AGENT_INTERVAL:-2}"

mkdir -p "$STATE_DIR"

OS="$(uname -s)"

cpu_percent() {
  case "$OS" in
    Darwin)
      # "CPU usage: 8.6% user, 5.3% sys, 86.0% idle"
      top -l 1 -n 0 -s 0 2>/dev/null | awk '/CPU usage:/ {gsub(/%/,"",$7); print 100-$7}'
      ;;
    *)
      # /proc/stat deltas over ~0.5s (idle = idle + iowait).
      awk 'NR==1 {t1=$2+$3+$4+$5+$6+$7+$8; i1=$5+$6} END {t2=$2+$3+$4+$5+$6+$7+$8; i2=$5+$6; d=t2-t1; if (d>0) printf "%.1f", (d-(i2-i1))/d*100}' \
        <(sed -n '1p' /proc/stat) <(sleep 0.5; sed -n '1p' /proc/stat)
      ;;
  esac
}

cpu_cores() {
  case "$OS" in
    Darwin) sysctl -n hw.ncpu 2>/dev/null || echo 1 ;;
    *) nproc 2>/dev/null || grep -c '^processor' /proc/cpuinfo || echo 1 ;;
  esac
}

mem_stats() {
  local total used pct
  case "$OS" in
    Darwin)
      local psz free_pages speculative inactive avail
      psz=$(vm_stat 2>/dev/null | awk -F'bytes' '/page size of/ {gsub(/[^0-9]/,"",$1); print $1}')
      psz=${psz:-4096}
      total=$(sysctl -n hw.memsize 2>/dev/null || echo 0)
      free_pages=$(vm_stat 2>/dev/null | awk '/Pages free/ {gsub(/\.|: /,"",$3); print $3}')
      speculative=$(vm_stat 2>/dev/null | awk '/Pages speculative/ {gsub(/\.|: /,"",$3); print $3}')
      inactive=$(vm_stat 2>/dev/null | awk '/Pages inactive/ {gsub(/\.|: /,"",$3); print $3}')
      free_pages=${free_pages:-0}; speculative=${speculative:-0}; inactive=${inactive:-0}
      avail=$(( (free_pages + speculative + inactive) * psz ))
      used=$(( total - avail ))
      [[ $used -lt 0 ]] && used=0
      ;;
    *)
      read -r total_kb avail_kb <<<"$(awk '/MemTotal|MemAvailable/ {print $2}' /proc/meminfo | tr '\n' ' ')"
      total=$(( total_kb * 1024 )); avail=$(( avail_kb * 1024 ))
      used=$(( total - avail ))
      [[ $used -lt 0 ]] && used=0
      ;;
  esac
  pct=0
  [[ $total -gt 0 ]] && pct=$(awk -v u="$used" -v t="$total" 'BEGIN {printf "%.1f", u/t*100}')
  echo "$total $used $pct"
}

disk_stats() {
  # Boot volume in 1024-byte blocks: "Filesystem Size Used Avail Capacity Mounted on"
  df -k / 2>/dev/null | awk 'NR==2 {printf "%d %d %d", $2*1024, $3*1024, $4*1024}'
}

net_stats() {
  case "$OS" in
    Darwin)
      # Cumulative bytes across all interfaces (excluding lo0).
      netstat -ib 2>/dev/null | awk '
        $1 != "lo0" && $7 ~ /^[0-9]+$/ && $10 ~ /^[0-9]+$/ { rx += $7; tx += $10 }
        END { print rx, tx }'
      ;;
    *)
      # /proc/net/dev: cumulative bytes, excluding loopback.
      awk 'NR>2 {gsub(":","",$1); if ($1 != "lo") { rx += $2; tx += $10 }} END { print rx, tx }' /proc/net/dev
      ;;
  esac
}

os_string() {
  case "$OS" in
    Darwin)
      local k v
      k=$(uname -s | tr 'A-Z' 'a-z')
      v=$(sw_vers -productVersion 2>/dev/null || echo "")
      echo "$k $v"
      ;;
    *)
      local name
      name=$(sed -n 's/^PRETTY_NAME="\?\([^"]*\)"\?/\1/p' /etc/os-release 2>/dev/null | head -1)
      echo "${name:-$(uname -s)} $(uname -r)"
      ;;
  esac
}

load_stats() {
  case "$OS" in
    Darwin) sysctl -n vm.loadavg 2>/dev/null | sed 's/[{},]/ /g' | xargs | tr ' ' ',' ;;
    *) awk '{print $1","$2","$3}' /proc/loadavg 2>/dev/null ;;
  esac
}

uptime_stats() {
  case "$OS" in
    Darwin)
      local boot now
      boot=$(sysctl -n kern.boottime 2>/dev/null | awk -F'sec = ' '{split($2,a,","); print a[1]}')
      now=$(date +%s)
      echo $(( now - boot ))
      ;;
    *) awk '{printf "%.0f", $1}' /proc/uptime 2>/dev/null ;;
  esac
}

esc() { sed 's/"/\\"/g' <<<"$1"; }

sample() {
  local cpu cores mem_t mem_u mem_p disk_t disk_u disk_a net_rx net_tx load uptime now
  cpu=$(cpu_percent)
  cores=$(cpu_cores)
  read -r mem_t mem_u mem_p <<<"$(mem_stats)"
  read -r disk_t disk_u disk_a <<<"$(disk_stats)"
  disk_p=0
  if [[ ${disk_t:-0} -gt 0 ]]; then
    disk_p=$(awk -v u="${disk_u:-0}" -v t="$disk_t" 'BEGIN {printf "%.1f", u/t*100}')
  fi
  read -r net_rx net_tx <<<"$(net_stats)"
  load="[$(load_stats)]"
  uptime=$(uptime_stats)
  now=$(date +%s)

  cat > "$OUT" <<EOF
{"ts":$now,"source":"host-agent","cpu_percent":${cpu:-0},"cpu_cores":${cores:-1},"mem_total":${mem_t:-0},"mem_used":${mem_u:-0},"mem_percent":${mem_p:-0},"disk_total":${disk_t:-0},"disk_used":${disk_u:-0},"disk_percent":${disk_p:-0},"net_rx_bytes":${net_rx:-0},"net_tx_bytes":${net_tx:-0},"load":${load:-0},"uptime":${uptime:-0},"hostname":"$(esc "$(hostname)")","os":"$(esc "$(os_string)")"}
EOF
}

if [[ "${1:-}" == "--once" ]]; then
  sample
  exit 0
fi

# The watchdog (host-watchdog.sh) owns start/stop; run until killed.
trap 'exit 0' TERM INT
while true; do
  sample
  sleep "$INTERVAL"
done
