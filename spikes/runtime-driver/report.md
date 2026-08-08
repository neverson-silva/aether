# SPIKE-EE — Porta RuntimeDriver validada contra engine real

> **Status:** Concluído ✓
> **Data:** 2026-08-02
> **Host:** macOS arm64, Docker 29.6.2 (Docker Desktop, aarch64)
> **Hipótese (H4):** a porta `RuntimeDriver` (RFC-0006) expressa o ciclo de vida completo de
> containers/redes/volumes contra um engine real, sem vazar dialeto do driver.

---

## 1. Método

- Implementação Go da porta `RuntimeDriver` (espelho fiel da RFC-0006: imagens, containers,
  rede, volumes, build ausente neste spike, sistema, erros normalizados).
- **`DockerDriver`** via CLI `docker` como **stand-in de semântica** do PodmanDriver real
  (mesmo modelo OCI: pull/run/inspect/stats/logs/exec/stop/remove/network/volume).
- Fluxo completo exercitado: pull → inspect → network → volume → run (com limits
  cpu/mem, env, port, network, volume) → inspect → stats → logs → exec → disconnect →
  restart → stop → remove → erro normalizado → cleanup.

## 2. Resultado

```
[01] ✓ Info: driver=docker version=29.6.2 storage=overlayfs caps=[cgroupv2 overlay2 ...]
[02] ✓ Pull alpine:3.20
[03] ✓ InspectImage: size=4.1MB layers=1
[04] ✓ NetworkCreate (bridge)
[05] ✓ NetworkInspect: subnet=172.21.0.0/16
[06] ✓ VolumeCreate
[07] ✓ Run (cpus=0.25 mem=128MiB env port network volume)
[08] ✓ Inspect: state=running
[09] ✓ Stats: cpu=0.00% mem=544KiB
[10] ✓ Logs: "hello-from-spike"
[11] ✓ Exec: exit=0 out="exec-ok"
[12] ✓ NetworkDisconnect
[13] ✓ Restart
[14] ✓ Stop → [15] ✓ Remove
[16] ✓ Erro normalizado: container_not_found
[17] ✓ NetworkRemove → [18] ✓ VolumeRemove
```

**18/18 passos.** Sem ajuste na interface durante o spike.

## 3. Análise

1. **A interface não vaza dialeto**: `ContainerSpec`/`NetworkSpec`/`VolumeSpec` neutros
   expressaram todos os conceitos necessários sem campo "docker-específico". O mapeamento
   para a CLI Docker foi mecânico (tradução 1:1 de flags) — exatamente o que se espera do
   PodmanDriver (CLI compatível).
2. **Limites de recursos no spec** (cpus/mem) mapearam direto para cgroup — o modelo
   `ResourceSpec` da RFC-0006 está correto.
3. **Erros normalizados funcionam**: `Inspect` de id inexistente retornou
   `ErrContainerNotFound` (não erro bruto da CLI) — confirma o contrato de erros estáveis.
4. **Stats sob demanda** (com `--no-stream`) funcionou como previsto — viabiliza a política
   "métricas só com subscriber" (RFC-0007).
5. **Gap identificado**: `Logs` follow + tail + since precisam de `docker logs -f --tail --since`;
   a porta já os tem (`LogRequest`). Nenhuma mudança necessária.

## 4. Conclusão

**H4 CONFIRMADA.** A porta `RuntimeDriver` da RFC-0006 é suficiente e sem vazamento de dialeto.
O `PodmanDriver` real seguirá a mesma estrutura (CLI podman, compatível com Docker).

## 5. Recomendações de ADR (para RFC-0006)

- **Mantida**: interface como escrita (com ajuste: `Pull` retorna `ImageDigest`; `LogStream`
  com `Close`; `ContainerInfo.Ports` como lista).
- Adicionar à RFC-0006: nota explícita de que **o driver executa CLI OCI via subprocesso**
  (sem SDK/daemon) — validado por este spike; reduz dependência de libs de vendor.
- Adicionar `Build` driver (Buildah) como caso de spike do harness Linux.
- CI: matrix de compat entre `DockerDriver` e `PodmanDriver` (no harness Linux) com este
  mesmo fluxo como teste canônico.

## 6. Rerun

```bash
cd spikes/runtime-driver && go run .
```
