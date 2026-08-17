# RFC-0034 — Runtime OCI genérico + Podman First

## Status: Implementado (migração de acoplamentos docker → OCI/Podman)

## Objetivo
Eliminar acoplamento com o Docker e adotar **Podman como runtime base** (oficial
para produção nos servidores), mantendo o modelo OCI-genérico: qualquer runtime
que fale o protocolo do CLI (`podman`/`docker`) funciona sem mudança de código.

## Auditoria — acoplamentos encontrados (e corrigidos)
| Arquivo | Acoplamento | Correção |
|---------|-------------|----------|
| `internal/core/deploycompose.go` | `exec "docker" compose/inspect` | `c.Driver.Name()` (podman/docker) |
| `internal/core/compose.go` | `exec "docker" compose` up/down | `c.Driver.Name()` |
| `internal/core/images.go` | `LookPath("docker")` (já tinha fallback podman) | mantido (fallback correto) |
| `internal/agent/agent.go` | `autoDriver()` via `/usr/bin/podman` | `exec.LookPath` (podman first) |
| `install.sh` | macOS → Docker Desktop; Linux ambíguo | macOS → **podman machine**; Linux → podman por distro |
| `cmd/aether` / `core.go` | `AETHER_RUNTIME` já preferia podman | mantido (podman first) |

O driver `internal/runtime/cli.go` já era OCI-genérico (`d.name` = binário);
`NewPodman()`/`NewDocker()` são os únicos pontos que escolhem o binário.

## Plano de migração (genérico, base = Podman)
1. **Driver OCI**: `Driver.Name()` é a fonte do binário de runtime — nenhuma
   chamada de exec usa `"docker"` hardcoded.
2. **Podman first**: detecção de runtime prefere `podman` (via `LookPath`), depois
   `docker` como fallback. `AETHER_RUNTIME=podman` explícito para prod.
3. **Compose**: `podman compose up -d` / `podman compose down` (mesmo arquivo
   docker-compose.yml, spec-first).
4. **macOS (dev)**: `brew install podman` + `podman machine init/start` — sem
   Docker Desktop. Desenvolvimento no mesmo runtime de produção.
5. **Linux (prod)**: Podman nativo por distro (apt/dnf/pacman/apk/zypper).

## Validação
- Build limpo; suite 8/8 (driver docker nos testes, que é o fallback; o código
  não tem mais `"docker"` hardcoded fora dos nomes de arquivo `docker-compose.yml`
  e referências a imagens Docker).
- `AETHER_RUNTIME=podman` alterna o runtime sem recompilar.

## Pendências
- Testes E2E com Podman real (CI com podman).
- `podman machine` no dev (instalação local).
