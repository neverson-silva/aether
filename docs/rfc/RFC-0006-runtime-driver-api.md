# RFC-0006 — Runtime Driver API

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P9
- **Dependências:** RFC-0001

---

## 1. Objetivo

Definir a interface canônica que o **Execution Engine** usa para conversar com qualquer
implementação de runtime OCI. É o "contrato" que torna Podman uma mera implementação e que
permite drivers futuros (Docker, containerd, Kubernetes).

## 2. Escopo

**Dentro:** definição completa da interface `RuntimeDriver` e tipos de dados de entrada/saída.
**Fora:** implementação de drivers; UI; lógica de negócio.

## 3. Responsabilidades

- Contrato estável e versionado (`RuntimeDriver`).
- Semântica de erros normalizada.
- Semântica de assíncrono (operações longas) com status.
- Suporte a *capability detection* (o que o driver suporta).

## 4. Arquitetura

```
ExecutionEngine (usa porta)
      │
   RuntimeDriver (interface — esta RFC)
      │
  ┌───┼───────────┬───────────────┐
Podman   Docker   containerd    k8s
Driver   Driver   Driver        Driver
```

## 5. Fluxos

N/A (interface; fluxos em RFC-0001). Exceções: fluxo de detecção de capacidade:

```
RuntimeDriver.Info(ctx) → RuntimeInfo{ driver, version, storageDriver, caps[] }
```

## 6. Interfaces

```go
// versão 1.0 da porta
type RuntimeDriver interface {
    // --- Imagens ---
    Pull(ctx context.Context, ref ImageRef, auth *RegistryAuth) (ImageDigest, error)
    Push(ctx context.Context, ref ImageRef, auth *RegistryAuth) error
    InspectImage(ctx context.Context, ref ImageRef) (*ImageInfo, error)
    ListImages(ctx context.Context, filter ImageFilter) ([]ImageInfo, error)
    PruneImages(ctx context.Context, policy PrunePolicy) (*PruneReport, error)

    // --- Containers ---
    Run(ctx context.Context, spec ContainerSpec) (*ContainerHandle, error)
    Start(ctx context.Context, id ContainerID) error
    Stop(ctx context.Context, id ContainerID, timeout time.Duration) error
    Restart(ctx context.Context, id ContainerID, timeout time.Duration) error
    Remove(ctx context.Context, id ContainerID, opts RemoveOpts) error
    Inspect(ctx context.Context, id ContainerID) (*ContainerInfo, error)
    List(ctx context.Context, sel ContainerSelector) ([]ContainerInfo, error)
    Exec(ctx context.Context, id ContainerID, req ExecRequest) (*ExecResult, error)
    Logs(ctx context.Context, id ContainerID, req LogRequest) (LogStream, error)
    Stats(ctx context.Context, id ContainerID) (*ContainerStats, error)

    // --- Rede ---
    NetworkCreate(ctx context.Context, name string, spec NetworkSpec) (NetworkID, error)
    NetworkRemove(ctx context.Context, id NetworkID) error
    NetworkInspect(ctx context.Context, id NetworkID) (*NetworkInfo, error)
    NetworkConnect(ctx context.Context, net NetworkID, container ContainerID) error
    NetworkDisconnect(ctx context.Context, net NetworkID, container ContainerID) error

    // --- Volumes ---
    VolumeCreate(ctx context.Context, name string, spec VolumeSpec) (VolumeID, error)
    VolumeRemove(ctx context.Context, id VolumeID, force bool) error
    VolumeInspect(ctx context.Context, id VolumeID) (*VolumeInfo, error)
    VolumeList(ctx context.Context) ([]VolumeInfo, error)

    // --- Build ---
    Build(ctx context.Context, spec BuildSpec) (ImageDigest, error)

    // --- Sistema ---
    Info(ctx context.Context) (*RuntimeInfo, error)
    GC(ctx context.Context, policy GCRequest) (*GCReport, error)
}
```

### Tipos principais (resumo)

```go
type ImageRef struct{ Registry, Repository, Tag, Digest string }
type ContainerSpec struct { /* RFC-0001 */ }
type ContainerStats struct {
    CpuPercent float64; MemBytes uint64; MemLimit uint64
    Pids int; NetRxBytes, NetTxBytes uint64; IOReadBytes, IOWriteBytes uint64
}
type RuntimeInfo struct {
    Driver string; Version string; StorageDriver string
    Rootless bool; Capabilities []string
}
```

### Códigos de erro normalizados

`ErrImageNotFound`, `ErrContainerNotFound`, `ErrNetworkNotFound`, `ErrVolumeNotFound`,
`ErrPermissionDenied`, `ErrInsufficientResources`, `ErrTimeout`, `ErrDriver`, `ErrConflict`.

### Detecção de capacidade

`Capabilities` (ex.: `rootless`, `cgroupv2`, `user-namespaces`, `network-macvlan`,
`build-squash`, `http3`). O core consulta `Info()` uma vez e cacheia; features condicionais
são decididas por capacidade, não por nome do driver.

## 7. Eventos

A porta não emite eventos (quem emite é o Execution Engine — RFC-0001). A porta é síncrona/assíncrona
via retorno + canais; a política de eventos pertence ao executor.

## 8. Casos de uso

| Caso | Uso da porta |
|------|--------------|
| Deploy app imagem | Pull + NetworkCreate/Connect + VolumeCreate + Run |
| Logs | Logs (stream) |
| Métricas | Stats (sob demanda) |
| Build | Build |
| Cleanup | ListImages + PruneImages; VolumeList + VolumeRemove |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Interface única para build + runtime | simplicidade | driver de build (buildah) tem um perfil diferente; aceitável |
| Erros normalizados | robustez | mapeamento por driver precisa de cuidado |
| Síncrono com timeout | previsibilidade | operações longas exigem design de chamadas |

## 10. Decisões

- **D-001:** porta versionada (v1.0); mudanças = v2 com interface nova.
- **D-002:** capabilities declarativas em vez de feature flags por driver.
- **D-003:** `Info()` com cache no core.
- **D-004:** erros normalizados obrigatórios para todos os drivers.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Drivers reais não encaixam na interface | média | matriz de teste com podman/docker real |
| Timeout vs long-run mismatch | média | documentar contratos de timeout |
| Perda de capacidade específica | média | capabilities dinâmicas |

## 12. Alternativas descartadas

- **Interface mínima + type-switch por driver**: descartado (quebra abstração; acoplamento).
- **Eventos na porta**: descartado (separação de concerns; executor decide).
- **Uma porta para build e outra para runtime**: considerada; mantida única por simplicidade
  (revisitar se o build se diferenciar muito).
