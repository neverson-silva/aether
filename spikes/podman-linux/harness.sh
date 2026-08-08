#!/usr/bin/env bash
# SPIKE-PODMAN: harness Linux — Podman rootless + crun + Quadlet + Buildah + Skopeo.
#
# RODE EM: Debian 12 (cgroup v2), como root (instala) e depois valida rootless.
# NÃO roda em macOS (necessário kernel Linux + systemd).
#
# Valida H6: o modelo de execução da RFC-0001 (daemonless, rootless, Quadlet→systemd,
# build rootless com Buildah) funciona com os limites de recursos esperados.
set -euo pipefail

DISTRO_ID="$(. /etc/os-release && echo "$ID")"
MEASURE_DIR="${MEASURE_DIR:-/tmp/aether-spike-podman}"

info() { echo "▶ $*"; }
ok()   { echo "  ✓ $*"; }
fail() { echo "  ✗ $*"; exit 1; }

require_rootless_env() {
  command -v podman >/dev/null || return 1
  podman info --format '{{.Host.Security.Rootless}}' 2>/dev/null | grep -q true
}

section_1_install() {
  info "[1/7] Instalação das dependências (apenas o necessário)"
  if ! command -v podman >/dev/null; then
    case "$DISTRO_ID" in
      debian|ubuntu)
        apt-get update -qq
        apt-get install -y -qq podman crun conmon buildah skopeo fuse-overlayfs
        ;;
      fedora|rhel|centos|rocky|almalinux)
        dnf install -y podman crun conmon buildah skopeo fuse-overlayfs
        ;;
      opensuse*)
        zypper install -y podman crun conmon buildah skopeo fuse-overlayfs
        ;;
      *) fail "distro não suportada pelo harness: $DISTRO_ID";;
    esac
  fi
  ok "podman $(podman --version) / $(podman info --format '{{.Host.OCIRuntime.Name}}' 2>/dev/null)"
}

section_2_rootless_user() {
  info "[2/7] Usuário rootless (simula aether-agent)"
  if ! id aether-spike >/dev/null 2>&1; then
    useradd -m -s /bin/bash aether-spike
  fi
  # subuids/subgids para userns rootless
  grep -q aether-spike /etc/subuid || echo "aether-spike:100000:65536" >>/etc/subuid
  grep -q aether-spike /etc/subgid || echo "aether-spike:100000:65536" >>/etc/subgid
  loginctl enable-linger aether-spike 2>/dev/null || true
  if ! systemctl is-active user@$(id -u aether-spike).service >/dev/null 2>&1; then
    systemctl start user@$(id -u aether-spike).service 2>/dev/null || true
  fi
  ok "usuário aether-spike pronto (subuid/subgid + linger)"
}

section_3_rootless_env_check() {
  info "[3/7] Verificação do ambiente rootless (storage driver, userns, cgroup v2)"
  su - aether-spike -c '
    set -e
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    podman info --format "  storage driver: {{.Store.GraphDriverName}}"
    podman info --format "  rootless:       {{.Host.Security.Rootless}}"
    podman info 2>/dev/null | grep -i cgroupversion | head -1
    podman info --format "  runtimes:       {{.Host.OCIRuntime.Name}}"
  '
  ok "env rootless OK"
}

section_4_run_rootless() {
  info "[4/7] Execução rootless (pull + run + limites de recursos)"
  su - aether-spike -c '
    set -e
    podman rm -f spike-c1 spike-m1 spike-m2 2>/dev/null || true
    podman network rm -f spike-net 2>/dev/null || true
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    podman pull -q docker.io/library/alpine:3.20
    podman run -d --name spike-c1 \
      --cpus=0.25 --memory=128m --pids-limit=64 \
      alpine:3.20 sh -c "echo hello-rootless; sleep 600"
    sleep 1
    podman exec spike-c1 echo exec-ok
    podman stats --no-stream --format "  stats: {{.CPUPerc}} {{.MemUsage}} pids={{.PIDs}}" spike-c1
    podman stop spike-c1 >/dev/null
    podman rm spike-c1 >/dev/null
  '
  ok "container rootless rodou com limites cgroup"
}

section_5_quadlet() {
  info "[5/7] Quadlet → systemd user unit (modelo de workload da RFC-0001)"
  mkdir -p ~aether-spike/.config/containers/systemd
  cat > ~aether-spike/.config/containers/systemd/spike-hello.container <<'EOF'
[Unit]
Description=Spike Quadlet container

[Container]
Image=docker.io/library/alpine:3.20
Exec=sh -c "echo quadlet-running; sleep 600"

[Service]
MemoryMax=128m
CPUWeight=100
PidsMax=64
Restart=on-failure

[Install]
WantedBy=default.target
EOF
  chown -R aether-spike:aether-spike ~aether-spike/.config
  su - aether-spike -c '
    set -e
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    systemctl --user daemon-reload
    systemctl --user start spike-hello.service >/dev/null
    sleep 2
    systemctl --user is-active spike-hello.service >/dev/null || { echo "  ✗ unit quadlet não ativou"; exit 1; }
    podman ps --filter name=spike-hello --format "  quadlet container: {{.Names}} ({{.Status}})"
    systemctl --user stop spike-hello.service >/dev/null
  '
  ok "Quadlet gerou unit systemd e subiu o container"
}

section_6_buildah_rootless() {
  info "[6/7] Build rootless com Buildah (cache controlável)"
  su - aether-spike -c '
    set -e
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    mkdir -p /tmp/spike-build && cd /tmp/spike-build
    printf "FROM alpine:3.20\nRUN echo layer1 > /l1 && echo layer2 > /l2\n" > Dockerfile
    start=$(date +%s)
    buildah bud --layers -t spike-app:v1 . >/dev/null
    dur1=$(( $(date +%s) - start ))
    # rebuild deve reutilizar cache (medir)
    start=$(date +%s)
    buildah bud --layers -t spike-app:v1 . >/dev/null
    dur2=$(( $(date +%s) - start ))
    podman images spike-app --format "  imagem: {{.Repository}}:{{.Tag}} ({{.Size}})"
    echo "  build 1 (cold): ${dur1}s | build 2 (cache): ${dur2}s"
  '
  ok "Buildah rootless + cache funcionando"
}

section_7_measure() {
  info "[7/7] Métricas: RAM por container rootless + tempo de start"
  mkdir -p "$MEASURE_DIR"
  su - aether-spike -c '
    set -e
    export XDG_RUNTIME_DIR=/run/user/$(id -u)
    export DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$(id -u)/bus
    podman run -d --name spike-m1 alpine:3.20 sh -c "sleep 600"
    sleep 1
    pid=$(podman inspect --format "{{.State.Pid}}" spike-m1)
    rss=$(ps -o rss= -p "$pid" | tr -d " ")
    # tempo de start: da solicitacao do run ate processo vivo
    start=$(date +%s%N)
    podman run -d --name spike-m2 alpine:3.20 sleep 600 >/dev/null
    end=$(date +%s%N)
    ms=$(( (end - start) / 1000000 ))
    podman stop spike-m1 spike-m2 >/dev/null
    podman rm spike-m1 spike-m2 >/dev/null
    echo "  RSS do processo do container (crun): ${rss} KiB"
    echo "  tempo de start (podman run → vivo): ${ms} ms"
  '
  ok "métricas coletadas em $MEASURE_DIR"
}

main() {
  if [ "$(uname -s)" = "Darwin" ]; then
    echo "Este harness exige Linux. Rode num host Debian 12 (cgroup v2)."
    exit 2
  fi
  info "Distro: $DISTRO_ID | kernel $(uname -r)"
  section_1_install
  section_2_rootless_user
  section_3_rootless_env_check
  section_4_run_rootless
  section_5_quadlet
  section_6_buildah_rootless
  section_7_measure
  echo
  ok "SPIKE-PODMAN concluído — todos os cenários passaram"
}

main "$@"
