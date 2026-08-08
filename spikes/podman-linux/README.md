# SPIKE-PODMAN — Podman rootless + crun + Quadlet + Buildah

> **Status:** Harness pronto — **aguardando execução em Linux** (Debian 12, cgroup v2).
> **Hipótese (H6):** o modelo de execução da RFC-0001 (daemonless, rootless, Quadlet→systemd,
> build rootless com Buildah, crun como runtime low-level) funciona com os limites de recursos
> esperados e com as invariantes da plataforma.

---

## 1. Por que Linux e não macOS

Podman rootless exige kernel Linux (user namespaces + cgroup v2). O alvo de referência da
plataforma é **Debian 12** (ver [`docs/03-metas-engenharia.md`](../../docs/03-metas-engenharia.md) §1).
O harness é o entregável: roda em servidor limpo e mede o que a RFC-0001 precisa.

## 2. O que o harness valida

| Etapa | Valida |
|-------|--------|
| 1. Instalação mínima | apt/dnf/zypper de `podman crun conmon buildah skopeo fuse-overlayfs` — nada além do necessário |
| 2. Usuário rootless | usuário `aether-agent` simulado + subuid/subgid (userns) |
| 3. Ambiente | storage driver, rootless=true, cgroup v2, runtime = crun |
| 4. Run rootless | pull + run com `--cpus/--memory/--pids-limit` + exec + stats |
| 5. Quadlet | unit systemd user gerada por `.container` → `systemctl --user` → container ativo |
| 6. Buildah | build rootless com `--layers` (cache); mede cold vs warm build |
| 7. Métricas | RSS do processo do container (crun) + tempo de start |

## 3. Execução

```bash
# no servidor Debian 12 limpo:
curl -fsSL https://raw.githubusercontent.com/.../harness.sh | sudo bash
# ou
sudo bash harness.sh
```

## 4. Critérios de aceite esperados (hipóteses)

| Métrica | Esperado | Fonte |
|---------|----------|-------|
| Storage driver rootless | `overlay` (fuse-overlayfs) ou `vfs` fallback | podman info |
| Rootless | `true` | podman info |
| cgroup | `v2` | podman info |
| Runtime low-level | `crun` | podman info |
| RSS por container (alpine idle) | < 15 MiB (crun) | ps do PID |
| Start de container | < 300 ms (imagem quente) | run→vivo |
| Build warm (cache) | ≪ build cold | buildah --layers |
| Quadlet unit | `active (running)` | systemctl --user |

## 5. Registro de resultados

Preencher `results.md` após execução no alvo. Este arquivo é o stub.

## 6. Rerun / notas

- Requer `systemd` user instance habilitado (`loginctl enable-linger aether-spike`).
- `XDG_RUNTIME_DIR=/run/user/$(id -u)` já é tratado no harness.
- Se `fuse-overlayfs` falhar em kernels sem fuse, fallback `vfs` (mais lento) — documentar.
