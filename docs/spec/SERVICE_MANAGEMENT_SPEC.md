# RFC-0030 — Gestão de Services da Plataforma Aether

**Especificação de Produto, UX e Arquitetura**

- **Status:** Draft — Especificação
- **Autores:** Aether Product Team (Product, Design, UX, Architecture, Engineering)
- **Data:** 2026-08-05
- **Versão:** 1.0
- **Dependências:** RFC-0006 (Runtime Driver), RFC-0020 (Environments), RFC-0021 (PostgreSQL), RFC-0022 (Environment Variables)

---

**AVISO DE LEITURA:** Este documento é uma especificação completa — não contém código,
componentes, APIs nem implementação. É a fonte única de verdade para todo o desenvolvimento
subsequente. Cada decisão de UX é justificada; cada trade-off é documentado; cada fluxo
possui critérios de aceite explícitos.

---

# 1. Visão do Produto

## 1.1 Filosofia da Plataforma

Aether é uma plataforma PaaS self-hosted. Diferentemente de provedores cloud (Vercel, Railway,
Render), o usuário **possui a infraestrutura** — a Aether é o plano de controle que gerencia
containers, bancos, redes e deployments sobre Docker/Podman/Kubernetes em servidores próprios.

O usuário **nunca deve pensar em Docker**. Ele pensa em entidades de negócio:

- **Applications** (serviços web ou APIs)
- **Databases** (PostgreSQL, MySQL, Redis, MongoDB, SQL Server, Oracle, etc.)
- **Workers** (processamento assíncrono)
- **Cron Jobs** (tarefas agendadas)
- **Queues** (mensageria)
- **AI Services** (LLMs, embeddings, vector DBs)
- **Templates** (stacks prontas)
- **Compose** (multi-service via docker-compose)

Infraestrutura é **consequência** da declaração do usuário — nunca o foco da experiência.

## 1.2 Hierarquia de Entidades

```
Workspace (Org)
  └── Project ("meu-app")
       ├── Project Variables
       ├── Environment: production (default)
       │    ├── Environment Variables
       │    ├── Service: api (Node/Express, port 3000)
       │    │    ├── Deployments
       │    │    ├── Logs
       │    │    ├── Service Variables
       │    │    └── Domains
       │    ├── Service: worker (background)
       │    └── Service: redis (database)
       └── Environment: staging
            ├── Service: api (staging)
            └── Service: postgres (database)
```

**Regras de relacionamento:**
- Um **Workspace** (Org) contém N **Projects**
- Um **Project** contém 1+ **Environments** (criado com `production` default em transação)
- Um **Environment** contém N **Services**
- Um **Service** pertence a exatamente 1 Environment
- Variables: **Service > Environment > Project** (precedência)
- Um Service pode ser de tipos: `application`, `database`, `worker`, `cron`, `compose`

## 1.3 Product Principles

1. **Declarativo sobre imperativo** — o usuário declara o que quer (ex: "Postgres 16 com 2GB RAM"); a plataforma resolve o como.
2. **Progressive disclosure** — complexidade exposta sob demanda; defaults inteligentes cobrem 90% dos casos.
3. **Transparência total** — logs, métricas, eventos e estado são visíveis sem camadas de abstração desnecessárias.
4. **Self-healing** — healthchecks, rollbacks automáticos, retry de deployments.
5. **Zero vendor lock-in** — tudo roda sobre runtimes padrão (Docker/Podman/K8s); exportação completa via `aether.yml`.
6. **Experience-first** — a plataforma antecipa a intenção do usuário (auto-detect de frameworks, sugestões de recursos, templates recomendados).

## 1.4 UX Principles

1. **Contexto sobre navegação** — o usuário sempre sabe onde está (breadcrumb: Workspace > Project > Environment > Service).
2. **Ações agrupadas, não espalhadas** — menus contextuais, command palette, atalhos de teclado.
3. **Feedback imediato** — toda ação tem resposta visual em <200ms (otimista UI); estados longos mostram progresso real.
4. **Consistência absoluta** — mesmo componente para menus, popovers, dropdowns, autocomplete em toda a plataforma.
5. **Keyboard-first quando possível** — toda ação acessível via teclado; power users não precisam de mouse.
6. **Redução de carga cognitiva** — a header transmite apenas contexto; administração fica em menus secundários.

## 1.5 Design Principles

1. **Superfícies em camadas** — background → card → popover → modal, cada uma com cor e elevação próprias.
2. **Bordas mínimas** — separação por contraste de superfície e espaçamento, não por linhas.
3. **Tipografia hierárquica** — display-lg → headline-sm → body-md → body-sm → label-caps → caption.
4. **Sistema de elevação** — shadow-sm (cards) → shadow-md (popovers) → shadow-lg (modais).
5. **Motion com propósito** — transições de 150ms ease-out para hover; 200ms para abrir/fechar; nada de animações decorativas.
6. **Dark-first** — tema escuro como padrão (plataforma de infraestrutura); light theme como opção futura.

## 1.6 Information Architecture

```
Sidebar (persistente)
├── Projects (dashboard)
├── Services (lista global)
├── Databases (lista global)
├── Marketplace
├── Monitoring
├── Schedules
├── Requests (Networking)
│
├── Infrastructure
│   ├── Clusters
│   ├── Nodes (Servers)
│   └── Databases (infra)
│
├── Storage
│   └── S3 Destinations
│
├── Security
│   ├── SSO
│   ├── Certificates
│   ├── Secrets
│   ├── Members
│   └── API Keys
│
└── Platform
    ├── Whitelabeling
    ├── Registry
    ├── Notifications
    ├── CI/CD
    ├── GitOps
    ├── Backups
    └── Marketplace
```

**Navegação contextual (header do projeto):**
```
[Project Name]  [environment ▼]  [Actions ▼]
```
- Switcher de environment: contexto atual (não é um filtro — é onde você está)
- Actions: menu agrupado (Environment Variables, Project Variables, Create Environment, Rename, Set Default, Delete)

## 1.7 Mental Model

O usuário pensa em termos de **"meu app"** — um projeto com múltiplos ambientes. Cada ambiente é uma
"cópia" do app em um estágio diferente do pipeline (dev → staging → production). Dentro de cada
ambiente, serviços concretos (API, banco, worker) materializam a stack.

**Analogia mental:** um projeto é como um repositório Git com branches (environments). Cada branch
tem seu próprio conjunto de serviços. Trocar de environment é como `git checkout`.

---

# 2. Fluxo Create Service

## 2.1 Entry Points

O usuário pode iniciar a criação de um Service por 6 caminhos:

1. **Botão "Create Service" na página do Projeto** (dentro de um Environment) — fluxo mais comum para usuários existentes.
2. **Botão "New Service" na listagem global de Services** — requer seleção de projeto + environment.
3. **Command Palette** (`Cmd+K`) → "Create Service" — power users.
4. **Launcher** (atalho `Cmd+Shift+K`) — interface unificada de criação (ver Capítulo 3).
5. **Marketplace → Install** em um template — preenche o formulário automaticamente.
6. **Import aether.yml** — criação em lote via GitOps.

## 2.2 Modal vs Full Page vs Wizard

**Alternativas consideradas:**
- **Full page dedicada** — boa para formulários complexos, mas tira o contexto do projeto.
- **Wizard multi-step** — reduz carga cognitiva por etapa, mas adiciona fricção para casos simples (ex: "nginx:alpine, porta 80").
- **Modal com progressive disclosure** — melhor balanço: formulário simples por padrão; seções avançadas expandíveis.

**Decisão:** **Modal com progressive disclosure.**

Justificativa:
- 80% dos serviços são criados com: imagem + porta (ou git URL + branch). Isso cabe num modal de 2 campos.
- Para os 20% que precisam de healthcheck, resources, volumes, env vars — seções expandíveis dentro do modal.
- Mantém o contexto visual (o usuário vê o projeto atrás do modal) e é consistente com o resto da plataforma.

## 2.3 Estrutura do Modal

### Estado Inicial (Simples)
```
┌─────────────────────────────────────────────────┐
│  Create Service                              ✕  │
├─────────────────────────────────────────────────┤
│                                                 │
│  Source                                         │
│  ┌─────────────────────────────────────────┐   │
│  │ ○ OCI Image    ○ Git Repository         │   │
│  │ ○ Template     ○ Docker Compose          │   │
│  │ ○ Database                              │   │
│  └─────────────────────────────────────────┘   │
│                                                 │
│  ── OCI Image ──                               │
│  Image                                          │
│  ┌─────────────────────────────────────────┐   │
│  │ nginx:alpine                        📋  │   │
│  └─────────────────────────────────────────┘   │
│  Port                                           │
│  ┌─────────────────────────────────────────┐   │
│  │ 80                                      │   │
│  └─────────────────────────────────────────┘   │
│                                                 │
│  [+ Advanced settings]                          │
│                                                 │
│              [Cancel]    [Create & Deploy ▼]    │
└─────────────────────────────────────────────────┘
```

O botão "Create & Deploy ▼" tem um split: clique principal = criar + deploy imediato; dropdown = criar sem deploy.

### Estado Expandido (Advanced)
Ao clicar "+ Advanced settings", revela seções colapsáveis:
- **Resources**: CPU (slider 0.25–8 cores), Memory (slider 128MB–32GB), GPU (toggle)
- **Health Check**: path, interval, timeout, retries
- **Volumes**: nome + mount path (lista dinâmica com botão "+")
- **Environment Variables**: editor KEY=value inline (3 linhas, com link "Open full editor")
- **Networking**: publish port (auto/default/custom), domain (create new or link existing)
- **Build** (para git): Dockerfile path, build method (dockerfile/nixpacks/buildpacks), build args

Cada seção tem um badge com o valor default aplicado (ex: "CPU: 1 core", "Health: /"). Isso transmite
que o usuário não PRECISA expandir — os defaults já são production-ready.

## 2.4 Fluxo Template

Quando o source é "Template":
- O modal expande para mostrar um grid de templates (recentes + populares + busca).
- Selecionar um template preenche automaticamente imagem, porta, resources, volumes e env vars defaults.
- O usuário pode ajustar qualquer campo antes de criar.
- Templates têm um badge "Verified" ou "Community".

## 2.5 Estados e Feedback

### Loading States
- **Puxando imagem**: após criar, a linha do serviço na lista mostra um skeleton shimmer até o container iniciar.
- **Deploy em progresso**: timeline de deploy inline (etapas: pulling → building → starting → healthcheck → ready).
- **Timeout**: após 5 minutos sem healthcheck, mostra "Deploy taking longer than expected" com link para logs.

### Empty States
- **Nenhum serviço no ambiente**: ilustração + "No services yet. Create one to get started." + CTA Create Service.
- **Nenhum deploy**: "Deploy your service to see it here." + CTA Deploy.

### Error States
- **Image pull failed**: mensagem clara com o erro do registry + sugestão (ex: "Check image name and registry access").
- **Build failed**: link direto para os logs de build com a linha do erro destacada.
- **Port conflict**: sugere porta alternativa ou parar o serviço conflitante.
- **Out of resources**: sugere reduzir recursos ou adicionar mais nodes ao cluster.

## 2.6 Keyboard Navigation

- `Tab` / `Shift+Tab`: navega entre campos
- `Enter` no último campo: submit (Create)
- `Escape`: fecha o modal (com confirmação se houver dados não salvos)
- `Cmd+Enter`: Create & Deploy
- `Cmd+Shift+Enter`: Create without deploy

## 2.7 Progressive Disclosure

A filosofia: o modal começa com **2 campos** (imagem + porta) e cresce sob demanda.
Isso reduz a ansiedade de "formulário gigante" e faz o fluxo parecer simples mesmo
quando o resultado é complexo.

Campos nunca desaparecem — apenas novas seções aparecem abaixo.
Isso evita "onde foi parar aquele campo que eu vi antes?"

---

# 3. Launcher

## 3.1 Conceito

O Launcher é uma **interface unificada de criação e navegação** inspirada em Raycast,
Spotlight e VSCode Command Palette. Diferentemente de um menu dropdown tradicional, o
Launcher é uma **overlay central** que ocupa o foco total do usuário, permitindo:

- Criar qualquer entidade (Service, Database, Environment, Project, Cron Job, Worker...)
- Navegar para qualquer página (projetos, serviços, settings)
- Executar ações rápidas (deploy, restart, rollback, view logs)
- Buscar templates do Marketplace
- Ver recentes e favoritos
- Filtrar por categoria

## 3.2 Ativação

- **Atalho global**: `Cmd+Shift+K` (configurável)
- **Ícone na sidebar**: botão "Launch" no footer da sidebar
- **Campo de busca no dashboard**: ao focar, expande para o Launcher

## 3.3 Layout

```
┌──────────────────────────────────────────────────────────┐
│  ██████████████████████████████████████████████████████  │
│  █ Search or create...                              █  │
│  ██████████████████████████████████████████████████████  │
│                                                          │
│  RECENTES                                                │
│  ┌──────────────────────────────────────────────────────┐│
│  │ 🚀 api-gateway           production · funvest        ││
│  │ 🗄️ postgres-main          production · funvest       ││
│  │ ⚡ worker-emails          staging · funvest           ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  CREATE                                                  │
│  ┌──────────────────────────────────────────────────────┐│
│  │ ➕ Service               Deploy an image or repo     ││
│  │ 🗄️ Database               Provision PostgreSQL...    ││
│  │ 🌐 Environment            Staging, QA, Prod...        ││
│  │ ⏰ Cron Job               Scheduled task              ││
│  │ ⚙️  Worker                 Background process         ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  NAVIGATE                                                │
│  ┌──────────────────────────────────────────────────────┐│
│  │ 📁 funvest               Project                     ││
│  │ 📊 Monitoring             Dashboard                   ││
│  │ 🔑 API Keys               Security                    ││
│  └──────────────────────────────────────────────────────┘│
│                                                          │
│  TIP: Type ">" for commands, "@" for projects,           │
│  "#" for templates                                       │
└──────────────────────────────────────────────────────────┘
```

## 3.4 Modos de Busca

| Prefixo | Modo | Exemplo |
|---|---|---|
| (nenhum) | Global: search + create + navigate | `postgres` → mostra databases, templates, serviços |
| `>` | Commands / Actions | `> deploy api` → dispara deploy do serviço "api" |
| `@` | Filter by project | `@funvest` → escopo reduzido ao projeto "funvest" |
| `#` | Marketplace / Templates | `#wordpress` → busca templates com "wordpress" |

## 3.5 Ranking e Ordenação

- **Recentes**: últimos 5 itens criados ou acessados (por timestamp, local)
- **Criação**: ordenado por relevância (fuzzy match no nome + categoria)
- **Navegação**: ordenado por frequência de acesso (contador interno)
- **Templates**: trending (installs recentes) + featured (curadoria)

## 3.6 Comportamento

- Ao selecionar uma ação de criação → abre o modal correspondente (Create Service, Database, etc.)
- Ao selecionar navegação → navega diretamente (router push)
- Ao selecionar comando → executa a ação com feedback via toast
- `Escape` fecha o Launcher
- Clicar fora fecha o Launcher
- O campo de busca recebe foco automaticamente ao abrir
- Ao fechar e reabrir, o texto da busca anterior é mantido (limpo após 30s de inatividade)

## 3.7 Acessibilidade

- Navegação completa por teclado: `↑↓` para navegar itens, `Enter` para selecionar
- `Tab` navega entre seções (Recentes → Create → Navigate)
- Anunciado como "dialog" para leitores de tela
- Resultados anunciados conforme a busca é digitada (aria-live polite)

---

# 4. Application Wizard

## 4.1 Visão Geral

Para serviços do tipo **Application**, a plataforma oferece um assistente de criação que
detecta automaticamente o framework e pré-configura recursos, build method e variáveis.

O Wizard é ativado quando o source é "Git Repository" — para "OCI Image" o fluxo é o modal
simples (imagem + porta).

## 4.2 Step 1: Connect Repository

```
┌─────────────────────────────────────────────────┐
│  Create Service — Step 1 of 4              ◎◎◎◎ │
├─────────────────────────────────────────────────┤
│                                                 │
│  Connect a Git Repository                       │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │ 🔗 Paste Git URL                         │  │
│  │ └────────────────────────────────────────┘  │
│                                                 │
│  Or connect a provider:                         │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐         │
│  │🐙    │ │🦊    │ │🔷    │ │🟦    │         │
│  │GitHub│ │GitLab│ │Azure │ │Bit-  │         │
│  │      │ │      │ │DevOps│ │bucket│         │
│  └──────┘ └──────┘ └──────┘ └──────┘         │
│                                                 │
│  After connecting, select a repository:         │
│  ┌──────────────────────────────────────────┐  │
│  │ 🔍 Search repositories...                │  │
│  │ ─────────────────────────────────────── │  │
│  │ 📁 org/backend-api         main · 2d ago │  │
│  │ 📁 org/frontend-web        main · 5d ago │  │
│  │ 📁 org/worker-emails       main · 1w ago │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│                       [Cancel]    [Continue →]  │
└─────────────────────────────────────────────────┘
```

**Detalhes de implementação:**
- Integração OAuth com GitHub/GitLab/Bitbucket para listar repositórios.
- Sem OAuth: o usuário cola a URL SSH/HTTPS e configura chave SSH ou credenciais.
- Branch default: `main` (detectado do repositório; editável).

## 4.3 Step 2: Framework Detection

Após conectar o repositório, a plataforma **clona superficialmente** (shallow clone, depth=1)
e executa a detecção automática:

```
┌─────────────────────────────────────────────────┐
│  Create Service — Step 2 of 4              ◎◎◎◎ │
├─────────────────────────────────────────────────┤
│                                                 │
│  Analyzing repository...                    ⏳  │
│  ┌──────────────────────────────────────────┐  │
│  │ ✓ Detected: Node.js 20.x                 │  │
│  │ ✓ Framework: Next.js 14 (App Router)      │  │
│  │ ✓ Package manager: pnpm                  │  │
│  │ ✓ Build command: next build              │  │
│  │ ✓ Output directory: .next/standalone     │  │
│  │ ✓ Start command: node server.js          │  │
│  │ ✓ Port: 3000 (detected from next.config) │  │
│  │ 💡 Template: nextjs-standalone available  │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Detected configuration. Adjust if needed:      │
│  Framework                                      │
│  ┌──────────────────────────────────────────┐  │
│  │ Next.js                          ▼       │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Build method                                   │
│  ┌──────────────────────────────────────────┐  │
│  │ Nixpacks (auto-detect)          ◉        │  │
│  │ Dockerfile                       ○        │  │
│  │ Buildpacks                       ○        │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│                       [Back]     [Continue →]   │
└─────────────────────────────────────────────────┘
```

**Engine de detecção:**
| Indicador | Framework | Build | Port |
|---|---|---|---|
| `package.json` + `next` dep | Next.js | `next build` | 3000 |
| `package.json` + `react` + `vite` | Vite React | `vite build` | 5173 (dev) / 80 (prod) |
| `go.mod` | Go | `go build` | 8080 |
| `Cargo.toml` | Rust | `cargo build --release` | 8080 |
| `requirements.txt` / `pyproject.toml` | Python | `pip install -r requirements.txt` | 8000 |
| `Gemfile` | Rails | `bundle exec rails s` | 3000 |
| `mix.exs` | Elixir/Phoenix | `mix phx.server` | 4000 |
| `build.gradle` / `pom.xml` | Java/Spring | `./mvnw package` | 8080 |
| `deno.json` / `deno.lock` | Deno | `deno task build` | 8000 |
| `bun.lock` | Bun | `bun run build` | 3000 |

**Fallback:** se nenhum framework for detectado, o usuário escolhe manualmente ou usa "Generic Dockerfile".

## 4.4 Step 3: Configure Resources

```
┌─────────────────────────────────────────────────┐
│  Create Service — Step 3 of 4              ◎◎◎◎ │
├─────────────────────────────────────────────────┤
│                                                 │
│  Resources                                      │
│                                                 │
│  CPU                                            │
│  ┌──────────────────────────────────────────┐  │
│  │ ●────────────○────────────○──── 1.0 core │  │
│  │ 0.25      1.0       2.0        4.0       │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Memory                                         │
│  ┌──────────────────────────────────────────┐  │
│  │ ●────────────○────────────○──── 512 MiB  │  │
│  │ 128       512       1GiB      2GiB       │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  💡 Recommended for Next.js: 1 core, 512 MiB    │
│     (based on framework detection)              │
│                                                 │
│  ☐ Enable autoscaling (min 1, max 4 replicas)  │
│  ☐ GPU (NVIDIA CUDA)                            │
│                                                 │
│                       [Back]     [Continue →]   │
└─────────────────────────────────────────────────┘
```

## 4.5 Step 4: Review

```
┌─────────────────────────────────────────────────┐
│  Create Service — Step 4 of 4              ◎◎◎◎ │
├─────────────────────────────────────────────────┤
│                                                 │
│  Review & Deploy                                │
│                                                 │
│  ┌──────────────────────────────────────────┐  │
│  │ 📋 Summary                               │  │
│  │                                          │  │
│  │ Repository   github.com/org/backend-api  │  │
│  │ Branch       main                        │  │
│  │ Framework    Next.js 14                  │  │
│  │ Build        nixpacks                    │  │
│  │ Port         3000                        │  │
│  │ CPU          1 core                      │  │
│  │ Memory       512 MiB                     │  │
│  │ Healthcheck  GET / (interval 5s)         │  │
│  └──────────────────────────────────────────┘  │
│                                                 │
│  Environment Variables (0 configured)           │
│  [+ Add variable]                               │
│                                                 │
│  Domain (optional)                              │
│  [+ Add domain]                                 │
│                                                 │
│              [Back]    [Create & Deploy 🚀]     │
└─────────────────────────────────────────────────┘
```

---

# 5. Database Wizard

## 5.1 Visão Geral

Criar um Database deve ser tão simples quanto criar um Application — mas com UX adaptada
para o domínio de bancos de dados. O usuário escolhe a engine e versão; a plataforma
providencia o container, volume persistente, healthcheck e connection string.

## 5.2 Categorias de Database

| Categoria | Engines |
|---|---|
| **SQL** | PostgreSQL, MySQL, MariaDB, SQL Server, Oracle |
| **NoSQL** | MongoDB, CouchDB, Cassandra, ScyllaDB |
| **Cache** | Redis, Valkey, Dragonfly, Memcached |
| **Message Queue** | RabbitMQ, NATS, Mosquitto (MQTT) |
| **Streaming** | Kafka (Redpanda), Pulsar |
| **Analytics** | ClickHouse, DuckDB, TimescaleDB |
| **Vector Database** | Qdrant, Weaviate, Milvus, pgvector (PostgreSQL) |
| **Object Storage** | MinIO, SeaweedFS, Garage |
| **Search** | Meilisearch, Typesense, Elasticsearch, OpenSearch |
| **Time Series** | InfluxDB, VictoriaMetrics, M3 |

## 5.3 Fluxo de Criação

### Step 1: Choose Engine

Grid de cards com ícone, nome e descrição curta. Agrupado por categoria (tabs laterais ou
sidebar de categorias). Campo de busca no topo para filtrar.

```
┌─────────────────────────────────────────────────┐
│  Create Database                           ✕    │
├─────────────────────────────────────────────────┤
│  ┌────┐ ┌────┐ ┌────────┐ ┌──────┐ ┌─────┐   │
│  │SQL │ │NoSQL│ │Cache   │ │Queue │ │...+│   │
│  └────┘ └────┘ └────────┘ └──────┘ └─────┘   │
│                                                 │
│  🔍 Search engines...                           │
│                                                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │🐘        │ │🐬        │ │🦭        │       │
│  │PostgreSQL│ │MySQL     │ │MariaDB   │       │
│  │16,15,14  │ │8.4,8.0   │ │11,10     │       │
│  │🔄 2.4K   │ │🔄 1.8K   │ │🔄 890    │       │
│  └──────────┘ └──────────┘ └──────────┘       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │🔷        │ │🔶        │ │          │       │
│  │SQL Server│ │Oracle    │ │          │       │
│  │2022      │ │23        │ │          │       │
│  │🔄 320    │ │🔄 150    │ │          │       │
│  └──────────┘ └──────────┘ └──────────┘       │
└─────────────────────────────────────────────────┘
```

O número `🔄 2.4K` indica quantas instâncias dessa engine existem na plataforma (social proof).

### Step 2: Configure

- **Version**: dropdown com versões disponíveis (ex: PostgreSQL 16, 15, 14)
- **Resources**: CPU/Memory slider com recomendação baseada na engine
- **Storage**: tamanho do volume (slider 1GB–1TB)
- **Database name**: default = nome do serviço
- **User**: default = `aether` (ou customizado)
- **Password**: auto-gerado (forte) ou customizado
- **Advanced**: parâmetros de config (postgresql.conf, my.cnf) em editor de texto

### Step 3: Provision

Após criar, a tela mostra o progresso do provisionamento:

```
┌─────────────────────────────────────────────────┐
│  Provisioning postgres-main...             ⏳   │
├─────────────────────────────────────────────────┤
│  ✓ Pulling postgres:16 (125MB)                  │
│  ✓ Creating volume (10GB)                       │
│  ⏳ Starting container...                        │
│  ○ Running healthcheck (pg_isready)             │
│  ○ Database ready                               │
│                                                 │
│  Estimated time remaining: ~15s                 │
└─────────────────────────────────────────────────┘
```

## 5.4 UX Específica por Engine

**PostgreSQL:**
- Conexão: `postgres://user:pass@host:5432/db`
- Backup: `pg_dump -Fc`
- Extensões populares sugeridas: `pgvector`, `postgis`, `uuid-ossp`, `pg_stat_statements`
- Streaming replication: toggle com número de replicas (se multi-node)

**Redis:**
- Conexão: `redis://:pass@host:6379/0`
- Modo: standalone / sentinel / cluster
- Política de evicção: `allkeys-lru`, `volatile-lru`, `noeviction`
- Persistência: RDB, AOF, ambos

**MongoDB:**
- Conexão: `mongodb://user:pass@host:27017/db`
- Replica set: toggle (mínimo 3 nodes)
- Sharding: configuração avançada

---

# 6. Docker Compose Wizard

## 6.1 Visão Geral

Para serviços do tipo **Compose**, a plataforma oferece um editor completo com syntax highlighting,
autocomplete, validação em tempo real e preview visual.

## 6.2 Editor Monaco

- **Syntax highlighting**: YAML com cores do tema escuro da plataforma
- **Autocomplete**: serviços, imagens, portas, volumes, networks, environment
- **Error lens**: erros de sintaxe e validação inline (squiggly lines)
- **Minimap**: overview do documento (opcional, toggle)
- **Line numbers**: absolutos e relativos
- **Bracket matching**: colorido
- **Multi-cursor**: `Cmd+D` para selecionar próximas ocorrências
- **Find/Replace**: `Cmd+F` / `Cmd+H`
- **Go to line**: `Cmd+G`

## 6.3 Validação em Tempo Real

Enquanto o usuário edita, o painel lateral mostra:

```
┌──────────────────────────────────────────┐
│  Compose Analysis                        │
│                                          │
│  Services: 3                             │
│  ┌────────────────────────────────────┐  │
│  │ ✅ api        nginx:alpine  :80    │  │
│  │ ✅ db         postgres:16   :5432  │  │
│  │ ⚠️ redis      redis:7       :6379 │  │
│  │   └ no healthcheck defined         │  │
│  └────────────────────────────────────┘  │
│                                          │
│  Volumes:  2                             │
│  Networks: 1 (aether-app)               │
│                                          │
│  ⚠️ 1 warning                            │
│                                          │
│  Dependency Graph                        │
│  ┌────────────────────────────────────┐  │
│  │         ┌───────┐                  │  │
│  │         │  api  │                  │  │
│  │         └──┬──┬─┘                  │  │
│  │           │  │                     │  │
│  │     ┌─────┘  └─────┐              │  │
│  │  ┌──┴──┐        ┌──┴──┐           │  │
│  │  │ db  │        │redis│           │  │
│  │  └─────┘        └─────┘           │  │
│  └────────────────────────────────────┘  │
│                                          │
│  Interpolation Preview                   │
│  ┌────────────────────────────────────┐  │
│  │ DATABASE_URL → postgres://...      │  │
│  │ REDIS_URL    → redis://...         │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

---

# 7. Marketplace

## 7.1 Arquitetura do Catálogo

O Marketplace é um **catálogo curado de templates** que podem ser instalados com um clique.
Cada template define: imagem(ens), portas, volumes, variáveis de ambiente, recursos recomendados,
e documentação.

**Arquitetura de dados:**
```go
type Template struct {
    ID          string
    Name        string
    Description string
    Category    string      // CMS, Database, Monitoring...
    Tags        []string    // ["postgres", "sql", "relational"]
    Icon        string      // URL ou emoji
    Image       string      // Docker image
    Ports       []string    // ["80:80", "443:443"]
    Volumes     []VolumeDef
    EnvVars     []EnvDef    // defaults + descrição
    Resources   ResourceDef // CPU/Memory recomendados
    ComposeYAML string      // para stacks multi-service
    Readme      string      // markdown
    Homepage    string
    GitHub      string
    License     string
    Version     string
    Installs    int         // contador de instalações (social proof)
    Featured    bool
    Verified    bool        // curadoria oficial vs comunidade
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

## 7.2 Curadoria

**Três níveis de qualidade:**
- **Verified** (✅): mantido oficialmente pela equipe Aether; testado em CI; garantia de funcionamento.
- **Community** (👥): enviado pela comunidade; revisão básica de segurança (não contém malware); funcionalidade não garantida.
- **Unverified** (⚠️): auto-publicado; use por conta própria.

## 7.3 Experiência de Navegação

- **Featured**: 4-6 templates em destaque no topo (carrossel horizontal)
- **Trending**: ordenado por instalações nos últimos 7 dias
- **Editor's Choice**: seleção manual da equipe
- **Categories**: grid de categorias com contagem de templates
- **Search**: busca por nome, descrição, tags, categoria
- **Filters**: verified/community, categoria, ordenação (popular, recente, nome)

## 7.4 Página do Template

- **Hero**: ícone + nome + badge verified/community + botão Install
- **Gallery**: screenshots (carrossel)
- **Description**: readme renderizado (markdown)
- **Details**: versão, license, homepage, GitHub, maintainer
- **Stats**: instalações totais, avaliação (estrelas)
- **Related**: templates relacionados (mesma categoria ou tags)
- **Versions**: changelog + histórico de releases

## 7.5 Instalação

1. Clique "Install" → modal de configuração (pré-preenchido com defaults do template).
2. Revisão dos resources, ports, volumes, env vars.
3. Confirmação → provisionamento.
4. Redirecionamento para a página do serviço recém-criado.

---

# 8. Catálogo de Templates

## 8.1 Coleção Completa (Categorias e Exemplos)

### CMS
WordPress, Strapi, Ghost, Directus, Payload, Apostrophe, KeystoneJS, Sanity, Contentful, Drupal, Joomla, Hugo, Gatsby, Next.js (static), Astro, Eleventy

### ERP / CRM
Odoo, ERPNext, SuiteCRM, EspoCRM, Dolibarr, Twenty, Monica, InvoiceNinja, Crater, Akaunting

### Wiki / Knowledge Base
Wiki.js, Outline, BookStack, DokuWiki, MediaWiki, TiddlyWiki, Confluence (DC), Notion (open alt: AppFlowy), SiYuan, Trilium, Logseq, Obsidian sync server

### Monitoring
Grafana, Prometheus, VictoriaMetrics, Mimir, Zabbix, Nagios, CheckMK, Uptime Kuma, Statping, BetterUptime, Upptime, Glances, Netdata, Datadog agent

### Observability
OpenTelemetry Collector, Jaeger, SigNoz, HyperDX, Grafana Tempo, Loki, Fluentd, Fluent Bit, Vector, Quickwit, OpenObserve

### Logging
Graylog, ELK Stack (Elasticsearch + Logstash + Kibana), Grafana Loki, Seq, Papertrail, Loggly (alternatives), syslog-ng, rsyslog

### Analytics
Metabase, Superset, Redash, Lightdash, Cube.js, Plausible, Umami, Matomo, PostHog, Mixpanel (alt), Fathom, GoatCounter, Tinybird, ClickHouse, DuckDB

### Developer Tools
GitLab, Gitea, Gogs, OneDev, Forgejo, Sourcegraph, OpenGrok, Code-server (VSCode), JupyterHub, JupyterLab, RStudio Server, Eclipse Theia

### AI / LLM / ML
Ollama, Open WebUI, LocalAI, vLLM, text-generation-webui, LangFlow, Flowise, Dify, AnythingLLM, Open Interpreter, PrivateGPT, Hugging Face TGI, LlamaIndex, LangChain, Weaviate, Qdrant, Milvus, Chroma, pgvector, LanceDB

### Databases
PostgreSQL, MySQL, MariaDB, SQL Server, Oracle, MongoDB, CouchDB, Cassandra, ScyllaDB, CockroachDB, YugabyteDB, TiDB, PlanetScale (Vitess), RethinkDB, ArangoDB, Neo4j, Dgraph, SurrealDB

### Queues / Streaming
RabbitMQ, NATS, Mosquitto (MQTT), Apache Kafka, Redpanda, Apache Pulsar, EMQX, VerneMQ, Memphis, LavinMQ

### Search
Elasticsearch, OpenSearch, Meilisearch, Typesense, ZincSearch, Sonic, Manticore Search, Apache Solr

### Object Storage
MinIO, SeaweedFS, Garage, Ceph (RGW), MinIO Gateway

### Email
Postal, Mailcow, Mailu, MailHog, Mailpit, Mautic, Listmonk, Sendy

### DNS
Pi-hole, AdGuard Home, Bind9, CoreDNS, PowerDNS, Technitium, dnsmasq, Unbound

### Reverse Proxy
Traefik, Caddy, Nginx, HAProxy, Envoy, Apache httpd, Tengine, Pound

### Media / Streaming
Jellyfin, Plex, Emby, Kavita, Komga, Calibre-web, Audiobookshelf, Navidrome, Icecast, Owncast, PeerTube, LiveKit, Janus, Jitsi

### Automation
n8n, Node-RED, Huginn, Activepieces, Temporal, Cadence, Windmill, Kestra, Prefect, Dagster, Airflow, Argo Workflows

### Networking
WireGuard (wg-easy), Tailscale (headscale), NetBird, Netmaker, Nebula, ZeroTier, OpenVPN, Pritunl, StrongSwan, FRP, Ngrok (alt), Cloudflare Tunnel

### Identity / Security
Authentik, Keycloak, Authelia, Zitadel, Ory (Hydra/Kratos/Keto), Casdoor, SuperTokens, Hanko, Cerbos, OPAL (authorization), Vault, Teleport, Pomerium, oauth2-proxy

### Finance / Invoicing
InvoiceNinja, Crater, Akaunting, Firefly III, Kill Bill, Ghostfolio, Maybe

### IoT
Home Assistant, Node-RED, ThingsBoard, Mainflux, Eclipse Hono, EMQX, Thingspeak, openHAB, Zigbee2MQTT, Zwave2Mqtt

### Backup
Restic, Borg, Duplicati, Kopia, Velero (K8s), Rsync, Rclone, BorgBase, Duplicacy, UrBackup, Bareos, Amanda

### Productivity
Nextcloud, ownCloud, Seafile, Syncthing, FileBrowser, Paperless-NGX, DocuSeal, Stirling PDF, IT-Tools, Dashy, Homepage, Flame, Homarr, Organizr, Fenrus

### Home Lab
Portainer, Yacht, Cockpit, CasaOS, Umbrel, StartOS, Runtipi, Cosmos, HomelabOS, Dockge

### Messaging
Mattermost, Rocket.Chat, Matrix (Synapse/Dendrite), Element, Zulip, Revolt, SimpleX, Signal (proxy), Telegram (alt), WhatsApp (alt)

### Video
PeerTube, Owncast, LiveKit, Janus, Jitsi, BigBlueButton, Galene, Mediasoup, OvenMediaEngine, MistServer

### Storage
Nextcloud, ownCloud, Seafile, FileBrowser, SeaweedFS, MinIO, Garage, Ceph, GlusterFS, Longhorn (K8s), OpenEBS, Rook

### Blog / Headless CMS
Ghost, Strapi, WordPress, Hugo, Gatsby, Astro, Eleventy, Jekyll, Hexo, Publii, WriteFreely, Plume, Mataroa

### Static Hosting
Nginx (static), Caddy (static), serve, thttpd, lighttpd, darkhttpd, miniserve

### Frameworks
Next.js, Nuxt, SvelteKit, Remix, Astro, RedwoodJS, Blitz.js, AdonisJS, NestJS, Fastify, Hono, Express, Koa, Flask, Django, FastAPI, Rails, Laravel, Phoenix, Gin, Echo, Fiber, Rocket, Actix

### Git
GitLab, Gitea, Forgejo, OneDev, Gogs, Soft Serve, GitDaemon, git-webui

### CI/CD
Jenkins, Drone, Woodpecker, Concourse, GoCD, Tekton, ArgoCD, Flux, Spinnaker, Buildbot, Agola

---

# 9. Tela de Detalhes do Service

## 9.1 Estrutura Geral

```
┌────────────────────────────────────────────────────────────┐
│  🕸️ api-gateway  ● Running                     Actions ▼  │
│  node:20-alpine · port 3000 · funvest / production         │
├────────────────────────────────────────────────────────────┤
│  [Overview] [Deployments] [Logs] [Metrics] [Variables]    │
│  [Settings] [Cron] [Workers] [Terminal]                   │
├────────────────────────────────────────────────────────────┤
│                                                            │
│  (conteúdo da tab ativa)                                   │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Header:**
- Ícone (por tipo: webhook para web, code para git, database para DB, schedule para cron)
- Nome do serviço (headline-sm)
- Badge de status (Running · Stopped · Deploying · Failed · Degraded)
- Source (imagem ou git URL) e porta
- Breadcrumb implícito: Project · Environment
- Botões de ação (Start/Stop, Restart, Redeploy, Rebuild, Visit URL)
- Actions menu (Settings, Delete, Duplicate)

## 9.2 Abas

### Overview (default)
Resumo executivo do serviço:
- **Service Details** card: Live URL (com copy), Source, Type, Port, Environment, Project
- **CPU / Memory** mini gauges (atualizados a cada 5s)
- **Latest Deployment** card: commit, status, timestamp, "View Logs"
- **Live Logs** terminal (últimas 20 linhas, segue em tempo real, terminal escuro)

### Deployments
- Timeline vertical de deployments (#1, #2, #3...)
- Cada item: commit (8 chars), status badge, timestamp relativo (2m ago), expandir para ver log
- Rollback button por deployment (confirmação)
- Diff entre deployments (comparar #2 com #1: imagem, env vars, resources)
- Filtro por status (All, Ready, Failed, Building)

### Logs
Ver Capítulo 11.

### Metrics
Ver Capítulo 16.

### Variables
Ver Capítulo 12.

### Settings
- General: nome, source type, image, git URL, branch, dockerfile path, build method, port
- Resources: CPU, Memory (sliders), GPU toggle
- Health Check: path, interval, timeout, retries, enable/disable
- Volumes: lista, adicionar, remover
- Autopilot: policy card (igual ao existente)
- Webhook: configuração de secret
- Danger Zone: Delete service (com aviso de cascade)

### Cron / Workers / Terminal
Componentes existentes (já implementados).

---

# 10. Deployments

## 10.1 Timeline

Timeline vertical com:
- **Status icon**: ✅ (ready), ❌ (failed), ⏳ (building), 🟡 (queued)
- **Commit hash** (8 chars, clicável → link para o commit no GitHub/GitLab)
- **Trigger**: webhook, manual, rollback, rebuild
- **Timestamp relativo** (2 minutes ago)
- **Expand**: mostra log completo do build/deploy
- **Rollback button**: por deployment (com confirmação)

## 10.2 Rollback

- Exibe o diff do que vai ser restaurado (imagem, env vars, resources)
- Confirmação com modal
- Rollback cria um NOVO deployment (#N+1) com a imagem/env do deployment anterior — NÃO altera o histórico
- Timeline mostra o rollback como um deployment normal com trigger "rollback"

## 10.3 Deploy Comparison

Selecionar dois deployments (checkboxes) → botão "Compare":
- Imagem: antes vs depois
- Env vars: adicionadas, removidas, alteradas (diff visual)
- Resources: CPU/Memory antes vs depois
- Commit log: lista de commits entre os dois (se git)

## 10.4 Failure Analysis

Quando um deploy falha, o card de erro mostra:
- **Mensagem de erro** (do docker/build/runtime)
- **Sugestão automática**: análise heurística do erro:
  - "port already in use" → sugere mudar porta ou parar serviço conflitante
  - "out of memory" → sugere aumentar limite de memória
  - "image not found" → sugere verificar nome e registry
  - "build failed" → link para a linha do erro no log de build
- **Retry button**: re-executa o deploy com os mesmos parâmetros

---

# 11. Logs

## 11.1 Visão Geral

Sistema de logs em tempo real com:
- **Live Tail**: stream via SSE (Server-Sent Events), segue o fim automaticamente
- **Pause/Resume**: congela o scroll para inspeção; botão "Follow" retoma
- **Download**: exporta as últimas N linhas como .txt ou .json
- **Clear**: limpa o buffer visual (não afeta o arquivo de log)

## 11.2 Formatação

- **ANSI Colors**: suporte completo a escape codes (cores, bold, underline)
- **JSON Detection**: linhas que começam com `{` ou `[` são detectadas como JSON e renderizadas com syntax highlighting
- **Timestamps**: coluna de timestamp fixa à esquerda (formatos detectados automaticamente: ISO 8601, Unix, custom)
- **Level badges**: `[INFO]` verde, `[WARN]` amarelo, `[ERROR]` vermelho, `[DEBUG]` cinza, `[HTTP]` azul

## 11.3 Filtros e Busca

### Barra de Filtros
- **Search**: regex ou texto literal (toggle)
- **Level**: dropdown multi-select (INFO, WARN, ERROR, DEBUG, HTTP)
- **Service**: para logs agregados de múltiplos serviços? Não neste contexto (log de UM serviço)
- **Date range**: desde/até (datetime picker)
- **Bookmarks**: salvar buscas frequentes ("erros nas últimas 24h")

### Highlight
- Termos da busca destacados em amarelo no texto do log
- Erros destacados com background vermelho sutil
- Linhas com match da busca têm indicador na scrollbar (minimap)

## 11.4 Performance

- **Virtual scrolling**: renderiza apenas as linhas visíveis (milhões de linhas sem lag)
- **Buffer ring**: mantém as últimas 10.000 linhas em memória; busca no arquivo para histórico mais antigo
- **Infinite scroll para cima**: carrega chunks de 1.000 linhas do arquivo sob demanda

---

# 12. Variables

## 12.1 Hierarquia

```
System (runtime defaults: PORT, AETHER_APP_ID, ...)
  └── Project Variables (disponíveis para todos os environments)
       └── Environment Variables (disponíveis para todos os serviços do environment)
            └── Service Variables (específicas do serviço)
```

**Precedência:** Service > Environment > Project > System.
Variáveis com o mesmo nome no nível mais específico **sobrepõem** as dos níveis acima.

## 12.2 Interpolação

Sintaxe suportada no valor das variáveis:

| Sintaxe | Resolve para | Exemplo |
|---|---|---|
| `${{environment.KEY}}` | Variável do environment atual | `${{environment.API_URL}}` → `https://api.example.com` |
| `${{project.KEY}}` | Variável do projeto | `${{project.SMTP_HOST}}` → `smtp.global.com` |
| `${{service.KEY}}` | Variável do próprio serviço | `${{service.PORT}}` → `3000` |
| `${{secret.KEY}}` | Secret do vault (futuro) | `${{secret.STRIPE_KEY}}` → `sk_live_xxx` |

A resolução é **recursiva em 1 nível** (evita loops infinitos). Se `A=${{environment.B}}` e `B=${{environment.C}}`, a resolução para em `B` (não segue para `C`).

## 12.3 Editor

- **Editor de texto** (não tabela): textarea grande com syntax KEY=value
- **Syntax highlighting**: chave em azul, valor em branco, comentários em cinza
- **Validação inline**: linhas inválidas marcadas com ⚠️ e mensagem de erro
- **Duplicate detection**: chaves duplicadas detectadas e destacadas
- **Secrets masking**: valores de secrets mostrados como `••••••••••`; toggle "Reveal secrets" mostra temporariamente
- **Auto-save**: salva automaticamente após 2s de inatividade (com indicador "Saved" ✓)
- **Undo**: `Cmd+Z` funciona no editor
- **Export/Import**: botões para download `.env` e upload de arquivo

## 12.4 Secrets e Criptografia

- Secrets são **criptografados em repouso** (AES-256-GCM com KEK/DEK)
- Na API, secrets são mascarados (`••••••••••`) a menos que explicitamente revelados (`?secrets=1` com auth)
- No runtime do container, secrets são descriptografados e injetados como variáveis de ambiente normais
- Rotação de chaves: preparada na arquitetura (KEK/DEK separados), interface futura
- Auditoria: toda alteração em secrets é registrada (quem, quando, qual chave — sem valor)

## 12.5 Auditoria

Tabela `variable_audit`:
- Quem (usuário)
- O quê (action: set, delete, bulk_replace)
- Qual chave
- Valor anterior (criptografado, para secrets)
- Timestamp
- Acessível via aba "Audit" no modal de variáveis

---

# 13. Networking

## 13.1 Domains

- **Add Domain**: host (ex: `api.example.com`), HTTPS toggle
- **Certificate**: status (pending, issued, failed, renewing), emissor (Let's Encrypt), expiração
- **Wildcard**: `*.example.com` (requer DNS challenge)
- **Auto-renew**: 30 dias antes da expiração

## 13.2 TLS

- **Provider**: Let's Encrypt (ACME) — default; custom certificate upload (PEM)
- **Challenge**: HTTP-01 (padrão), DNS-01 (para wildcards)
- **Minimum TLS**: 1.2 (default), configurável para 1.3 only

## 13.3 Ingress / Load Balancer

- **Internal**: tráfego entre serviços no mesmo projeto (rede `aether-<project>`)
- **External**: domínio público com TLS (via Traefik)
- **Port mapping**: container port → host port (auto ou manual)

## 13.4 Service Discovery

- **DNS interno**: `aether-db-postgres` resolve para o IP do container no Docker network
- **Environment variable**: connection strings injetadas automaticamente para databases vinculados
- **Future**: integração com Consul/etcd para service mesh

---

# 14. Resources

## 14.1 CPU e Memória

- **CPU**: slider de 0.25 a 64 cores (escala logarítmica para valores baixos)
- **Memory**: slider de 128MB a 256GB (escala logarítmica)
- **GPU**: toggle NVIDIA CUDA (requer node com GPU)
- **Recomendações automáticas**: baseadas na detecção de framework e uso histórico

## 14.2 Autoscaling (Horizontal)

- **Min/Max replicas**: slider
- **Scale up trigger**: CPU > N% por M minutos
- **Scale down trigger**: CPU < N% por M minutos
- **Cooldown**: intervalo mínimo entre scaling events
- **Preview**: gráfico mostrando comportamento esperado

## 14.3 Scheduling / Affinity

- **Node affinity**: labels (ex: `gpu=true`, `region=us-east`)
- **Cluster affinity**: associar serviço a um cluster específico
- **Anti-affinity**: evitar co-localizar réplicas no mesmo node

---

# 15. Volumes

## 15.1 Persistent Volumes

- **Criar volume**: nome + tamanho (1GB–10TB) + driver (local, NFS, S3-backed)
- **Montar**: container path (ex: `/data`, `/var/lib/postgresql/data`)
- **Modo**: read-write (default), read-only
- **Lista**: volumes do serviço com uso atual (ex: 2.3GB de 10GB)

## 15.2 Snapshots

- **Criar snapshot**: nome + descrição; instantâneo do volume
- **Restaurar**: volta o volume para o estado do snapshot
- **Agendamento**: snapshot diário/semanal (cron)
- **Dedup**: content-addressed chunking (zstd, 1MiB chunks, SHA-256) — implementado na F4
- **Reter**: política de retenção (últimos N snapshots)

## 15.3 Backup / Clone / Migração

- **Backup**: export do volume como archive (tar.gz) para S3 ou storage local
- **Clone**: duplicar volume para outro serviço
- **Migração**: mover volume entre nodes/servers

---

# 16. Monitoring

## 16.1 Métricas

- **CPU**: percentual de uso (user, system, iowait)
- **Memory**: RSS, cache, swap
- **Disk**: leitura/escrita (IOPS, throughput)
- **Network**: RX/TX bytes, pacotes, erros
- **Latency**: p50, p95, p99 (do healthcheck probe)

## 16.2 Visualização

- **Gauges**: cards com valor atual + barra de progresso (CPU 42%, Memory 256MB/1GB)
- **Sparklines**: mini gráfico de linha nas últimas N amostras
- **Time range**: 5m, 15m, 1h, 6h, 24h, 7d, 30d
- **Granularidade**: automática (1s para <1h, 1m para <24h, 1h para >24h)

## 16.3 Healthchecks

- **HTTP**: GET no path configurado, esperado 2xx/3xx
- **TCP**: connect na porta
- **Command**: exec no container (ex: `pg_isready`)
- **Timeout**: falha após N segundos
- **Retries**: número de falhas consecutivas antes de marcar unhealthy
- **Auto-restart**: opção de restart automático após N falhas

## 16.4 Alertas

- **Canais**: Slack, Discord, Telegram, Email, Webhook (já implementados)
- **Triggers**: CPU > 90%, Memory > 90%, Disk > 85%, serviço down, deploy failed
- **Silence**: período de silêncio configurável (ex: não alertar durante deploy)

## 16.5 Dashboards

- **Service dashboard**: métricas do serviço individual
- **Environment dashboard**: agregado de todos os serviços do ambiente
- **Project dashboard**: Fleet Overview (já implementado)
- **Custom**: futuramente, dashboards criados pelo usuário com PromQL / Grafana

---

# 17. UI / UX — Design System

## 17.1 Design Tokens

### Colors (Surface Layers)
```
surface.background   #0a0a0f  (mais escuro — profundidade)
surface.section      #101014  (seções dentro da página)
surface.card         #16161c  (cards)
surface.popover      #1e1e26  (dropdowns, menus)
surface.modal        #24242e  (modais, dialogs)
```

### Colors (Semantic)
```
primary              #b0c6ff  (ações principais, links, foco)
primary.container    #568dff  (backgrounds de primary)
error                #ffb4ab  (erros, danger zone)
warning              #fbbf24  (warnings)
success              #4ade80  (ready, healthy)
```

### Elevation / Shadows
```
shadow.none          0
shadow.sm            0 1px 2px rgba(0,0,0,0.3)     (cards sutis)
shadow.md            0 8px 24px rgba(0,0,0,0.4)     (popovers)
shadow.lg            0 20px 60px rgba(0,0,0,0.6)    (modais)
shadow.xl            0 32px 80px rgba(0,0,0,0.7)    (launcher)
```

### Radius
```
radius.sm            0.25rem   (checkboxes, badges)
radius.md            0.5rem    (inputs, buttons, cards)
radius.lg            0.75rem   (modais, popovers)
radius.xl            1rem      (containers grandes)
radius.full          9999px   (pills, avatares)
```

### Spacing
```
spacing.xs           0.25rem (4px)
spacing.sm           0.5rem  (8px)
spacing.md           1rem    (16px)
spacing.lg           1.5rem  (24px)
spacing.xl           2rem    (32px)
spacing.2xl          3rem    (48px)
```

### Typography
```
display-lg           48px / 56px  (-0.02em)  weight 700   (hero titles)
headline-sm          24px / 32px  (-0.01em)  weight 600   (page titles)
body-md              14px / 20px             weight 400   (body text)
body-sm              13px / 18px             weight 400   (captions)
label-caps           11px / 16px  (0.05em)   weight 600   (labels, uppercase)
code-md              13px / 20px             weight 450   (mono: JetBrains Mono)
```

### Motion
```
duration.instant     0ms
duration.fast        150ms   (hover, focus)
duration.normal      200ms   (open/close, transitions)
duration.slow        300ms   (page transitions, modals)
easing.default       cubic-bezier(0.4, 0, 0.2, 1)  (ease-out)
easing.bounce        cubic-bezier(0.34, 1.56, 0.64, 1)  (overshoot sutil)
```

## 17.2 Componentes Shared

### Popover (componente único para toda a plataforma)
```
┌──────────────────────────────┐
│ ← surface.popover bg         │
│ ← border.subtle border       │
│ ← shadow.md                  │
│ ← radius.lg                  │
│ ← p-1.5 (spacing interno)    │
└──────────────────────────────┘
```
**Regra:** todo dropdown, menu contextual, autocomplete e combobox usa este componente.
Nunca criar variações inline com `bg-popover` direto.

### Dropdown Menu Item
```
┌──────────────────────────────┐
│  [icon]  Label               │  ← py-2, px-2.5, rounded-lg
│                              │  ← hover: bg-surface-container-high
│                              │  ← transition: 150ms ease-out
│                              │  ← text-[16px] icon, font-body-sm
└──────────────────────────────┘
```

### Command Palette / Launcher
Ver Capítulo 3. Overlay central com backdrop blur + shadow.xl.

### Skeleton Loading
- **Card**: pulso shimmer (animação de gradiente horizontal, 2s infinite)
- **Text**: barras de largura variável (80%, 60%, 40%)
- **Table**: 5 linhas de skeleton, cada uma com 4 colunas de larguras variadas

### Transitions
- **Modal open**: fade in (200ms) + scale 0.95→1.0
- **Modal close**: fade out (150ms) + scale 1.0→0.95
- **Popover open**: fade in (150ms) + translateY(-4px→0)
- **Page transition**: crossfade 200ms (futuro, com view transitions API)

## 17.3 Accessibility

- **Focus ring**: `focus-visible:ring-2 focus-visible:ring-ring/60` em todos os elementos interativos
- **Keyboard navigation**: Tab, Shift+Tab, Enter, Escape, setas em todos os menus e modais
- **Screen readers**: aria-label em ícones, aria-expanded em dropdowns, role="dialog" em modais
- **Color contrast**: todo texto ≥4.5:1 contra o fundo (WCAG AA)
- **Reduced motion**: `prefers-reduced-motion` → desabilita animações e transições

## 17.4 Responsive

- **Sidebar**: colapsa em drawer com hamburger menu em <1024px
- **Header**: stack vertical em <768px
- **Tabelas**: scroll horizontal em telas pequenas (não tentar comprimir colunas)
- **Modais**: full-screen em mobile (<640px)

---

# 18. Arquitetura Técnica

## 18.1 Visão Geral

```
┌────────────────────────────────────────────────────────┐
│                    Frontend (React)                     │
│  TanStack Router · Query · RHF+Zod · Tailwind v4       │
├────────────────────────────────────────────────────────┤
│                    API (Go)                             │
│  net/http ServeMux · auth JWT · perm RBAC               │
├────────────────────────────────────────────────────────┤
│                    Core (Go)                            │
│  Deploy · DB · Compose · Env · Backup · Notify          │
├────────────────────────────────────────────────────────┤
│                    Runtime (Go)                         │
│  Docker CLI · Podman CLI · K8s REST · Quadlet/systemd   │
├────────────────────────────────────────────────────────┤
│                    Persistence                          │
│  PostgreSQL 15+ (schema per tenant opcional)            │
│  File system (logs, backups, snapshots, certs)          │
└────────────────────────────────────────────────────────┘
```

## 18.2 Entidades (Domínio)

| Entidade | Tabela | Relacionamentos |
|---|---|---|
| Workspace | `orgs` | 1:N `projects` |
| Project | `projects` | 1:N `environments`; 1:N `project_variables` |
| Environment | `environments` | N:1 `projects`; 1:N `apps`; 1:N `env_variables` |
| Service (App) | `apps` | N:1 `environments`; 1:N `deployments`; 1:N `app_env` |
| Deployment | `deployments` | N:1 `apps` |
| Database | `databases` | N:1 `projects` |
| Variable | `project_variables`, `env_variables`, `app_env` | Hierarquia de precedência |

## 18.3 APIs (Principais)

```
POST   /api/v1/projects/:pid/services              Create Service
GET    /api/v1/projects/:pid/services              List Services (by project)
GET    /api/v1/services/:id                        Get Service detail
PATCH  /api/v1/services/:id                        Update Service
DELETE /api/v1/services/:id                        Delete Service (cascade)

POST   /api/v1/services/:id/deploy                 Deploy
POST   /api/v1/services/:id/start                  Start container
POST   /api/v1/services/:id/stop                   Stop container
POST   /api/v1/services/:id/restart                Restart container
POST   /api/v1/services/:id/rollback               Rollback

GET    /api/v1/services/:id/logs                   Logs (SSE)
GET    /api/v1/services/:id/stats                  Stats (CPU/Mem/Net/IO)
GET    /api/v1/services/:id/deployments            Deployments list

GET    /api/v1/services/:id/env                    Service env vars
PUT    /api/v1/services/:id/env                    Bulk replace env
```

## 18.4 Eventos

| Evento | Publicador | Consumidores |
|---|---|---|
| `service.created` | API → Bus | Notificações, Webhooks, Audit |
| `deployment.created` | Core.Deploy | Pipeline (runPipeline) |
| `deployment.ready` | Core (healthcheck OK) | DNS, Proxy, Webhooks |
| `deployment.failed` | Core | Notificações, Webhooks, Alerts |

## 18.5 Cache

- **Variáveis de ambiente**: cache em memória (map por projectID/environmentID), invalidado em writes
- **Sessões**: JWT stateless (sem cache de sessão)
- **Templates do marketplace**: cache em memória, invalidado em deploy/update de templates
- **Futuro**: Redis para cache distribuído multi-instância

## 18.6 Background Jobs

- **Scheduler**: tick 15s (cron jobs, healthchecks, probes)
- **Autopilot**: loop 60s (resource scaling)
- **GitOps watcher**: poll 60s (sync repos)
- **NetQ probes**: poll 30s (latency probes)
- **Server watchdog**: poll 10s (agent health)

## 18.7 Escalabilidade

- **Single binary**: um processo Go contém API + Core + Runtime + Scheduler
- **Read replicas**: PostgreSQL replicas para leitura (futuro, F5)
- **Horizontal scaling**: múltiplas instâncias da API (stateless), com advisory locks para migrations
- **Multi-server**: core + N agents (RFC-0015, implementado v1 para deploys de imagem)

---

# 19. Checklist de Implementação

## Fase 1 — Fundação (em produção)

- [x] **Create Service modal** com campos obrigatórios (source, image/name, port)
  - Accept: modal abre/fecha; campos validados; submit cria o service no banco; deploy dispara
  - Para cada source type: OCI Image (funciona), Git (funciona)
- [ ] **Progressive disclosure** no modal  <!-- PENDENTE: seções advanced sempre visíveis -->
  - Accept: seções avançadas colapsadas por padrão; expandir mostra resources/health/volumes/env
- [ ] **Estado de loading** durante deploy  <!-- PENDENTE: sem skeleton na lista -->
  - Accept: skeleton shimmer na linha do serviço até container iniciar
- [x] **Empty state** para ambientes sem serviços
  - Accept: ilustração + CTA visível
- [x] **Error states** para falhas comuns (image pull, port conflict, OOM)  <!-- via failure analysis + toast -->
  - Accept: toast com mensagem clara + sugestão
- [x] **Launcher básico** (search + create + navigate)
  - Accept: Cmd+Shift+K abre overlay; digitar filtra; Enter executa ação
- [x] **Modos de busca no Launcher**: `>`, `@`, `#`
  - Accept: cada prefixo filtra corretamente
- [x] **Recentes no Launcher**
  - Accept: últimos 5 itens aparecem ao abrir (antes de digitar)

## Fase 2 — Marketplace

- [ ] **Catálogo de templates** (≥100 templates)  <!-- PARCIAL: 79 templates/26 categorias -->
  - Accept: grid com nome, ícone, descrição, badge verified/community; busca funcional
  - Categorias mínimas: CMS, Database, Monitoring, DevTools, AI, Storage, Media, Automation
- [x] **Página do template** (readme, stats, install)  <!-- markdown render (código/bold/headers/listas/quote) + stats + install -->
  - Accept: renderização markdown do readme; botão Install funcional
- [x] **Featured + Trending + Editor's Choice**  <!-- migration 15: editors_choice + 6 curados; endpoint ?editors_choice=true -->
  - Accept: seções na home do marketplace com ordenação correta
- [x] **Instalação com um clique**
  - Accept: modal pré-preenchido com defaults do template; submit cria + provisiona

## Fase 3 — UX Premium

- [x] **Popover unificado** (componente single source of truth)
  - Accept: todo dropdown/menu/context usa o mesmo componente
- [x] **Design tokens completos** (surface layers, elevation, motion)
  - Accept: variáveis CSS aplicadas consistentemente; inspeção visual mostra camadas
- [x] **Animation system**  <!-- modal-pop/fade-in + prefers-reduced-motion -->
  - Accept: transições em modais/popovers/páginas; reduced-motion respeitado
- [~] **Keyboard navigation completa**  <!-- Escape/arrows ok; Tab audit pendente -->
  - Accept: Tab/Enter/Escape/Arrow keys em todos os componentes interativos
- [x] **Accessibility baseline** (WCAG AA)  <!-- role=dialog + aria-modal + aria-live no launcher e modais -->
  - Accept: contraste ≥4.5:1; screen reader anuncia ações; focus visível

## Fase 4 — Wizards

- [~] **Application Wizard** (framework detection)  <!-- detect+fallback ok; sem OAuth GitHub/GitLab -->
  - Accept: conectar repo → detectar framework → sugerir config → review → deploy
- [x] **Database Wizard** (categories UX)  <!-- grid por categoria (Relational/Document/Cache) com seleção visual -->
  - Accept: grid de engines por categoria; provisionamento com progresso
- [~] **Compose Wizard** (Monaco editor + live validation)  <!-- editor custom + validação real; sem Monaco/error lens -->
  - Accept: editor com syntax highlighting; painel de análise; dependency graph

## Fase 5 — Detalhe do Service Premium

- [x] **Timeline de deployments** visual  <!-- tabela #/status/duration/commit + compare select + failure badges -->
  - Accept: timeline vertical com status, commit, timestamp, expandir
- [x] **Deploy comparison** (diff visual entre 2 deployments)
  - Accept: selecionar 2 → mostrar diff de imagem/env/resources
- [x] **Failure analysis** com sugestões automáticas
  - Accept: erro parseado → sugestão contextual
- [~] **Log viewer avançado** (ANSI colors, JSON, regex search, virtual scroll)  <!-- buffer 1500 linhas, sem virtual scroll 100k -->
  - Accept: ANSI renderizado; JSON pretty-print; busca regex funcional; scroll em 100k+ linhas sem lag
- [x] **Variable editor avançado** (auto-save, syntax highlight, duplicate detection)  <!-- auto-save 2s debounce; duplicate detection + secrets reveal já existiam -->
  - Accept: textarea com highlight; salva automático; duplicatas detectadas

## Fase 6 — Enterprise

- [ ] **Autoscaling horizontal** (min/max replicas, CPU triggers)  <!-- pendente -->
  - Accept: slider de replicas; deploy cria N containers; scale up/down conforme CPU
- [x] **Snapshot scheduling** (diário/semanal com retenção)  <!-- migration 16: cron @daily/@weekly/@hourly/5-campos + retenção; scheduler 30s; notificação backup.finished; UI no Storage -->
  - Accept: agenda configurável; snapshots criados automaticamente; antigos removidos
- [x] **Custom dashboards** (Prometheus/Grafana integration)  <!-- /metrics Prometheus pronto -->
  - Accept: prometheus endpoint exportando métricas; dashboards custom no Grafana externo
- [ ] **Backup to S3** (automated, encrypted)  <!-- pendente -->
  - Accept: destino S3 configurável; backups automáticos; restore funcional


---

# 20. Deploy Events e Notificações em Tempo Real

## 20.1 Visão Geral

Toda ação de deploy na plataforma emite um **evento de ciclo de vida** que é consumido
pelo sistema de notificações. O objetivo é que **todos os membros da organização** saibam
o que está acontecendo nos serviços — em tempo real se estiverem logados, ou ao acessar
o centro de notificações quando retornarem.

## 20.2 Ciclo de Vida de um Deploy (Eventos)

Cada deploy passa por estados bem definidos, e cada transição emite um evento:

```
DEPLOY TRIGGER (manual, webhook, rollback, rebuild, GitOps)
  │
  ▼
deployment.queued          ← "Service 'api-gateway' deploy queued · triggered by Neverson"
  │
  ▼
deployment.building        ← "Building 'api-gateway' (nixpacks) · commit a84f9b2"
  │
  ▼
deployment.starting        ← "Starting 'api-gateway' container on node eu-west-1"
  │
  ▼
deployment.healthcheck     ← "Health check 'api-gateway' · GET /health · attempt 3/30"
  │
  ├──✅ deployment.ready    ← "✅ 'api-gateway' deployed successfully · 47s · port 3000"
  │
  └──❌ deployment.failed   ← "❌ 'api-gateway' deploy failed · port 3000 already in use"
```

## 20.3 Estrutura do Evento

```go
type DeployEvent struct {
    ID            string    `json:"id"`
    Type          string    `json:"type"`           // deployment.queued | .building | .starting | .healthcheck | .ready | .failed
    ServiceID     string    `json:"service_id"`
    ServiceName   string    `json:"service_name"`
    EnvironmentID string    `json:"environment_id"`
    EnvironmentName string  `json:"environment_name"`
    ProjectID     string    `json:"project_id"`
    ProjectName   string    `json:"project_name"`
    OrgID         string    `json:"org_id"`
    DeploymentID  string    `json:"deployment_id"`
    DeployNumber  int64     `json:"deploy_number"`
    Commit        string    `json:"commit,omitempty"`       // git only
    Image         string    `json:"image,omitempty"`         // image only
    Trigger       string    `json:"trigger"`                 // manual | webhook | rollback | rebuild | gitops
    TriggeredBy   string    `json:"triggered_by,omitempty"`  // user email (manual) or "github-webhook", "gitops"
    DurationMs    int64     `json:"duration_ms,omitempty"`   // ready/failed only
    Error         string    `json:"error,omitempty"`         // failed only
    Timestamp     time.Time `json:"timestamp"`
}
```

**Regras de atribuição de `triggered_by`:**
- Deploy manual (UI ou CLI): email do usuário autenticado que disparou → "neversonbs13@gmail.com"
- Deploy por webhook (GitHub/GitLab/Bitbucket): "github-webhook", "gitlab-webhook", "bitbucket-webhook"
- Rollback: email do usuário que iniciou o rollback
- GitOps (auto-apply): "gitops"
- Rebuild: email do usuário

## 20.4 Comportamento da Notificação

### Formato da Mensagem (Resumo Humano)

Cada evento gera uma mensagem legível:

| Evento | Mensagem |
|---|---|
| `deployment.queued` | **api-gateway** deploy queued · triggered by Neverson |
| `deployment.building` | Building **api-gateway** (#3) · nixpacks · commit a84f9b2 |
| `deployment.ready` | ✅ **api-gateway** deployed · 47s · port 3000 |
| `deployment.failed` | ❌ **api-gateway** failed · port 3000 already in use |

A mensagem inclui:
- **Nome do serviço** (em negrito)
- **Nome do projeto** e **environment** (contexto)
- **Quem disparou** (para deploys manuais: email; para webhooks/gitops: label)
- **Status final** (ready/failed) com duração ou mensagem de erro
- **Ícone de status** (✅ verde para ready, ❌ vermelho para failed, ⏳ amarelo para em progresso)

### Atributos Visuais da Notificação

```
┌──────────────────────────────────────────────────────────┐
│  ⏳ api-gateway                                          │
│  Building · commit a84f9b2 · funvest/production          │
│  triggered by neversonbs13@gmail.com · 32s ago           │
└──────────────────────────────────────────────────────────┘
```

Cores:
- Fundo do card: `surface.card`
- Borda esquerda: 3px sólido — cor do status (amarelo para building, verde para ready, vermelho para failed)
- Ícone à esquerda do nome do serviço
- Timestamp relativo alinhado à direita

---

# 21. Arquitetura de Notificações em Tempo Real

## 21.1 Canais de Entrega

| Canal | Público | Quando |
|---|---|---|
| **Toast (sooner) live** | Usuários logados na UI | Imediato, via SSE/WebSocket. Aparece no canto inferior direito (sooner), empilhando múltiplos. Auto-dismiss em 8s (ready) ou persistente (failed, clicável) |
| **Bell icon (header)** | Usuários logados ou que retornam | Contador de notificações não lidas no ícone de sino no header. Dropdown mostra as últimas 20. Clicar marca como lida. Marcar todas como lidas |
| **Webhook (outgoing)** | Integrações externas (Slack, Discord, etc.) | Enviado via HTTP POST com HMAC-SHA256 para canais configurados (já implementado) |
| **Email** | Usuários offline | Futuro: digest diário de deploys (opcional, configurável) |

## 21.2 Arquitetura Técnica

```
┌──────────┐     ┌──────────────┐     ┌─────────────────┐
│  Core    │────▶│  Event Bus   │────▶│  Notif. Engine  │
│ .Deploy  │     │  (Postgres   │     │  (in-memory +   │
│          │     │   outbox)    │     │   fan-out)      │
└──────────┘     └──────────────┘     └────────┬────────┘
                                               │
                          ┌────────────────────┼────────────────────┐
                          ▼                    ▼                    ▼
                    ┌──────────┐        ┌──────────┐        ┌──────────┐
                    │ SSE Hub  │        │ Webhook  │        │ Bell     │
                    │ (fan-out │        │ Sender   │        │ Store    │
                    │  per org)│        │          │        │ (DB)     │
                    └────┬─────┘        └──────────┘        └──────────┘
                         │
                    ┌────┴────┐
                    │ Clients │
                    │ (SSE)   │
                    └─────────┘
```

### Componentes

**1. Event Bus (já implementado)**
- `events.Bus` com outbox pattern em PostgreSQL
- `core.Bus.Publish(ctx, aggregate, id, type, payload, beforeCommit)`
- Eventos `deployment.queued`, `deployment.ready`, `deployment.failed` já publicados

**2. Notification Engine (novo)**
- Subscreve ao Event Bus para eventos `deployment.*`
- Transforma eventos crus em notificações formatadas (mensagem humana + metadados)
- Armazena notificações na tabela `notifications` para consumo assíncrono (bell icon)
- Publica no SSE Hub para entrega em tempo real

**3. SSE Hub (novo)**
- Mantém conexões SSE abertas por organização
- Cada cliente conectado recebe eventos da sua organização
- Heartbeat a cada 15s para manter a conexão viva
- Reconexão automática no client (Exponential backoff: 1s, 2s, 4s, 8s, max 30s)
- Endpoint: `GET /api/v1/events/stream?org=<orgID>` (autenticado)
- Formato do evento SSE:
  ```
  event: notification
  data: {"id":"...","type":"deployment.ready","message":"...","timestamp":"..."}
  ```

**4. Bell Store (novo)**
- Tabela `notifications`:
  ```sql
  CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    message TEXT NOT NULL,
    payload JSONB,
    read INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
  );
  CREATE INDEX idx_notif_org_user_read ON notifications(org_id, user_id, read, created_at DESC);
  ```
- API:
  - `GET /api/v1/notifications` — últimas 50 (com paginação por cursor)
  - `GET /api/v1/notifications/unread-count` — contador do badge no sino
  - `POST /api/v1/notifications/{id}/read` — marcar como lida
  - `POST /api/v1/notifications/read-all` — marcar todas como lidas

**5. Toast (Sooner) System (frontend)**
- Componente `ToastProvider` já existe no ui.tsx (canto inferior direito)
- Estender para receber notificações do SSE stream:
  ```tsx
  // Conectar ao SSE ao logar
  useEffect(() => {
    const es = new EventSource(`/api/v1/events/stream?org=${orgID}`, {
      headers: { Authorization: `Bearer ${token}` }
    });
    es.addEventListener('notification', (e) => {
      const data = JSON.parse(e.data);
      if (data.type === 'deployment.ready') {
        toast(data.message, 'success');   // verde, auto-dismiss 8s
      } else if (data.type === 'deployment.failed') {
        toast(data.message, 'error');     // vermelho, persistente, clicável → /services/:id/deployments
      } else {
        toast(data.message, 'info');      // neutro, auto-dismiss 5s
      }
    });
    es.onerror = () => { /* reconectar com backoff */ };
    return () => es.close();
  }, [orgID]);
  ```

## 21.3 UX do Bell Icon

```
┌──────────────────────────────────────────┐
│  🔔 3                                     │  ← badge com contador de não lidas
│                                           │
│  ┌────────────────────────────────────┐  │
│  │ Notifications                  ✕   │  │  ← dropdown ao clicar
│  ├────────────────────────────────────┤  │
│  │                                    │  │
│  │ ✅ api-gateway deployed            │  │
│  │    47s · port 3000 · 2m ago        │  │  ← verde, não lida (bold)
│  │                                    │  │
│  │ ❌ worker-emails failed            │  │
│  │    OOM · triggered by webhook      │  │  ← vermelha, não lida (bold)
│  │    · 5m ago                        │  │
│  │                                    │  │
│  │ ✅ postgres-main restored          │  │
│  │    · 1h ago                        │  │  ← lida (normal weight)
│  │                                    │  │
│  │ ──────────────────────────────     │  │
│  │ Mark all as read                   │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

- **Não lida**: peso da fonte `font-semibold`, background sutil `bg-surface-container-high/40`
- **Lida**: peso normal, sem background
- **Clicar** em uma notificação → marca como lida + navega para a página relevante (service detail / deployments tab)
- **"Mark all as read"**: marca todas como lidas, reseta o badge
- **Contador**: número de não lidas no badge do sino; esconde quando 0
- **Polling fallback**: se SSE falhar, o bell faz polling `GET /unread-count` a cada 30s

## 21.4 Comportamento Offline → Online

Quando um usuário **não está logado** durante um deploy:

1. Os eventos são publicados normalmente no Event Bus
2. O Notification Engine os transforma e armazena em `notifications` (com `user_id=''` para notificações da org)
3. Quando o usuário **loga**, o Bell Store retorna as notificações não lidas da sua organização
4. O badge do sino mostra a contagem acumulada
5. O dropdown lista as notificações em ordem cronológica reversa
6. Ao marcar como lida, o badge atualiza

Isso garante que **nenhum evento é perdido** — o usuário sempre vê o histórico do que aconteceu
enquanto estava offline, e recebe live updates quando online.

## 21.5 Estados e Edge Cases

### SSE Connection States
| Estado | Comportamento |
|---|---|
| **Connecting** | Spinner no badge do sino |
| **Connected** | Badge normal; toasts aparecem |
| **Reconnecting** | Backoff exponencial (1s → 2s → 4s → 8s → 30s max); badge mantém última contagem via polling |
| **Failed (após N tentativas)** | Fallback para polling a cada 30s; badge sem spinner |

### Edge Cases
- **Múltiplos deploys simultâneos**: toasts empilhados no canto direito (max 5 visíveis; restantes na fila)
- **Deploy cancelado**: evento `deployment.cancelled` → toast amarelo "Deploy cancelled"
- **Usuário em múltiplas orgs**: SSE stream filtra por `org_id` atual (a org ativa no contexto)
- **Alta latência**: notificações chegam via SSE com atraso; timestamp real (não do recebimento)
- **Token expirado durante SSE**: EventSource.onerror → reautenticar → reconectar

---

# 22. Ordem de Construção (Pré-requisitos)

## 22.1 Por que Notificações Antes de Service Management

A experiência de criação e deploy de serviços **depende** do sistema de notificações para
transmitir feedback ao usuário. Sem ele:

- O usuário cria um serviço e **não vê** o deploy acontecendo (a menos que recarregue a página)
- O usuário sai da página durante o deploy e **nunca sabe** se deu certo ou errado
- Múltiplos membros da organização **não são informados** sobre deploys de colegas
- Erros de deploy passam **despercebidos** até alguém verificar manualmente

**O sistema de notificações em tempo real é um pré-requisito arquitetural** para a
experiência completa de Service Management.

## 22.2 Sequência de Construção Recomendada

```
1. REAL-TIME NOTIFICATION INFRASTRUCTURE    (RFC-0031)
   ├── Event Bus outbox (já existe - ✅)
   ├── Notification Engine (subscrever deployment.*)
   ├── Bell Store (tabela notifications + API)
   ├── SSE Hub (fan-out por organização)
   └── Bell Icon UI + Toast live updates

2. DEPLOY EVENT EMISSION                     (RFC-0030 §20)
   ├── Enriquecer deployment.* eventos com triggered_by, service_name, etc.
   ├── Publicar deployment.ready / .failed com payload completo
   └── Wire ao Notification Engine

3. SERVICE MANAGEMENT UX                     (RFC-0030 §1-17)
   ├── Create Service modal + wizard
   ├── Launcher
   ├── Marketplace + Templates
   ├── Service Detail
   ├── Logs avançados
   └── Variables / Resources / Monitoring
```

**Por que nesta ordem:**
- Passo 1 estabelece a infraestrutura que os passos 2 e 3 consomem
- Passo 2 injeta os eventos que o passo 1 distribui
- Passo 3 é a experiência que o usuário final vê — e cada ação nela gera notificações do passo 1

## 22.3 Critérios de Aceite do Pré-requisito (RFC-0031)

Antes de iniciar qualquer implementação de Service Management:

- [ ] **SSE stream funcional**: `GET /api/v1/events/stream?org=X` retorna eventos em tempo real
- [ ] **Bell icon com badge**: contador de não lidas visível; dropdown lista notificações
- [ ] **Toast live**: ao receber evento SSE, aparece sooner no canto inferior direito
- [ ] **Offline→Online**: ao logar, badge mostra notificações acumuladas durante ausência
- [ ] **Bell Store**: CRUD de notificações; marcar lida; marcar todas lidas
- [ ] **Notification Engine**: transforma `deployment.*` eventos em notificações formatadas
- [ ] **Teste E2E**: disparar deploy → toast aparece em outra sessão (mesma org) em <2s
- [ ] **Reconexão SSE**: simular queda de rede → reconexão automática com backoff

---

## Checklist Fase 0 (Pré-requisito: Notificações)

- [ ] **RFC-0031: Real-Time Notification Infrastructure** (nova spec separada)
  - Accept: documento descrevendo arquitetura SSE Hub, Bell Store, Notification Engine, Toast system
- [ ] **Tabela `notifications`** + índices + migração
  - Accept: tabela criada; queries de list/unread-count/read/read-all funcionais
- [ ] **SSE Hub** (`GET /api/v1/events/stream`)
  - Accept: conexão mantida; eventos broadcast por org_id; heartbeat 15s
- [ ] **Notification Engine** (subscriber do Event Bus)
  - Accept: subscreve `deployment.*`; transforma em notificação; armazena no Bell Store; publica no SSE Hub
- [ ] **Bell Icon UI** (header, badge, dropdown, mark read)
  - Accept: badge visível com contagem; dropdown com últimas 20; clicar navega + marca lida; "Mark all read"
- [ ] **Toast System Upgrade** (SSE → sooner)
  - Accept: toast verde para ready, vermelho para failed, info para progresso; clicável para failed
- [ ] **Offline support** (polling fallback + histórico)
  - Accept: badging mantém contagem via polling quando SSE falha; ao logar, badge mostra acumulado


---

# DETALHAMENTO EXPANDIDO POR CAPÍTULO

As seções abaixo complementam os capítulos originais com especificações de nível
"implementação" — cada parágrafo deve ser tratado como critério de aceite.

---

## §2 (Expansão): Fluxo Create Service — Detalhamento por Campo

### 2.8 Validação por Campo

**Project (Select):**
- Obrigatório. Lista projetos da organização ordenados por nome.
- Se o modal foi aberto de dentro de um projeto (`fixedProjectId`), o select é desabilitado e exibe apenas o projeto atual.
- Placeholder: "Select a project..."
- Erro: "Project is required" (inline, abaixo do campo, vermelho)
- O campo recebe foco automaticamente ao abrir o modal (autoFocus)

**Name (Input):**
- Obrigatório. 1-64 caracteres. Regex: `^[a-z][a-z0-9-]*$` (lowercase, inicia com letra, permite números e hífens).
- Validação em tempo real (onBlur + onChange após primeiro blur): se inválido, mensagem abaixo do campo.
- Exemplos válidos: `api-gateway`, `worker-emails`, `frontend-v2`
- Exemplos inválidos: `API-Gateway` (maiúsculas), `_worker` (underscore inicial), `a` (muito curto após trim)
- Slug gerado automaticamente a partir do nome (usado em URLs, DNS interno, container naming)
- Placeholder: "ex: api-gateway"

**Source Type (Select):**
- Opções: "OCI Image", "Git Repository", "Template", "Docker Compose", "Database"
- Default: "OCI Image"
- Ao trocar, o formulário se reorganiza (sem perder dados já preenchidos — se possível, mapeia campos equivalentes)
- Ícone ao lado de cada opção no dropdown:
  - OCI Image: `image`
  - Git Repository: `code`
  - Template: `dashboard`
  - Docker Compose: `deployed_code`
  - Database: `storage`

**Image (Input, condicional — source=OCI Image):**
- Obrigatório quando source é OCI Image
- Formato: `[registry/][namespace/]name[:tag]`
- Validação: regex `^([a-zA-Z0-9._-]+(\/[a-zA-Z0-9._-]+)*\/)?[a-zA-Z0-9._-]+(:[a-zA-Z0-9._-]+)?$`
- Auto-complete: sugere imagens populares conforme digita (nginx:alpine, node:20-alpine, python:3.12-slim, redis:7-alpine, postgres:16-alpine) — 5 sugestões fixas que aparecem em um popover abaixo do campo após 2 caracteres
- Placeholder: "nginx:alpine"
- Ícone de copy (📋) no canto direito do campo: copia a última imagem usada do clipboard? Não — copia o valor ATUAL do campo (atalho visual)

**Git URL (Input, condicional — source=Git):**
- Obrigatório quando source é Git
- Aceita HTTPS (`https://github.com/org/repo.git`) e SSH (`git@github.com:org/repo.git`)
- Validação: regex para ambos os formatos
- Se o usuário conectou um provider (GitHub OAuth), este campo é substituído por um seletor de repositórios (dropdown com busca)
- Placeholder: "git@github.com:org/repo.git"

**Git Branch (Input, condicional — source=Git):**
- Default: "main"
- Placeholder: "main"
- Não obrigatório (usa o branch default do repositório se vazio)

**Dockerfile Path (Input, condicional — source=Git):**
- Default: "Dockerfile"
- Caminho relativo à raiz do repositório
- Placeholder: "Dockerfile"
- Não obrigatório (usa "Dockerfile" se vazio)

**Build Method (Select, condicional — source=Git):**
- Opções: "Dockerfile", "Nixpacks (auto-detect)", "Cloud Native Buildpacks"
- Default: "Dockerfile"
- Descrição curta abaixo de cada opção quando selecionada:
  - Dockerfile: "Use the Dockerfile in your repository"
  - Nixpacks: "Auto-detect framework and generate Dockerfile (requires nixpacks CLI)"
  - Buildpacks: "Cloud Native Buildpacks via pack CLI"

**Port (Input, type=number):**
- Obrigatório. Range: 1-65535.
- Default: 80
- Validação: número inteiro, dentro do range
- Placeholder: "80"
- Sugestão automática baseada na imagem (ex: 3000 para node, 8080 para go, 5432 para postgres) — apareceria como um chip abaixo do campo: "💡 Detected: 3000 (click to use)"

**Memory (Input, type=number):**
- Opcional. Range: 0-262144 (0 = unlimited). Unidade: MiB.
- Default: 0
- Slider visual (range input estilizado) + input numérico sincronizados
- Marcadores no slider: 128, 256, 512, 1024, 2048, 4096, 8192
- Placeholder: "0 = unlimited"

**Build Type (Select, condicional — source=Git):**
- Mesmo que Build Method — este campo aparece dentro da seção "Advanced"

**Health Check Enabled (Checkbox):**
- Default: false
- Quando marcado, revela campos adicionais: Path, Interval, Timeout, Retries

**Health Check Path (Input, condicional):**
- Default: "/"
- Placeholder: "/health"

### 2.9 Sequência de Animação do Modal

1. **Open**: overlay fade in (150ms opacity 0→1) + modal scale (150ms 0.95→1.0, ease-out)
2. **Focus**: autoFocus no primeiro campo (Project ou Name)
3. **Source switch**: ao trocar source type, a seção condicional (Image vs Git) faz fade out/in com duração de 200ms (crossfade). Campos que existem em ambos (Port, Memory) mantêm posição (animação de layout: altura automática com transition)
4. **Submit**: botão mostra spinner + texto "Creating..." + desabilita todos os campos (opacity 50%, pointer-events none). Overlay permanece até resposta da API.
5. **Success**: modal fecha (fade out 150ms). Toast aparece no canto inferior direito: "✅ Service 'api-gateway' created · deploying..."
6. **Redirect**: se o modal foi aberto da página do projeto, a lista de serviços recarrega e o novo serviço aparece com shimmer até o container iniciar
7. **Close**: se o usuário digitou algo e tenta fechar (Escape, clicar fora, botão Cancel), mostra um confirm: "Discard changes?" se houver campos preenchidos

### 2.10 Atalhos de Teclado Específicos do Modal

| Atalho | Ação |
|---|---|
| `Tab` | Próximo campo |
| `Shift+Tab` | Campo anterior |
| `Enter` | Submit (Create & Deploy) — somente se o foco NÃO estiver num Select aberto |
| `Cmd+Enter` | Create & Deploy (força, mesmo com Select aberto) |
| `Cmd+Shift+Enter` | Create without deploying |
| `Escape` | Fecha o modal (com confirmação se dirty) |
| `↑/↓` | Navega opções de Select quando aberto |

---

## §9 (Expansão): Tela de Detalhes do Service — Especificação por Aba

### 9.3 Overview Tab — Comportamento Detalhado

**Service Details Card:**
- Live URL: se o serviço tem um domínio configurado, mostra `https://<domain>` (com ícone de link externo). Se não tem domínio, mostra `http://<host>:<port>` (obtido do probe NetQ). Se nenhum dos dois, mostra "—". Botão copy (📋) copia a URL para clipboard com feedback visual (ícone muda para ✓ por 1.5s).
- Source: para image, mostra a imagem clicável (se for Docker Hub, link para hub.docker.com). Para git, mostra `org/repo:branch` com ícone de link externo para o repositório.
- Type: badge com o tipo (Application, Database, Worker, Cron, Compose).
- Port: número da porta (sem ":" na label, o ":" aparece antes do valor).
- Environment: nome do ambiente com link para a página do projeto com aquele ambiente selecionado.
- Project: nome do projeto com link.

**CPU / Memory Mini Gauges:**
- Atualização: a cada 5 segundos (poll `GET /api/v1/services/:id/stats`)
- CPU: barra horizontal, cor primária. Label: "CPU · 42%". Hover mostra tooltip com breakdown (user/system/iowait, se disponível no driver).
- Memory: barra horizontal, cor secundária. Label: "Memory · 256 MiB / 1 GiB". Hover mostra tooltip com RSS/Cache/Swap.
- Network: soma RX+TX bytes (incremental desde o início do container). Label: "Network · 1.2 MB ↓ · 340 KB ↑"
- IO: read+write bytes. Label: "Disk · 5 MB read · 2 MB write"
- Se o container está parado (state=exited), os gauges mostram "—" e um botão "Start" abaixo.

**Latest Deployment Card:**
- Se não há deployments: "No deployments yet. Click Deploy to start." + botão Deploy
- Se há um deployment em progresso: mostra o status atual com animação de pulso (pulse-dot) + timer desde o início (ex: "Building · 32s")
- Se o último deploy é ready: commit hash (8 chars, link para o repo), branch, trigger, "View Logs" link
- Se o último deploy falhou: erro em vermelho, "View Logs" link, botão "Retry" e botão "Rollback"
- Timestamp relativo (ex: "2 minutes ago", "1 hour ago") — atualiza a cada 30s

**Live Logs Terminal:**
- Altura: 300px, scrollável, fundo `#050505` (preto absoluto, contraste máximo)
- Fonte: JetBrains Mono 12px, line-height 1.5
- Segue o fim automaticamente (scroll ancorado no bottom). Se o usuário scrolla para cima, o auto-scroll pausa e um botão "Follow" aparece no canto inferior direito.
- Botão "Pause/Resume" no header do terminal
- Botão "Pop out" (abrir em full screen) — abre a tab Logs
- Atualização: SSE stream (mesmo endpoint de logs do serviço)

### 9.4 Deployments Tab — Comportamento Detalhado

**Timeline Vertical:**
- Cada deployment é uma linha horizontal com:
  - Coluna 1 (40px): círculo com ícone de status (✓ verde, ✗ vermelho, ◉ amarelo pulsando, ○ cinza)
  - Coluna 2 (flex): conteúdo:
    - Linha 1: `#N` (número do deploy) + status badge + trigger badge + timestamp relativo
    - Linha 2 (expansível): commit hash (8 chars, mono) + branch + "by Neverson" (se trigger=manual)
    - Linha 3 (condicional, failed only): mensagem de erro em vermelho
  - Coluna 3 (ações): botão "Rollback" (se status=ready), botão "View Logs" (link para tab Logs com filtro para esse deploy)
- Conector visual entre deployments: linha vertical de 2px conectando os círculos (cor: outline-variant)
- Ordem: deployment mais recente no topo (ordem cronológica reversa)
- Paginação: mostra últimos 20; botão "Load more" no final

**Filtro por Status:**
- Chips horizontais acima da timeline: All | Ready | Failed | Building | Queued
- Selecionar um chip filtra a timeline (animação: itens não-matching fazem fade out e colapsam altura)
- Badge com contagem ao lado de cada chip: "Ready (12)", "Failed (2)"

**Rollback Flow:**
1. Clique "Rollback" no deployment #2
2. Modal de confirmação:
   - Título: "Rollback to deployment #2?"
   - Descrição: "This will create a new deployment (#N) with the image and configuration from #2."
   - Se houver mudanças de env vars entre #2 e o atual: diff visual inline (verde = adicionada, vermelho = removida)
   - Checkbox: "☐ Also restore environment variables from #2" (default: false — mantém as atuais)
3. Confirma → deploy inicia com trigger "rollback"
4. Toast: "⏳ Rollback started · deploying #N..."
5. Timeline atualiza em tempo real (novo item aparece no topo com status "queued" → "building" → ...)

### 9.5 Settings Tab — Detalhamento

**General Section:**
- Service Name (readonly — renomear não é suportado; o nome é identidade imutável)
- Source Type (readonly — definido na criação)
- Image (editável para source=image): input com validação de formato. Botão "Save" abaixo.
- Git URL / Branch / Dockerfile Path / Build Method (editáveis para source=git)
- Port (editável): input number com validação. Alterar porta requer redeploy (mostra warning: "Changing port requires a new deploy")
- Botão "Save" unificado no final da seção

**Resources Section:**
- CPU: slider + input sincronizados. Step: 0.25. Range: 0.25-64.
- Memory: slider + input sincronizados. Step: 128. Range: 128-262144.
- GPU: toggle switch
- Alterar recursos aplica `docker update` (sem redeploy) para o container atual. Se o container estiver parado, as mudanças aplicam no próximo start.

**Health Check Section:**
- Toggle: Enable/Disable
- Path: input
- Interval: select (1s, 5s, 10s, 30s, 60s) ou custom
- Timeout: input number (segundos)
- Retries: input number
- Preview: indicador visual mostrando o comportamento: "After 3 failures (15s), service is marked unhealthy"

**Volumes Section:**
- Lista de volumes montados (nome → path)
- Adicionar: modal inline com campos Name + Mount Path + botão Add
- Remover: ícone de lixeira em cada linha, com confirmação
- Nota: "Volume changes require a new deploy"

**Webhook Section:**
- Secret: input com toggle mostrar/ocultar
- URL de exemplo: `POST /api/v1/webhooks/github/<serviceID>` (code block, copiável)
- Header: `X-Hub-Signature-256: sha256=<hmac>`
- Botão "Regenerate secret" (com confirmação)

**Danger Zone:**
- Card com borda vermelha sutil
- "Delete this service"
- Descrição: "This will permanently delete the service, all its deployments, logs, environment variables, domains and linked volumes. Active containers will be stopped. This action cannot be undone."
- Botão "Delete service" (danger variant)
- Confirmação: modal com input para digitar o nome do serviço (proteção contra clique acidental)
- Após confirmar: serviço deletado, redirect para /apps

### 9.6 Abas Restantes

**Cron Tab:**
- Lista de cron jobs do serviço (tabela: Name, Schedule, Command, Last Run, Next Run, Status)
- Botão "New Cron Job" → modal (Name, Schedule em formato cron ou preset, Command)
- Validação do cron em tempo real (mostra próximas 3 execuções previstas)
- Presets: Every minute, Every 5 min, Every hour, Daily at midnight, Weekly on Sunday

**Workers Tab:**
- Lista de workers (tabela: Name, Command, Replicas, Status)
- Botão "New Worker" → modal (Name, Command, Replicas)
- Start/Stop/Restart por worker

**Terminal Tab:**
- WebSocket terminal (`/api/v1/ws/terminal/:id`)
- Terminal escuro (xterm.js-like, fonte mono)
- Barra de título: nome do serviço + container ID (12 chars)
- Suporte a cores ANSI, redimensionamento responsivo, Ctrl+C/D
- Botão "Pop out" (abre em janela separada)

---

## §10 (Expansão): Deployments — Detalhamento Adicional

### 10.5 Deploy Comparison Algorithm

Quando o usuário seleciona dois deployments para comparar:

1. **Imagem**: string diff simples. Se diferente, mostra `de → para` com cores.
2. **Environment Variables**: diff de chaves:
   - Adicionadas (verde, sinal `+`): chaves presentes no target mas ausentes no source
   - Removidas (vermelho, sinal `-`): chaves presentes no source mas ausentes no target
   - Alteradas (amarelo, sinal `~`): chaves presentes em ambos com valores diferentes — mostra valor antigo → novo. Para secrets, mostra `•••• → ••••` (sem revelar valores).
3. **Resources**: CPU/Memory antes vs depois com barras visuais comparativas.
4. **Commit Range** (git only): se ambos deployments têm commits diferentes, mostra `git log a84f9b2..c91d2e3` — lista de commits entre eles (até 20, com link para o repo).
5. **Health Check Config**: se diferir, mostra diff.

### 10.6 Rollback Safety

- Rollback só é permitido se o deployment alvo está `ready` (não `failed` nem `building`).
- Rollback não altera o deployment original — sempre cria um NOVO (#N+1) com ImageRef do alvo.
- O deployment alvo permanece no histórico como estava.
- Se o rollback falhar, o serviço continua rodando com o deployment atual (não há "rollback do rollback" automático — é um deploy normal com trigger "rollback", que pode ser feito rollback dele mesmo depois).
- Métricas de rollback: contador de rollbacks por serviço (exibido no card do serviço como "Rollbacks: 3").

---

## §11 (Expansão): Logs — Especificação do Engine

### 11.5 Virtual Scrolling Engine

- **Biblioteca**: `@tanstack/virtual` (já usado em outros grids) ou implementação própria com IntersectionObserver.
- **Tamanho do buffer**: renderiza 100 linhas visíveis + 50 acima/abaixo (total 200 linhas no DOM).
- **Altura da linha**: fixa em 20px (code-md line-height).
- **Scroll**: ao atingir o topo, dispara fetch de chunk anterior (1.000 linhas) do arquivo de log via API com parâmetro `?before=<cursor>`.
- **Âncora de scroll**: ao receber novas linhas via SSE, se o usuário está no fim (a <40px do bottom), o scroll segue automaticamente. Caso contrário, um badge "▼ 23 new lines" aparece no canto inferior direito.

### 11.6 ANSI Parser

- **Implementação**: parser state machine que interpreta sequências de escape ANSI (SGR — Select Graphic Rendition).
- **Cores suportadas**: 16 cores padrão (30-37, 90-97), 256 cores (38;5;n), true color (38;2;r;g;b).
- **Atributos**: bold (1), dim (2), italic (3), underline (4), reverse (7), strikethrough (9).
- **Background**: 40-47, 100-107, 48;5;n, 48;2;r;g;b.
- **Reset**: 0 (reseta todos os atributos).
- **Renderização**: cada sequência gera um `<span>` com classes CSS correspondentes. Ex: `\x1b[1;31mERROR\x1b[0m` → `<span class="font-bold text-red-400">ERROR</span>`.
- **Performance**: parser opera linha a linha (não acumula estado entre linhas — cada linha começa com reset implícito). Timeout de parse: se uma linha tem >100 sequências, fallback para texto puro (proteção contra ANSI bombs).

### 11.7 JSON Detection e Pretty Print

- **Detecção**: se uma linha começa com `{` ou `[` e termina com `}` ou `]`, é tratada como JSON.
- **Renderização**: JSON syntax highlighting (chaves em amarelo, strings em verde, números em azul, boolean/null em roxo, brackets em cinza).
- **Expandir**: clique na linha JSON → expande inline com indentação (2 espaços), colapsável por chave.
- **Copy**: botão copy no canto da linha JSON expandida (copia o JSON formatado).
- **Performance**: parsing JSON via `JSON.parse` com try/catch. Se falhar (JSON inválido ou muito grande >100KB), renderiza como texto puro.

### 11.8 Filter Engine

- **Regex search**: implementado com `new RegExp(pattern, 'gi')`. Input do usuário com toggle "Regex" (default: off — busca literal).
- **Highlight**: termo encontrado é envolvido em `<mark>` com fundo amarelo (`bg-yellow-400/30`) e texto corrente.
- **Match count**: "Found 42 matches" aparece na barra de busca.
- **Navegação**: setas `↑/↓` ou botões "Previous/Next" pulam entre matches (scroll até o match, highlight pulsante breve).
- **Scrollbar markers**: linhas com match são indicadas por traços coloridos na scrollbar (usando `scrollbar-gutter` + pseudo-elementos ou canvas overlay).
- **Multiple filters**: futuro — combinar filtros com AND/OR (ex: `level=ERROR AND message~timeout`).
- **Saved filters**: futuro — bookmarks de filtros frequentes.

### 11.9 Download e Export

- **Download raw**: botão "Download" baixa as últimas N linhas (default 10.000, configurável) como `.txt`.
- **Download JSON**: se o log é predominantemente JSON, opção de baixar como `.jsonl` (JSON Lines).
- **Download range**: date picker para selecionar range de datas e baixar logs daquele período.

---

## §12 (Expansão): Variables — Detalhamento

### 12.6 Cache e Invalidação

- **Estrutura de cache**: `sync.Map` em memória, keyed by `projectID` ou `environmentID`.
- **Invalidação**: qualquer write (set/delete/bulk_replace) invalida a entrada do cache.
- **TTL**: 60 segundos (fallback — se o cache não for invalidado por um bug, expira naturalmente).
- **Multi-instância**: no futuro, usar PostgreSQL `LISTEN/NOTIFY` para invalidar cache entre instâncias — channel `env_cache_inval:<projectID>`.
- **Warm-up**: ao iniciar a API, pre-carrega cache dos projetos mais acessados (opcional, configurável).

### 12.7 Rotação de Chaves de Criptografia

- **KEK (Key Encryption Key)**: chave mestra armazenada em `keys/master.key` (32 bytes, gerada no primeiro boot, permissão 0600).
- **DEK (Data Encryption Key)**: chave derivada do KEK via HKDF-SHA256 com salt fixo `"aether:env:v1"`.
- **Rotação**: comando `aether rotate-keys`:
  1. Gera novo KEK.
  2. Re-encripta todos os secrets com o novo KEK (em transação).
  3. Armazena KEK antigo como backup (`keys/master.key.old`).
  4. Registra evento de auditoria.
- **Rollback**: `aether rotate-keys --rollback` reverte para o KEK anterior.

### 12.8 Integração Futura com Vaults Externos

- **Providers planejados**: HashiCorp Vault, AWS Secrets Manager, GCP Secret Manager, Azure Key Vault.
- **Modo**: o Aether lê secrets do provider externo e os injeta como variáveis de ambiente no runtime, sem armazená-los no banco.
- **Configuração**: por projeto ou por environment, definindo `vault_provider`, `vault_path`, credenciais de acesso.
- **Resolução**: se um secret tem referência a vault (`${{vault.path/to/secret}}`), o runtime busca no provider configurado.
- **Cache**: secrets de vault são cacheados em memória com TTL curto (30s) para evitar latência de rede a cada deploy.

---

## §16 (Expansão): Monitoring — Detalhamento

### 16.6 Agregação e Storage

- **Resolução de armazenamento**:
  - Últimos 60 minutos: 1 amostra a cada 5 segundos (720 pontos)
  - Últimas 24 horas: 1 amostra a cada 1 minuto (1440 pontos)
  - Últimos 7 dias: 1 amostra a cada 10 minutos (1008 pontos)
  - Últimos 30 dias: 1 amostra a cada 1 hora (720 pontos)
- **Armazenamento**: tabela `metrics` no PostgreSQL com particionamento por dia (partições automáticas).
- **Retenção**: 30 dias (configurável via `AETHER_METRICS_RETENTION_DAYS`).
- **Compressão**: dados com >24h são agregados (AVG, MIN, MAX, P95) e os pontos raw são descartados.

### 16.7 Prometheus Endpoint

- `GET /metrics` (público ou autenticado, configurável)
- Formato: Prometheus text exposition format.
- Métricas expostas:
  - `aether_service_cpu_percent{service,project,environment}` — gauge
  - `aether_service_memory_bytes{service,project,environment}` — gauge
  - `aether_service_network_rx_bytes{...}` / `_tx_bytes{...}` — counter
  - `aether_service_io_read_bytes{...}` / `_write_bytes{...}` — counter
  - `aether_service_up{service,project,environment}` — gauge (1/0)
  - `aether_deployments_total{service,status}` — counter
  - `aether_platform_info{version,runtime}` — gauge
- Labels incluem `service`, `project`, `environment` para filtragem no Prometheus/Grafana.

### 16.8 Alert Rules Engine

- **Triggers**:
  - `cpu > 90% for 5m` → severity: warning
  - `cpu > 95% for 2m` → severity: critical
  - `memory > 90% for 5m` → severity: warning
  - `memory > 95% for 2m` → severity: critical
  - `disk > 85%` → severity: warning
  - `service_down for 1m` → severity: critical
  - `deploy_failed` → severity: warning (evento, não métrica)
- **Actions**:
  - Notificação no Bell Icon (sempre)
  - Webhook para canais configurados (Slack/Discord/Telegram/Email)
  - Auto-restart (se configurado): quando `service_down for 1m`, tenta restart automático
- **Silence window**: configurável por serviço (ex: não alertar durante janela de manutenção 02:00-04:00 UTC)
- **Escalation**: se critical persiste por >15min, re-envia notificação com tag "ESCALATED"

---

## §17 (Expansão): UI/UX Design System — Especificação de Componentes

### 17.5 Biblioteca de Componentes — Catálogo Completo

**Form Components:**
- `Input` — text, email, password, number, url. Modos: default, com ícone à esquerda, com ação à direita (ex: toggle password visibility), loading (spinner à direita), disabled, readonly, error (borda vermelha + mensagem abaixo).
- `Select` — dropdown nativo com custom trigger (chevron animado 180° ao abrir). Opções com ícones, grupos (optgroup), disabled options. Search mode (typeahead — filtra opções conforme digita).
- `Textarea` — auto-resize (cresce conforme conteúdo, max-height configurável). Contador de caracteres no canto inferior direito. Suporte a syntax highlighting (modo code).
- `Checkbox` — custom (18px, rounded 6px, primary check). Label clicável.
- `Switch/Toggle` — alternativa ao checkbox para estados binários (ex: Enable/Disable). Animação de deslize (150ms ease-out).
- `Radio Group` — grupo de opções mutuamente exclusivas. Estilo: cards ou inline.
- `Slider` — range input customizado com track, fill, thumb. Labels nos extremos. Marcadores intermediários. Dica: tooltip no thumb mostrando valor atual.
- `Color Picker` — input de texto para hex + preview circular + popover com picker (futuro: spectrum completo).
- `File Upload` — área de drop com borda tracejada + botão "Browse". Preview do arquivo (nome, tamanho). Progresso de upload. Múltiplos arquivos.

**Navigation Components:**
- `Breadcrumb` — trilha de navegação: Workspace > Project > Environment > Service. Responsivo: em telas pequenas, mostra apenas o último item + "..." dropdown.
- `Tabs` — navegação horizontal com underline animado (desliza para a tab ativa, 200ms ease-out). Suporte a badges (contagem) em cada tab.
- `Sidebar` — implementado. Suporte a colapsar (ícones apenas, expande no hover).
- `Command Palette` — overlay central, busca fuzzy, navegação por teclado. (Ver §3 Launcher).
- `Pagination` — controles de página (Previous/Next + números). Opcional: infinite scroll com IntersectionObserver.

**Feedback Components:**
- `Toast (Sooner)` — notificações temporárias no canto inferior direito. Tipos: success (verde), error (vermelho), warning (amarelo), info (azul). Comportamento: empilha múltiplos, auto-dismiss configurável, clicável (ação customizada).
- `Alert` — banner inline no topo da página ou seção. Tipos: info, warning, error, success. Dismissível (✕). Ação opcional (botão).
- `Skeleton` — placeholder de carregamento com shimmer animation. Variantes: text (linhas de largura variável), card (retângulo), circle (avatar), table (grid de linhas).
- `Spinner` — indicador de carregamento circular. Tamanhos: sm (16px), md (24px), lg (40px). Com label opcional.
- `Progress Bar` — barra horizontal com preenchimento. Modos: determinado (percentual), indeterminado (animação infinita). Cor por contexto (primary, success, warning, error).
- `Empty State` — ilustração centralizada + título + descrição + CTA (action button).

**Overlay Components:**
- `Modal` — overlay com backdrop blur + centerizado. Tamanhos: sm (400px), md (560px), lg (800px), xl (1000px), fullscreen. Header com título + botão fechar. Footer com ações. Fecha com Escape, clique fora, botão ✕.
- `Popover` — dropdown posicionado relativo ao trigger. Ancoragem: top/bottom/left/right, com auto-flip (se não cabe, posiciona no lado oposto). Largura: min-content ou fixa. Conteúdo: lista de ações, formulário, informação. (Ver §17.2 para especificação do componente único).
- `Drawer` — painel que desliza da direita (ou esquerda). Overlay com opacidade reduzida. Tamanhos: sm (320px), md (480px), lg (640px).
- `Tooltip` — texto flutuante ao passar o mouse. Posição: top/bottom/left/right. Delay: 300ms para aparecer, instantâneo para desaparecer. Máximo de 200 caracteres.
- `ConfirmDialog` — modal especializado para confirmações destrutivas. Título + descrição + checkbox "I understand" + botão danger.

**Data Display Components:**
- `Table` — cabeçalho fixo, colunas ordenáveis (clicar no header alterna ASC/DESC/NONE, indicador visual ▲/▼). Linhas hover, striped opcional. Selecionável (checkbox por linha). Expansível (row detail). Responsivo: scroll horizontal.
- `Card` — container com borda, padding, radius. Variantes: default (bg surface-card), glass (backdrop-blur), interactive (hover border-primary).
- `Badge` — chip pequeno com texto. Variantes: default, primary, success, warning, error, outline. Tamanhos: sm, md.
- `StatusPill` — badge com dot pulsante para status (ready, failed, building, queued, ...). Implementado.
- `CodeBlock` — bloco de código com syntax highlighting (futuro) + botão copy. Scroll horizontal se necessário.
- `Timeline` — lista vertical com ícones e conectores. Usado em Deployments.
- `Stat` — número grande com label abaixo. Variantes: com ícone, com tendência (↑↓ porcentagem).
- `Avatar` — círculo com iniciais ou imagem. Tamanhos: sm (24px), md (32px), lg (40px). Com status dot (online/offline).
- `Tag` — etiqueta colorida (ex: "production", "staging"). Cores customizáveis.

**Chart Components (futuro, F5+):**
- `Sparkline` — mini gráfico de linha sem eixos (usado em cards de métrica).
- `Gauge` — indicador circular ou semicircular (CPU, Memory).
- `AreaChart` / `LineChart` — gráficos de séries temporais (métricas ao longo do tempo).
- `BarChart` — gráfico de barras (comparações).

---

## §18 (Expansão): Arquitetura — Schemas de API, Erros e Rate Limiting

### 18.8 API Response Envelope

Toda resposta da API segue o formato:

```json
{
  "data": { ... },           // payload principal (presente em 2xx)
  "error": {                 // presente em 4xx/5xx
    "code": "INVALID_INPUT",
    "message": "human-readable message",
    "details": [             // opcional — erros por campo
      { "field": "name", "message": "Name is required" }
    ]
  },
  "meta": {                  // presente em respostas paginadas
    "cursor": "abc123",
    "has_more": true
  }
}
```

**HTTP Status Codes:**
| Código | Significado |
|---|---|
| 200 | Success |
| 201 | Created |
| 202 | Accepted (async — deploy queued) |
| 204 | No Content (delete successful) |
| 400 | Bad Request (validação) |
| 401 | Unauthorized (token ausente/inválido) |
| 403 | Forbidden (sem permissão) |
| 404 | Not Found |
| 409 | Conflict (recurso duplicado, estado inválido) |
| 422 | Unprocessable Entity (regra de negócio — ex: deletar último environment) |
| 429 | Too Many Requests (rate limit) |
| 500 | Internal Server Error |
| 503 | Service Unavailable (banco fora, runtime down) |

### 18.9 Rate Limiting

- **Algoritmo**: Token Bucket por usuário (identificado pelo JWT `Subject`).
- **Limites**:
  - Leitura (GET/HEAD): 300 requisições por minuto
  - Escrita (POST/PUT/PATCH/DELETE): 60 requisições por minuto
  - Deploy: 10 requisições por minuto
- **Headers de resposta**:
  - `X-RateLimit-Limit`: limite da janela
  - `X-RateLimit-Remaining`: requisições restantes
  - `X-RateLimit-Reset`: timestamp Unix de quando o limite reseta
- **429 Response**: corpo com `{"error":{"code":"RATE_LIMITED","message":"Too many requests","retry_after":42}}` e header `Retry-After: 42`.
- **Burst**: permitido 20% acima do limite por até 10 segundos (short burst para operações em lote).

### 18.10 API Versioning

- **Versão atual**: `v1` (prefixo `/api/v1/`).
- **Política de depreciação**: endpoints deprecated continuam funcionando por 2 minor versions, retornando header `X-Deprecated: true` e `Sunset: <date>`.
- **Breaking changes**: somente em major versions (v2, v3). Migração documentada com período de coexistência.
- **Compatibilidade**: clientes podem especificar versão via header `Accept: application/json; version=1` como alternativa ao prefixo de URL.

### 18.11 Paginação

- **Método**: cursor-based (recomendado para listas grandes) ou offset-based (para listas pequenas com total conhecido).
- **Cursor-based**: parâmetro `?before=<cursor>` ou `?after=<cursor>`. Response inclui `meta.cursor` (último ID) e `meta.has_more`.
- **Offset-based**: parâmetros `?offset=0&limit=20`. Response inclui `meta.total`.
- **Default limit**: 20 itens. Max: 100 (configurável via `?limit=`).

---

## §19 (Expansão): Checklist — Expansão das Fases

### Fase 1 — Fundação (expandida)

- [ ] **Create Service modal com todos os campos e validações (§2.8)**
  - Accept: todos os campos do §2.8 implementados com validação inline, mensagens de erro e placeholders.
- [x] **Progressive disclosure funcional**
  - Accept: seções avançadas colapsadas; expandir/colapsar com animação; estado persiste durante a sessão (localStorage).
- [x] **Animações do modal (§2.9)**  <!-- fade+scale no Modal; spinner+disable no submit -->
  - Accept: open/close com fade+scale; transição de source type com crossfade; submit com spinner e disable.
- [ ] **Atalhos de teclado do modal (§2.10)**
  - Accept: Tab, Enter, Escape, Cmd+Enter funcionais em todos os campos.
- [x] **Auto-suggest de imagens populares**  <!-- dropdown 8 imagens populares -->
  - Accept: após 2 caracteres no campo Image, dropdown mostra 5 sugestões; clicar preenche.
- [x] **Template source type com grid de templates recentes+populares**  <!-- grid 18 trending preenche form -->
  - Accept: ao selecionar "Template", modal mostra grid; busca funcional; clicar preenche o form.
- [ ] **Launcher com search + create + navigate (§3)**
  - Accept: overlay abre com Cmd+Shift+K; busca fuzzy; 3 seções renderizadas; Enter executa.
- [ ] **Modos de busca no Launcher (§3.4)**
  - Accept: `>`, `@`, `#` filtram corretamente; indicador visual do modo ativo no placeholder.
- [ ] **Recentes e Favoritos no Launcher (§3.5)**
  - Accept: recentes persistem em localStorage; favoritos via toggle ⭐.
- [x] **Acessibilidade do Launcher (§3.7)**  <!-- role=dialog aria-live; teclado completo -->
  - Accept: navegação completa por teclado; role="dialog"; aria-live para resultados.

### Fase 2 — Marketplace (expandida)

- [x] **Catálogo com ≥100 templates, 30+ categorias**  <!-- 100 templates, 30 categorias, installs -->
  - Accept: cada template tem nome, ícone, descrição, badge verified/community, contagem de installs.
- [x] **Featured + Trending + Editor's Choice**  <!-- migration 15: editors_choice + 6 curados; endpoint ?editors_choice=true -->
  - Accept: featured = curadoria manual (flag no banco); trending = installs nos últimos 7d; editor's choice = flag separada.
- [ ] **Página do template com README renderizado**
  - Accept: markdown → HTML com syntax highlighting para blocos de código; scroll interno na descrição.
- [ ] **Instalação com 1 clique + modal de configuração**
  - Accept: defaults do template pré-preenchem o Create Service modal.
- [ ] **Busca com fuzzy matching + filtros**
  - Accept: busca por nome, descrição, tags, categoria; filtros: verified, categoria, ordenação.
- [ ] **Submission de templates pela comunidade**
  - Accept: formulário de submissão (GitHub PR ou formulário na UI); revisão manual pela equipe.

### Fase 3 — UX Premium (expandida)

- [ ] **Design tokens completos aplicados em todos os componentes**
  - Accept: cores de superfície (background → card → popover → modal) visualmente distintos; shadows por elevação; radius consistente.
- [ ] **Sistema de motion (§17.1 — Motion tokens)**
  - Accept: todas as transições usam `duration.*` + `easing.default`; `prefers-reduced-motion` desabilita animações.
- [ ] **Popover unificado — zero variações inline**
  - Accept: grep no código não encontra `bg-popover` ou `shadow-` hardcoded em componentes de dropdown; tudo usa `<Popover>`.
- [ ] **Skeleton loading em todas as listas e cards**
  - Accept: shimmer animation em: lista de serviços, lista de projetos, timeline de deployments, tabela de variáveis.
- [ ] **Keyboard navigation universal**
  - Accept: Tab/Enter/Escape/Arrow keys funcionais em: modais, popovers, selects, launcher, tabelas ordenáveis.
- [ ] **Acessibilidade WCAG AA**
  - Accept: contraste verificado com ferramenta (Lighthouse axe-core 0 violations); todos os ícones têm aria-label; modais têm role="dialog".

### Fase 4 — Wizards (expandida)

- [ ] **Application Wizard — Framework Detection Engine**
  - Accept: 15+ frameworks detectados (tabela §4.3); shallow clone do repo; sugestão de build method/resources.
- [ ] **Application Wizard — GitHub/GitLab OAuth**
  - Accept: conectar conta → listar repositórios → selecionar → preencher URL e branch automaticamente.
- [ ] **Application Wizard — Fallback quando detection falha**
  - Accept: mensagem clara + opções manuais (Generic Dockerfile, Nixpacks, Buildpacks).
- [ ] **Database Wizard — Grid de engines por categoria**
  - Accept: tabs laterais ou sidebar de categorias; busca filtra engines; card mostra ícone, nome, versões, count de instalações.
- [ ] **Database Wizard — Provisionamento com progresso real**
  - Accept: steps visuais (Pull → Volume → Start → Healthcheck → Ready) com checkmarks e timer estimado.
- [ ] **Compose Wizard — Monaco Editor + Live Validation**
  - Accept: syntax highlighting YAML; error lens (squiggly lines); painel lateral com análise (services, volumes, networks, warnings); dependency graph SVG.

### Fase 5 — Detalhe do Service Premium (expandida)

- [ ] **Todas as abas do Service Detail implementadas (§9.2-9.6)**
  - Accept: Overview, Deployments, Logs, Metrics, Variables, Settings, Cron, Workers, Terminal — cada uma renderiza dados reais.
- [ ] **Service Details Card com Live URL + Source + Type + Port + Env + Project**
  - Accept: todos os campos renderizados com links funcionais; copy URL com feedback visual.
- [ ] **CPU/Memory/Network/IO gauges com atualização a cada 5s**
  - Accept: polling ou SSE; gauges mostram valores reais; tooltip com breakdown.
- [x] **Latest Deployment Card com status, commit, timestamp, View Logs**
  - Accept: estados empty/progress/ready/failed renderizados corretamente; botões Retry e Rollback aparecem no estado failed.
- [ ] **Logs Engine completo (§11.5-11.9)**
  - Accept: virtual scrolling com infinite scroll; ANSI colors renderizados; JSON detection + pretty print; regex search com highlight e match count; download raw/JSON.
- [ ] **Filter Engine dos Logs**
  - Accept: search input com toggle regex; navigation entre matches; scrollbar markers.
- [x] **Variable Editor com auto-save, syntax highlight, duplicate detection**
  - Accept: editor textarea; salva após 2s; duplicatas destacadas; secrets toggle reveal.
- [ ] **Cache e invalidação de variáveis (§12.6)**
  - Accept: cache em memória; invalidação em write; TTL 60s fallback.
- [ ] **Settings Tab — todas as seções**
  - Accept: General, Resources, Health Check, Volumes, Webhook, Danger Zone — cada uma com save individual e validação.
- [x] **Danger Zone com confirmação por digitação do nome**  <!-- requireType no ConfirmDialog -->
  - Accept: modal exige digitar o nome exato do serviço antes de habilitar o botão Delete.

### Fase 6 — Enterprise (expandida)

- [ ] **Autoscaling horizontal com triggers de CPU**
  - Accept: slider min/max replicas; scale up/down baseado em CPU%; cooldown configurável; preview gráfico.
- [x] **Snapshot scheduling**  <!-- cron + retenção + scheduler + notificação -->
  - Accept: agenda cron; snapshots automáticos; política de retenção (últimos N).
- [ ] **Métricas com armazenamento e retenção (§16.6)**
  - Accept: tabela metrics com particionamento; agregação por resolução; retenção 30d.
- [ ] **Prometheus endpoint (§16.7)**
  - Accept: `/metrics` expõe métricas no formato Prometheus; labels de service/project/environment.
- [ ] **Alert Engine com triggers e silencing (§16.8)**
  - Accept: regras de alerta configuráveis; notificação via canais; silencing por janela.
- [ ] **Vault Integration (§12.8)**
  - Accept: suporte a HashiCorp Vault; leitura de secrets externos; injeção no runtime.
- [x] **Rate Limiting (§18.9)**  <!-- headers X-RateLimit-* + 429 + Retry-After -->
  - Accept: token bucket por usuário; headers X-RateLimit-*; resposta 429.
- [ ] **API Versioning (§18.10)**
  - Accept: header Sunset em endpoints deprecated; coexistência de versões.

EOF
echo "expandido"