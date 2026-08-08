# SPIKE-PODMAN — Resultados de execução (Linux real)

> **Status:** ✅ Concluído — H6 VALIDADA
> **Data:** 2026-08-03
> **Ambiente:** Lima VM (macOS host) — Debian GNU/Linux 13 (trixie), aarch64,
> kernel 6.12.95+deb13-cloud-arm64, systemd, cgroup v2
> **Runtime:** podman 5.4.2, crun, conmon 2.1.12, buildah 1.39.3, netavark, overlay

---

## 1. Resultados por etapa

| Etapa | Resultado |
|-------|-----------|
| 1. Instalação mínima | ✅ podman 5.4.2 / crun instalados via apt (nada além do necessário) |
| 2. Usuário rootless | ✅ aether-spike + subuid/subgid 100000:65536 + linger |
| 3. Ambiente | ✅ storage `overlay`, rootless `true`, cgroup `v2`, runtime `crun` |
| 4. Run rootless | ✅ `--cpus=0.25 --memory=128m --pids-limit=64`; exec OK; stats **0.37% CPU, 45-49 KiB RAM, pids=1** |
| 5. Quadlet | ✅ `.container` → generator → **unit systemd user ativa**, container `systemd-spike-hello` (Up) |
| 6. Buildah | ✅ build rootless com `--layers`; imagem `localhost/spike-app:v1` (9.12 MB); cache OK |
| 7. Métricas | ✅ **RSS do container (crun): 924 KiB**; **start (run→vivo): 102 ms** |

## 2. Descobertas técnicas (bugs reais corrigidos durante a validação)

| # | Descoberta | Correção |
|---|-----------|----------|
| D1 | `podman info` do podman 5.x não tem `.Host.Cgroups.Version` no template | harness usa saída textual |
| D2 | Rootless via `su` sem `DBUS_SESSION_BUS_ADDRESS` → crun falha `sd-bus: Interactive authentication required` | export do bus em todos os blocos `su` |
| D3 | **`MemoryMax`/`CPUWeight`/`PidsMax` são diretivas `[Service]`, não `[Container]`** — o quadlet-generator REJEITA o arquivo inteiro | corrigido em `internal/runtime/quadlet.go` e no harness |
| D4 | Units geradas (generator) são **efêmeras** — `systemctl enable` falha | usar `systemctl --user start` (generator regenera no daemon-reload/boot) |
| D5 | `pkill -u <user>` mata o `user@UID.service` — linger não reinicia sozinho no teste | harness garante `systemctl start user@UID.service` |

## 3. Validação do modelo de execução da RFC-0001

- **Daemonless confirmado**: nenhum daemon de containers em execução; podman é comando.
- **Quadlet + systemd**: workload = unit systemd user; restart policy, recursos e ciclo de vida
  delegados ao systemd — exatamente o design da RFC-0001.
- **Eficiência confirmada**: 924 KiB RSS por container alpine idle; start 102 ms — suporta as
  metas de [`docs/03-metas-engenharia.md`](../../docs/03-metas-engenharia.md).

## 4. Pendências derivadas

- Rodar o harness em **Debian 12 x86_64** (alvo de referência dos docs) para confirmar paridade.
- Instalar o binário Aether na VM para o benchmark idle (ver próxima etapa).
- Nota: template Lima usa Debian 13; diferenças irrelevantes para o modelo (kernel 6.12, cgroup v2).

## 5. Rerun

```bash
limactl start --name=aether-debian template://debian
limactl copy spikes/podman-linux/harness.sh aether-debian:/tmp/harness.sh
limactl shell aether-debian -- sudo bash /tmp/harness.sh
```
