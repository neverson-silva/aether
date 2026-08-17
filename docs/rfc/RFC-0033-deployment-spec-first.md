# RFC-0033 — Deployment Spec First + Docker Compose First

## Status: Implementado (fase 1 — modelo, generator, parser, histórico, aba Compose)

## Objetivo
Desacoplar a modelagem da execução: a plataforma passa a ser orientada por um
**Deployment Spec** tipado (fonte de verdade), serializado em docker-compose.yml
para execução. O Dockerfile deixa de ser o centro e vira apenas uma forma de
construir imagens.

## Arquitetura
```
UI → Deployment Spec (tipado, independente de runtime)
        ↓
   Compose Generator
        ↓
   docker-compose.yml
        ↓
   docker compose up
```

## Módulo runtime/compose (novo)
- `spec.go` — modelo tipado `DeploymentSpec`: service, build, image, command,
  entrypoint, ports, expose, environment, secrets, volumes, networks, domains,
  labels, restart, healthcheck, resources, runtime, strategy.
- `generator.go` — `Generate(spec)` → docker-compose.yml completo (services,
  networks, volumes, deploy/resources).
- `parser.go` — `Parse(yaml)` → DeploymentSpec (importação).
- `validator.go` — consistência básica + `ServiceName` sanitizador.

Outros runtimes futuros (kubernetes, swarm, nomad) consumirão o mesmo modelo
sem alterar a UI.

## Core (deployspec.go)
- `AppToSpec(app)` — converte domain.App (+ envs, volumes, domains, portas,
  healthcheck, resources) em DeploymentSpec.
- `GenerateCompose(app)` — compose ao vivo.
- No `Deploy`, o compose + hash são gerados e **armazenados por deployment**
  (migration 22: deployments.compose_yaml / deploy_spec / compose_hash),
  habilitando rollback, diff e auditoria.

## API
- `GET /apps/{id}/compose` (ao vivo, ?download=1)
- `GET /apps/{id}/spec` (Deployment Spec JSON)
- `GET /apps/{id}/deployments/{depID}/compose` (histórico)
- `POST /apps/{id}/compose/import` (importa compose → spec → aplica no app)

## UI
- Nova aba **Compose** com Monaco Editor (read-only, YAML highlight, line
  numbers, fold, word wrap, copy, download, fullscreen, live refetch 8s).
- Sempre mostra o compose real gerado a partir do spec.

## Validação
- Compose gerado corretamente a partir de um app (build, ports, networks,
  labels, restart, resources, networks top-level).
- Histórico: cada deploy persiste compose_yaml + compose_hash.
- UI: Monaco renderiza com 0 pageerrors. Suite 7/7.

## Pendências (fase 2)
- Execução de deploy via `docker compose up` (hoje o runtime ainda usa o driver;
  o compose é capturado mas não é o executor).
- Marketplace armazenando templates como DeploymentSpecs.
- Framework detection → spec → Dockerfile otimizado → compose.
- Export Kubernetes/Nomad a partir do mesmo spec.
