# Aether Platform — Documentação Oficial de Arquitetura

> **Status:** Documento de arquitetura v0.1 — Produzido antes do início da implementação.
> **Objetivo:** Servir como referência oficial e canônica de arquitetura durante todo o ciclo de desenvolvimento do produto.
> **Classificação:** Público (documentação de arquitetura de projeto open source).

---

## O que é a Aether Platform

Aether é uma plataforma **Self-Hosted PaaS Open Source** projetada como concorrente direta de
**Coolify** e **Dokploy**, com paridade funcional de primeira versão e uma vantagem competitiva
estrutural: **eficiência extrema no consumo de recursos**.

A plataforma é concebida como um **Sistema Operacional para Aplicações** — um plano de controle
que administra conceitos de alto nível (Applications, Projects, Deployments, Domains, Servers,
Organizations, Templates, Environments) e trata containers como **mera implementação** de um
runtime abstrato definido pelos padrões da **Open Container Initiative (OCI)**.

## Leitura obrigatória

A documentação deve ser lida na seguinte ordem conceitual:

| # | Documento | Descrição |
|---|-----------|-----------|
| 1 | [`00-manifesto.md`](00-manifesto.md) | Filosofia, missão, princípios inegociáveis |
| 2 | [`01-visao-geral.md`](01-visao-geral.md) | Visão geral do produto e das funcionalidades |
| 3 | [`02-analise-concorrentes.md`](02-analise-concorrentes.md) | Engenharia reversa profunda de Coolify e Dokploy |
| 4 | [`03-metas-engenharia.md`](03-metas-engenharia.md) | Metas objetivas e mensuráveis de engenharia |
| 5 | [`04-analise-consumo-recursos.md`](04-analise-consumo-recursos.md) | Análise profunda de SSD / RAM / CPU / processos |
| 6 | [`05-arquitetura-geral.md`](05-arquitetura-geral.md) | Arquitetura lógica, física, de domínio, modular e em camadas |
| 7 | [`06-dominios-sistema.md`](06-dominios-sistema.md) | Bounded Contexts e responsabilidades |
| 8 | [`07-execution-engine.md`](07-execution-engine.md) | Execution Engine OCI (Podman/Buildah/Skopeo/Quadlet/crun) |
| 9 | [`08-networking-engine.md`](08-networking-engine.md) | Networking Engine, proxy e TLS |
| 10 | [`09-certificate-engine.md`](09-certificate-engine.md) | Certificate Engine próprio |
| 11 | [`10-persistencia.md`](10-persistencia.md) | Estratégia de persistência (SQLite/PostgreSQL/HA) |
| 12 | [`11-cache.md`](11-cache.md) | Estratégia de cache (TTL/LRU/GC) |
| 13 | [`12-event-bus.md`](12-event-bus.md) | Event Bus, event-sourcing, zero polling |
| 14 | [`13-observabilidade.md`](13-observabilidade.md) | Logs, metrics, tracing, alertas, audit |
| 15 | [`14-plugin-system.md`](14-plugin-system.md) | Sistema de plugins desacoplados |
| 16 | [`15-installer.md`](15-installer.md) | Instalador inteligente |
| 17 | [`16-seguranca.md`](16-seguranca.md) | Arquitetura de segurança |
| 18 | [`17-api-cli.md`](17-api-cli.md) | API e CLI |
| 19 | [`18-roadmap.md`](18-roadmap.md) | Roadmap técnico em fases |

## RFCs

Cada grande módulo possui uma RFC própria com objetivo, escopo, responsabilidades, arquitetura,
fluxos, interfaces, eventos, casos de uso, trade-offs, decisões, riscos e alternativas descartadas.

Ver [`rfc/README.md`](rfc/README.md).

| RFC | Módulo |
|-----|--------|
| [`RFC-0001`](rfc/RFC-0001-execution-engine.md) | Execution Engine |
| [`RFC-0002`](rfc/RFC-0002-networking-engine.md) | Networking Engine |
| [`RFC-0003`](rfc/RFC-0003-certificate-engine.md) | Certificate Engine |
| [`RFC-0004`](rfc/RFC-0004-persistence.md) | Persistência |
| [`RFC-0005`](rfc/RFC-0005-event-bus.md) | Event Bus |
| [`RFC-0006`](rfc/RFC-0006-runtime-driver-api.md) | API de Runtime Driver |
| [`RFC-0007`](rfc/RFC-0007-observability.md) | Observabilidade |
| [`RFC-0008`](rfc/RFC-0008-security.md) | Segurança |
| [`RFC-0009`](rfc/RFC-0009-installer.md) | Instalador |
| [`RFC-0010`](rfc/RFC-0010-plugin-system.md) | Plugin System |
| [`RFC-0011`](rfc/RFC-0011-deployments.md) | Pipeline de Deployments |
| [`RFC-0012`](rfc/RFC-0012-git-integration.md) | Integração Git (GitHub/GitLab/Bitbucket) |
| [`RFC-0013`](rfc/RFC-0013-databases.md) | Databases gerenciadas |
| [`RFC-0014`](rfc/RFC-0014-backup-restore.md) | Backup e Restore |
| [`RFC-0015`](rfc/RFC-0015-multiserver.md) | Multi Server e Clusters |
| [`RFC-0016`](rfc/RFC-0016-marketplace.md) | Marketplace, Templates e One-Click Apps |
| [`RFC-0017`](rfc/RFC-0017-api-cli.md) | API e CLI |
| [`RFC-0018`](rfc/RFC-0018-rbac.md) | Organizations, Teams e RBAC |

## Convenções adotadas neste repositório de documentação

1. **Português** é o idioma canônico dos documentos; termos técnicos em inglês são preservados.
2. Nomes de módulos, componentes e domínios usam **Inglês técnico** (ex.: `Execution Engine`) para
   facilitar a futura tradução da base de código.
3. Cada decisão arquitetural deve conter: **Justificativa**, **Alternativas consideradas** e
   **Trade-offs**.
4. Nenhuma decisão pode ser alterada sem atualizar o documento que a descreve e a RFC correspondente.
5. As metas de engenharia definidas em [`03-metas-engenharia.md`](03-metas-engenharia.md) são o
   critério de aceite de qualquer PR de arquitetura.

---

## Sumário executivo

- **Concorrência:** paridade funcional com Coolify e Dokploy na primeira versão (migração simples).
- **Diferencial:** arquitetura de consumo mínimo de recursos, sem competir com as aplicações do usuário.
- **Base:** padrões OCI, `Execution Engine` abstrato, Podman/Buildah/Skopeo/Quadlet/crun como implementação padrão.
- **Persistência:** SQLite (Community), PostgreSQL (Business), PostgreSQL HA (Enterprise).
- **Estilo:** orientado a eventos, zero polling, plugins sob demanda, camadas superiores nunca conhecem o runtime.
