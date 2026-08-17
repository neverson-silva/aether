# RFC-0001 — Execution Engine

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1 (OCI First), P2, P3, P4, P9
- **Dependências:** RFC-0006 (Runtime Driver API)

---

## 1. Objetivo

Definir o **Execution Engine**: a única camada da plataforma que conversa com o mundo de
containers. Ele expõe uma semântica **neutra OCI** e delega a execução a um **Runtime Driver**
(podman por padrão). Garante que o runtime nunca seja conhecido pelas camadas superiores (P1).

## 2. Escopo

**Dentro:**
- Interface pública `RuntimeDriver` (ver RFC-0006).
- Specs neutros: `ContainerSpec`, `ComposeSpec`, `BuildSpec`, `NetworkSpec`, `VolumeSpec`.
- Tradução spec → operação do driver.
- Semântica de ciclo de vida, rede, volumes, imagens, build, logs, stats, exec.
- Modelo de conversão Compose → unidades (Quadlet) para a camada Runtime.

**Fora:**
- UI, domínio de aplicação (Applications/Deployments), proxy, certificados, observabilidade.
- Qualquer conhecimento de Podman/Docker específico acima da porta.

## 3. Responsabilidades

- Expõe operações abstratas: `run`, `start`, `stop`, `restart`, `remove`, `exec`, `logs`,
  `stats`, `list`, `pull`, `push`, `prune`, `network.*`, `volume.*`, `build`.
- Garante isolamento de *dialeto* do driver (ex.: diferenças Docker vs Podman).
- Mantém `RuntimeInfo` (versões, storage driver, capacidades) para decidir features.
- Aplica limites de recursos (cgroup) vindos do spec.
- Reporta erros normalizados (códigos estáveis).

## 4. Arquitetura

```
                        +-------------------------------+
   (domínio)            |   Execution Engine             |
  ContainerSpec ──────► |  ┌─────────────────────────┐   |
  ComposeSpec ────────► |  │ Spec Mapper (OCI normal)│   |
  BuildSpec ──────────► |  └───────────┬─────────────┘   |
                        |              │                 |
                        |  ┌───────────▼─────────────┐   |
                        |  │ RuntimeDriver (interface)│   |
                        |  └───────────┬─────────────┘   |
                        +──────────────┼─────────────────+
                                       │
                     ┌─────────────────┼─────────────────┐
                     │                 │                 │
              PodmanDriver      DockerDriver       ContainerdDriver / k8s (futuro)
                     │                 │                 │
              podman CLI/API    docker API        containerd / k8s API
```

## 5. Fluxos

### 5.1 Run (start service)

```
1. Runtime.receive spec (validado)
2. Spec Mapper → operações do driver
3. driver.Pull (se imagem não local)
4. driver.NetworkEnsure (network do app)
5. driver.VolumeEnsure (volumes referenciados)
6. driver.Run(container)
7. verifica status (started)
8. retorna handle; emite runtime.op_finished
```

### 5.2 Build

```
1. BuildSpec → driver.Build (Buildah)
2. cache dir preparado (quota)
3. resultado: ImageDigest
4. push opcional (skopeo) se registry configurado
5. build.finished | build.failed (com logs)
```

### 5.3 Remove / GC

```
1. driver.Stop (graceful, timeout)
2. driver.Remove
3. driver.PruneVolumes (com confirmação)
4. eventos runtime.op_finished + volume.removed
```

## 6. Interfaces

Ver RFC-0006 para a definição completa da porta `RuntimeDriver`.

Specs (resumo):

```go
type ContainerSpec struct {
    Name string
    Image ImageRef
    Command []string
    Env []EnvVar
    Ports []PortBinding
    Networks []string
    Volumes []VolumeMount
    Resources ResourceSpec   // cpu, mem, pids, devices
    Security SecuritySpec    // seccomp, capabilities, readonly
    Healthcheck *HealthcheckSpec
}

type ComposeSpec struct {
    Services []ComposeService
    Networks map[string]ComposeNetwork
    Volumes  map[string]ComposeVolume
}

type BuildSpec struct {
    Context string      // dir
    Dockerfile string
    BuildArgs map[string]string
    Cache *CacheOptions  // enabled, quota
    Target string
    Platform string
}
```

## 7. Eventos

Emitidos: `runtime.op_started`, `runtime.op_finished`, `runtime.op_failed`,
`runtime.image_pruned`, `runtime.volume_pruned`.
Consumidos: `deployment.scheduled`, `app.removed`, `service.scale`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Deploy app por imagem | container roda, unit ativa, health ok |
| Deploy stack compose | N serviços rodando, rede criada |
| Escalar réplicas | driver.Start/Stop de instâncias |
| Exec em container | saída + exit code |
| Logs | stream contínuo sem acumular |
| Rollback | restaurar unit/imagem anterior |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Abstraction layer | independência de driver; testabilidade | custo de mapeamento; perda de features específicas |
| Driver ativo por configuração | troca sem reescrita | precisa manter compat entre drivers |
| Specs neutros OCI | portabilidade | precisamos cobrir gaps entre drivers com fallbacks |

## 10. Decisões

- **D-001:** driver default = Podman (via RFC-0006).
- **D-002:** specs neutros definidos pela plataforma (não vendor).
- **D-003:** erros normalizados com códigos estáveis.
- **D-004:** resources limits aplicados via cgroup (no driver).

## 11. Riscos

| Risco | Prob. | Impacto | Mitigação |
|-------|-------|---------|-----------|
| Divergência semântica entre drivers | média | médio | matrix de compat em CI |
| Feature específica de driver | média | baixo | fallback documentado |
| Latência de mapeamento | baixa | baixo | specs diretos; zero round-trips desnecessários |

## 12. Alternativas descartadas

- **Falar diretamente com Podman/Docker em toda a base**: descartado (P1; lock-in).
- **docker-compose como formato universal**: descartado (sintaxe não é OCI; Compose é mapeado).
- **não abstrair e migrar quando necessário**: descartado (custo de reescrita alto).
