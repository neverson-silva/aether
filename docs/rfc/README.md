# RFCs — Diretório de Request for Comments

Cada grande módulo da plataforma possui uma RFC própria. RFCs são o **instrumento de decisão
arquitetural**: são revisadas, aprovadas e versionadas antes da implementação.

## Template

Todo novo RFC deve usar o template em [`RFC-0000-template.md`](RFC-0000-template.md).

## Ciclo de vida de uma RFC

1. **Draft** — proposta inicial, aberta a revisão.
2. **Review** — em discussão (comentários no PR/issue).
3. **Accepted** — aprovada; vira referência canônica.
4. **Implemented** — código corresponde à RFC (checagem em CI/docs).
5. **Superseded** — substituída por RFC mais nova (apontar para a nova).

## Índice de RFCs

| RFC | Título | Status | Depende de |
|-----|--------|--------|-----------|
| [`RFC-0001`](RFC-0001-execution-engine.md) | Execution Engine | Draft | — |
| [`RFC-0002`](RFC-0002-networking-engine.md) | Networking Engine | Draft | RFC-0003 |
| [`RFC-0003`](RFC-0003-certificate-engine.md) | Certificate Engine | Draft | — |
| [`RFC-0004`](RFC-0004-persistence.md) | Persistência | Draft | — |
| [`RFC-0005`](RFC-0005-event-bus.md) | Event Bus | Draft | RFC-0004 |
| [`RFC-0006`](RFC-0006-runtime-driver-api.md) | Runtime Driver API | Draft | RFC-0001 |
| [`RFC-0007`](RFC-0007-observability.md) | Observabilidade | Draft | RFC-0005 |
| [`RFC-0008`](RFC-0008-security.md) | Segurança | Draft | RFC-0004, RFC-0006 |
| [`RFC-0009`](RFC-0009-installer.md) | Instalador | Draft | — |
| [`RFC-0010`](RFC-0010-plugin-system.md) | Plugin System | Draft | — |
| [`RFC-0011`](RFC-0011-deployments.md) | Pipeline de Deployments | Draft | RFC-0001, RFC-0005 |
| [`RFC-0012`](RFC-0012-git-integration.md) | Integração Git | Draft | RFC-0011 |
| [`RFC-0013`](RFC-0013-databases.md) | Databases Gerenciadas | Draft | RFC-0001 |
| [`RFC-0014`](RFC-0014-backup-restore.md) | Backup e Restore | Draft | RFC-0004 |
| [`RFC-0015`](RFC-0015-multiserver.md) | Multi Server e Clusters | Draft | RFC-0005, RFC-0006 |
| [`RFC-0016`](RFC-0016-marketplace.md) | Marketplace e Templates | Draft | RFC-0011 |
| [`RFC-0017`](RFC-0017-api-cli.md) | API e CLI | Draft | — |
| [`RFC-0018`](RFC-0018-rbac.md) | Organizations, Teams e RBAC | Draft | — |

## Regras

- RFC **Accepted** não pode ser alterada silenciosamente; mudanças viram nova versão ou nova RFC.
- Implementação de módulo sem RFC Accepted não é permitida.
- Cada RFC cita os princípios do manifesto que atende (ver [`../00-manifesto.md`](../00-manifesto.md)).
