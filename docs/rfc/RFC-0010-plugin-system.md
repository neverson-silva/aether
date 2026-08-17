# RFC-0010 — Plugin System

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P7, P9, P10
- **Dependências:** —

---

## 1. Objetivo

Definir o sistema de plugins: extensível, desacoplado, carregado sob demanda, seguro.
**Nada é obrigatório; tudo carregado sob demanda.**

## 2. Escopo

**Dentro:** tipos de plugin (native/bundled/external), host, manifesto, permissões, sandbox,
lifecycle, segurança (assinatura), catálogo.
**Fora:** conteúdo dos plugins (são providers); UI de marketplace de plugins.

## 3. Responsabilidades

- Definir contratos de plugins por porta (GitProvider, DnsProvider, CloudProvider, etc.).
- Carregar/descarregar plugins sob demanda.
- Validar manifesto, assinatura e permissões.
- Executar plugins externos em sandbox.
- Rastrear lifecycle (install/enable/disable/upgrade) com eventos e audit.
- Gerenciar diretório de plugins com quota + GC.

## 4. Arquitetura

```
Core (pluginhost)
  ├── Registry      (catálogo, versões, manifestos)
  ├── Resolver      (baixa/instala)
  ├── Loader        (native/bundled: símbolos; external: subprocess)
  ├── Sandbox       (seccomp, rlimits, namespaces)
  ├── Permissions   (manifesto + aprovação admin)
  └── Lifecycle     (enable/disable/upgrade; eventos)

Portas (contratos): Git | DNS | Cloud | ObjectStore | Notify | IdP | AI | MCP | Monitor | ACME | Proxy | Runtime
```

## 5. Fluxos

### 5.1 Instalar e ativar (sob demanda)

```
1. usuário configura provider (ex.: Cloudflare DNS)
2. Resolver localiza plugin no catálogo
3. verifica assinatura
4. carrega (native: mem; external: subprocess sandbox)
5. valida permissões declaradas vs aprovadas
6. registra handlers de porta
7. plugin.enabled + audit
```

### 5.2 Desativar

```
1. remover provider → Lifecycle.disable
2. unload (external: termina processo)
3. zero RAM/CPU/processo
```

## 6. Interfaces

```go
type PluginManifest struct {
    ID string; Version string; Port string
    Runtime string // native | bundled | external
    Permissions []string
    Entrypoint string // external
}

type Plugin interface {
    Manifest() PluginManifest
    Init(ctx PluginContext) error
    Close(ctx) error
}

type PluginContext interface {
    Store() KVStore          // storage do plugin (quota)
    Secrets() SecretReader   // acesso a secrets aprovados
    Log() Logger
}
```

## 7. Eventos

Emitidos: `plugin.installed`, `plugin.enabled`, `plugin.disabled`, `plugin.upgraded`,
`plugin.permission_granted`, `plugin.sandbox_violation`.
Consumidos: `provider.configured`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Adicionar GitHub | plugin GitProvider ativo |
| Adicionar Cloudflare DNS | wildcard funcionando |
| Plugin de terceiros | roda em sandbox com permissões |
| Remover provider | plugin descarrega (zero custo) |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Native/Bundled para essenciais | zero custo, confiável | código no core |
| External sandbox | extensibilidade segura | overhead de subprocess (só quando ativo) |
| Manifesto + permissões | segurança | processo de aprovação |
| Sob demanda | zero idle | primeira ativação tem latência |

## 10. Decisões

- **D-001:** portas como contrato; core nunca depende de plugin.
- **D-002:** essenciais = native/bundled; terceiros = external.
- **D-003:** carregamento 100% sob demanda.
- **D-004:** manifestos assinados; permissões declaradas e aprovadas.
- **D-005:** sandbox para external (seccomp, rlimits, no-root).

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Plugin malicioso | baixa | assinatura, sandbox, permissões |
| Bug de plugin derruba core | baixa | external fora do processo core |
| Version drift | baixa | semver + compat matrix |
| Disco cresce | baixa | quota + GC |

## 12. Alternativas descartadas

- **WASM para todos os plugins**: descartado (maturidade; perf para alguns casos; considerar futuro).
- **Carregar tudo na inicialização**: descartado (viola P2/sob demanda).
- **Scripts sem sandbox**: descartado (inseguro).
