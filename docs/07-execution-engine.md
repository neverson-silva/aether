# 07 — Execution Engine (Runtime Engine OCI)

> **Status:** Núcleo técnico da plataforma.
> **RFC relacionada:** [`RFC-0001`](rfc/RFC-0001-execution-engine.md), [`RFC-0006`](rfc/RFC-0006-runtime-driver-api.md)
> **Princípios atendidos:** P1 (OCI First), P2, P3, P4, P9.

---

## 1. Propósito

O **Execution Engine** é a única porta de entrada para o mundo de containers. Ele converte
*intenções declarativas de alto nível* (rodar um serviço, um stack compose, um worker, um banco)
em operações concretas executadas por um **OCI Runtime Driver**.

A regra arquitetural absoluta: **nenhuma outra parte da plataforma chama `podman`, `docker`,
`containerd` ou fala com o runtime**. Tudo passa pelo Execution Engine.

## 2. Hierarquia conceitual

```
Execution Engine  (semântica neutra: run/stop/build/pull/push/logs/stats/exec/network/volume)
        │
        ├── Runtime Driver  (interface abstrata)
        │       ├── PodmanDriver     ← implementação padrão (v1)
        │       ├── DockerDriver     ← futura
        │       ├── ContainerdDriver ← futura
        │       └── KubernetesDriver ← futura
        │
        └── Build Driver
                └── BuildahDriver    ← v1 (Buildah)
```

- **Interface:** definida em código (interface/trait `RuntimeDriver`), com semântica OCI.
- **Driver:** implementação por ferramenta; carregada por configuração (`runtime.driver = podman`).
- **Estratégia de teste:** drivers são testados com conteiners de verdade em CI matrix
  (podman hoje; docker/containerd na matrix de compatibilidade contínua).

## 3. Contrato do Runtime Driver (resumo)

```go
type RuntimeDriver interface {
    // Imagens
    Pull(ctx, ImageRef, auth) (ImageDigest, error)
    Push(ctx, ImageRef, auth) error
    ExistsImage(ctx, ref) (bool, error)
    PruneImages(ctx, opts) error

    // Containers / pods
    Run(ctx, ContainerSpec) (ContainerHandle, error)
    Start/Stop/Restart(ctx, id) error
    Remove(ctx, id, force) error
    Exec(ctx, id, args, stdin) (ExecResult, error)
    Logs(ctx, id, since, follow) (LogStream, error)
    Stats(ctx, id) (ContainerStats, error)
    List(ctx, selector) ([]ContainerInfo, error)

    // Rede
    NetworkCreate/Delete/Inspect(ctx, name) error
    ContainerConnectToNetwork(ctx, id, net) error

    // Volumes
    VolumeCreate/Delete/Inspect(ctx, name, opts) error

    // Construção
    Build(ctx, BuildSpec) (ImageDigest, error)

    // Sistema
    Info(ctx) (RuntimeInfo, error)   // versões, storage driver, usuário
    GC(ctx, policy) (GCReport, error)
}
```

Specs são **neutros OCI** (`ContainerSpec`, `ComposeSpec`, `BuildSpec`) — definidos pela plataforma,
não pelo vendor. Driver traduz.

## 4. Justificativa profunda: por que Podman + Buildah + Skopeo + Quadlet + conmon + crun

### 4.1 Podman

**O que é:** engine de containers **daemonless**, rootless, que gerencia containers/pods/pods
com a mesma CLI compatível com Docker, mas sem um daemon central.

**Vantagens para Aether:**
1. **Zero daemon residente** — `dockerd` + `containerd` são eliminados; menos RAM e menos processo.
2. **Rootless nativo** — containers rodam como usuário não-privilegiado, com `fuse-overlayfs` ou
   `slirp4netns`/`pasta`; superfície de ataque drasticamente menor (P9).
3. **Storage content-addressable** — camadas deduplicadas automaticamente entre imagens
   (poupança de SSD).
4. **CLI/API compatível** — adotar `docker` é trivial (driver Docker reusa lógica).
5. **Gerencia quadlets e pods** — compatível com orquestração declarativa via systemd.
6. **É o mesmo stack do OCI "de verdade"** (runc/crun por baixo) — nenhuma lock-in.

**Desvantagens / trade-offs:**
- Rootless: redes mais complexas (slirp4netns vs macvlan root); portas < 1024 exigem
  `AmbientCapabilities` no systemd ou redirecionamento.
- Performance de rede rootless um pouco inferior a root em modo bridge; mitigado usando
  network `slirp4netns` com tuning ou rede host/socket quando apropriado.
- Maturidade de alguns recursos avançados (ex.: GPU) menor que Docker; aceitável em v1.

### 4.2 Buildah

**O que é:** ferramenta da família podman para construir imagens OCI **sem daemon**, rootless,
sem precisar de um container em execução para buildar.

**Vantagens:**
1. **Build rootless** — build não requer privilégio; perfeito para nosso modelo de segurança.
2. **Sem daemon de build** (diferente do buildkit/`docker build` via daemon).
3. **Cache controlável** — controle fino de camadas; suporte a `--squash`, `--cache-from`.
4. **Compatível com Dockerfile** — migração trivial de projetos existentes.
5. **Emite OCI image** — pode ser pushada via skopeo.

**Trade-offs:**
- Precisamos implementar fila e controle de concorrência (não há daemon que serialize por nós)
  — exatamente o que queremos (controle total de CPU/IO).
- Cache compartilhado entre builds exige gerenciamento (nosso GC de build).

### 4.3 Skopeo

**O que é:** ferramenta para inspecionar, copiar e gerenciar imagens e registries OCI sem
executar containers.

**Vantagens:**
1. **Copiar imagens entre registries** (registry mirror, migração) com `skopeo copy`.
2. **Inspecionar manifestos sem pull completo** — decisões baratas antes de baixar.
3. **Verificação de assinatura** (cosign/sigstore) — supply chain.
4. **Delegação de push/pull** para imagens de build — se tiver um registry privado, use skopeo.

### 4.4 Quadlet

**O que é:** gerador que converte declarações de containers/pods (arquivos `.container`,
`.pod`, `.volume`, `.network`) em units systemd.

**Vantagens:**
1. **Workloads = units systemd** — systemd dá de graça: restart policy, resource limits
   (`CPUWeight`, `MemoryMax`), socket activation, boot ordering, `systemctl` para tudo.
2. **Sem processo residente extra** — a unit apenas executa `podman run` no start.
3. **Recovery nativo** — após reboot, systemd reconstrói o que estiver habilitado.
4. **Isolamento por usuário** — units rodam como usuário dedicado (systemd user instances).
5. **Modelo declarativo** — o agente gera arquivos `.container`/`.pod`/`.volume`/`.network`
   a partir do spec de domínio; é o "Terraform" do workload.

**Trade-offs:**
- Gerenciar units via agent = um nível de abstração a mais; compensa pela robustez do systemd.
- Need para escrever arquivos de unit; feito pelo UnitManager, nunca manualmente.

### 4.5 conmon

**O que é:** supervisor minimalista de containers (gerencia stdio, PID, status).

**Vantagens:** leve, dedicado; fornece streams de logs, terminal, sinais. Substitui
supervisores pesados.

### 4.6 crun

**O que é:** runtime de baixo nível (OCI runtime spec) escrito em **C**.

**Vantagens:**
1. **Menos RAM por container** que runc (Go) — alinhado à meta de eficiência.
2. **Start mais rápido** que runc em workloads pequenos.
3. **Menos dependências** (C puro, sem Go runtime).

**Trade-offs:** ecossistema menor que runc em casos exóticos; suporte community é bom; por
ser OCI spec, qualquer runtime OCI (runc) é substituível via `runtime = "crun"` na config
do podman.

## 5. Modelo de execução de workload (Units systemd)

### 5.1 Pipeline spec → unit

```
Application spec (domínio) → UnitSpec (Runtime) → arquivo .container/.pod/.volume/.network (Quadlet)
→ systemctl --user enable/start → podman run (crun) + conmon
```

### 5.2 Estrutura de unidades por app

- `app-<id>-main.container` — serviço principal
- `app-<id>-worker.container` — worker
- `app-<id>-db.container` — banco
- `app-<id>-cron.timer` + `.container` — cron job
- `aether-app-<id>.volume`, `.network` — recursos

### 5.3 Gerenciamento pelo agente

- `agent` escreve units, faz `daemon-reload`, `enable --now`, monitora via `systemctl status`.
- Configuração de resource limits é injetada nas units (`CPUWeight`, `MemoryMax`, `PidsMax`).
- Rollback = restaurar a unit da versão anterior + restart.

## 6. Builds (Buildah) — fluxo

```
Buildspec → fila (concorrência limite) → cgroup de build (limites) →
Buildah (rootless, cache na pasta de build) → imagem → registry local (skopeo push)
→ event build.finished
```

- **Cache de build:** pasta dedicada com quota; LRU; camadas base assinadas.
- **Build externo:** usuário pode enviar imagem pronta (nenhum custo local).
- **Falha:** retry com backoff; logs de build salvos e expostos.

## 7. Logs e métricas do runtime

- Logs via `conmon` → arquivos/journals com rotação; streaming via sockets unix (sem acumular).
- Métricas via `cgroup v2` (`memory.current`, `cpu.stat`, `io.stat`) lidas sob demanda —
  nenhum processo extra.

## 8. Migração futura de drivers

| Driver futuro | Esforço | Motivação |
|---------------|---------|-----------|
| Docker | baixo (CLI compat) | usuários com stack legada |
| containerd | médio (não tem CLI completa) | mínimo footprint, K8s |
| Kubernetes | alto (modelo declarativo de recursos) | empresas com cluster |

A existência da interface `RuntimeDriver` + `RuntimeInfo` faz a troca ser config-driven. O
**modelo de domínio não muda**: spec OCI é neutro.

## 9. Riscos e mitigação

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Rootless network complexity | média | usar `slirp4netns`/`pasta`; documentar; rede host para casos de performance |
| Podman version drift entre distros | média | exigir versão mínima; vendoring de runtime próprio no instalador |
| crun compatibilidade exótica | baixa | fallback config para runc |
| Systemd user instance em distros mínimas | baixa | instalador habilita `user@.service` |

## 10. Decisões

- **Decisão:** Podman rootless + crun + Quadlet + Buildah + Skopeo + conmon como stack padrão.
- **Decisão:** `crun` como runtime low-level padrão (fallback runc).
- **Decisão:** rede default por app via network `podman`; portas publicadas via systemd
  `AmbientCapabilities=CAP_NET_BIND_SERVICE` no proxy/agente para portas 80/443.
- **Decisão:** nenhuma imagem de suporte; apenas binários.

## 11. Referências

- RFC-0001 (Execution Engine), RFC-0006 (Runtime Driver API).
- Domínio Runtime: [`06-dominios-sistema.md`](06-dominios-sistema.md) §2.5/2.6.
- Consumo de recursos: [`04-analise-consumo-recursos.md`](04-analise-consumo-recursos.md) §5–7.
