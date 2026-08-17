# RFC-0016 — Marketplace e Templates

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P2, P6, P10
- **Dependências:** RFC-0011 (Deployments)

---

## 1. Objetivo

Definir o **Marketplace** de one-click apps e o sistema de **Templates** (modelos
parametrizáveis de aplicações).

## 2. Escopo

**Dentro:** catálogo de templates, formato de template (versão, parâmetros), instanciação,
categories, busca (FTS5), atualização de catálogo.
**Fora:** build da aplicação (delega); UI (consome API).

## 3. Responsabilidades

- Manter catálogo de templates versionado e imutável por versão.
- Definir schema de template (inputs validados, defaults).
- Instanciar template → spec de Application (image/compose/build).
- Buscar/filtrar (SQLite FTS5).
- Atualizar catálogo (pull de manifestos) de forma transacional.

## 4. Arquitetura

```
Marketplace (domínio)
  TemplateCatalog (registry local)
   ├── TemplateDefinition (schema YAML)
   ├── Category
   └── FTS5 index (busca)
        │
  Instanciação → Application spec → Deployments
```

## 5. Fluxos

### 5.1 Instalar template

```
1. lista catálogo → escolhe template + version
2. preenche inputs (validação por schema)
3. instancia → spec Application (image/ports/env/volumes)
4. cria app → deploy (RFC-0011)
5. template.installed
```

### 5.2 Atualizar catálogo

```
1. pull de manifestos do registry oficial
2. valida assinatura/checksum
3. merge transacional (novas versões, sem quebrar instaladas)
4. template.updated
```

## 6. Interfaces

```go
type Template struct {
    ID string; Name string; Category string
    Version string; Schema json.RawMessage
    AppSpec AppTemplateSpec
}

type TemplateService interface {
    List(ctx, filter) ([]Template, error)
    Get(ctx, id, version) (*Template, error)
    Search(ctx, query) ([]Template, error)
    Install(ctx, req InstallTemplateRequest) (*Application, error)
    Refresh(ctx) error
}
```

## 7. Eventos

Emitidos: `template.installed`, `template.updated`, `marketplace.refreshed`.
Consumidos: (deploy) `app.created`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| One-click app | app criado e deployado |
| Template parametrizado | inputs validados |
| Buscar "wordpress" | resultados FTS5 |
| Update de catálogo | versões novas, instáveis preservadas |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Templates versionados imutáveis | reprodutibilidade | precisa ciclo de publicação |
| FTS5 embutido | zero serviço | recursos de busca limitados (ok) |
| Catálogo central opcional | usuário pode ter próprio | coordenação de distribuição |

## 10. Decisões

- **D-001:** formato YAML de template com schema (inputs validados).
- **D-002:** catálogo versionado e imutável por versão.
- **D-003:** busca FTS5 (SQLite).
- **D-004:** catálogo pode ser estendido por usuário (arquivos) e plugin.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Template quebrado | baixa | validação em CI de catálogo |
| Instalação incompatível | baixa | schema + dry-run |
| Catálogo cresce | baixa | index FTS5 + paginação |

## 12. Alternativas descartadas

- **Marketplace multi-vendor com pagamento**: descartado (fase 4+).
- **Meilisearch**: descartado (P7; FTS5 basta).
- **Templates não versionados**: descartado (reprodutibilidade).
