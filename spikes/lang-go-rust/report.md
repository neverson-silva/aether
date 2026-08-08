# SPIKE-LANG — Decisão de Linguagem: Go vs Rust

> **Status:** Concluído ✓
> **Data:** 2026-08-02
> **Host:** macOS arm64, 10 CPUs, 24 GB RAM
> **Hipótese (H3):** linguagem compilada (Go ou Rust) atende o orçamento de RAM idle
> (< 120 MB alvo, ver [`docs/03-metas-engenharia.md`](../../docs/03-metas-engenharia.md)).
> **Objetivo:** fechar o ADR-001 (docs assumem Go como premissa; F0 valida).

---

## 1. Método

Mini-core equivalente em cada linguagem (mesma carga de trabalho):
- 12 workers ociosos (goroutines Go / threads Rust bloqueadas) — representam os handlers
  de eventos do core.
- Cache em memória com 10k entradas (representa o `mem-lru`).
- Listener TCP idle em 127.0.0.1 (representa a API).
- Builds release otimizados (Go `-s -w`, Rust `lto+strip+panic=abort`).

Métricas: tamanho do binário, startup até READY, RSS idle, threads OS, tempo de build limpo.

## 2. Resultados

| Métrica | Go 1.26.5 | Rust 1.97.1 | Diferença |
|---------|-----------|-------------|-----------|
| Binário (release, stripped) | 2,1 MB | 0,3 MB | Rust 7× menor |
| Startup até READY | 4 ms | 4 ms | empate |
| RSS idle (com cache 10k) | 5,0 MB | 2,4 MB | Rust 2,1× menor |
| Threads OS | 6 | 13 (12 workers + main) | Go multiplexa em GOMAXPROCS |
| Build limpo | 0,22 s | 1,76 s | Go 8× mais rápido |
| RAM por 12 workers ociosos | incluída nos 5,0 MB | 12 threads separadas | ver análise |

## 3. Análise

### 3.1 Ambas cumprem o orçamento com folga colossal

- **H3 CONFIRMADA nas duas linguagens.** Mesmo o "pior caso" (Go, 5,0 MB) é **24× abaixo** do
  alvo de 120 MB do plano de controle. RAM idle **não discrimina** Go de Rust — ambos passam.
- A vantagem de RSS do Rust (2,4 vs 5,0 MB) é **irrelevante** para o objetivo do produto.

### 3.2 Modelo de concorrência (relevante para o core)

- **Go**: goroutines multiplexadas em ~GOMAXPROCS threads OS. O core com "poucos processos
  residentes" mantém threads OS constantes (~6-8) independentemente de N handlers. Modelo
  perfeito para o core **event-driven com muitos workers leves** (handlers de eventos, sagas,
  timers) — sem precisar de framework async.
- **Rust**: ou threads OS (uma por worker) ou async (tokio). Para replicar o modelo do Go,
  Rust exige tokio e `async fn` em toda a base — mais complexidade, mesma utilidade para o
  nosso perfil de carga (dezenas de eventos/s).

### 3.3 Ecossistema OCI — o fator decisivo

- **Toda a pilha OCI que Aether integra é Go**: Podman, Buildah, Skopeo, `containers/image`,
  `containers/storage`, `containers/common`. É a **base técnica do produto**.
- **Em Go**: podemos **embutir** `containers/image` e `containers/storage` diretamente
  (ex.: imagem de push/pull/inspect via skopeo como biblioteca — não como subprocesso), com
  **paridade de comportamento** com a pilha que rodamos. Menos wrapper, menos código de
  tradução, menos bugs de diferença de semântica entre a CLI e a biblioteca.
- **Em Rust**: qualquer integração OCI significa reimplementar ou fazer bindings de
  bibliotecas Go via CGO (custo e fragilidade), ou usar crates de terceiros menos maduros
  (`oci-spec`, `containerd-client`) — retrabalho real, sem benefício de runtime.

### 3.4 Produtividade e risco de entrega

- Build 8× mais rápido, ferramentas maduras, curva mais baixa para a equipe, maior pool de
  talento, menos fricção em CI. Para um projeto que precisa de **paridade de features em v1**
  (muito código de produto, não kernel), a velocidade de Go reduz risco de roadmap.

### 3.5 Trade-offs reconhecidos (argumentos de Rust)

| Argumento pró-Rust | Contraponto |
|--------------------|-------------|
| Menos RAM (2,4 vs 5,0 MB) | irrisório frente ao orçamento de 120 MB |
| Sem GC | GC do Go com pausas sub-ms; irrelevante em <100 ev/s |
| Mais controle fino | não precisamos de controle fino no core |
| Memória segura sem runtime | Go é memory-safe com GC |

## 4. Conclusão

**H3 CONFIRMADA.** Ambas atendem o orçamento. A decisão é guiada por **ecossistema OCI e
velocidade de entrega**, não por micro-benchmark de RSS:

> **Decisão: GO** como linguagem do core/agent (ADR-001 confirmado).
> Rust fica documentado como alternativa viável (especialmente se um componente futuro exigir
> controle de memória extremo — não é o caso).

## 5. Recomendações de ADR

- **ADR-001 (confirmado)**: core e agent em **Go**; binário único estático (CGO=0).
- Drivers de runtime (podman/docker) executam ferramentas OCI via subprocesso/CLI através do
  `RuntimeDriver` (RFC-0006) — compatível com Go ou qualquer linguagem; pode evoluir para
  embutir `containers/image` no RegistryDriver.
- **Proxy**: independente de linguagem (Traefik é Go). 
- UI: SPA servida estática (decisão separada, não deste spike).
- Registrar contramedida: se o orçamento de RAM idle um dia apertar < 15 MB, reavaliar Rust
  para o agent.

## 6. Rerun

```bash
cd spikes/lang-go-rust && cargo build --release && cd go-minicore && go build && cd ..
python3 measure.py go-minicore/minicore rust-minicore/target/release/minicore
```
