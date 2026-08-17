# 14 — Plugin System

> **Status:** Extensibilidade.
> **RFC relacionada:** [`RFC-0010`](rfc/RFC-0010-plugin-system.md)
> **Princípios atendidos:** P1, P7, P10, P9.

---

## 1. Propósito

Um sistema de plugins completo e desacoplado. **Nada é obrigatório. Tudo é carregado sob
demanda.** Plugins implementam as **portas** da plataforma (git, DNS, cloud, storage,
monitoramento, identidade, etc.) sem nunca alterar o core.

## 2. Filosofia

- **Core não conhece plugins**: o core define portas; plugins são adaptadores.
- **Sob demanda**: plugin só é baixado/ativado quando o usuário configura o provider.
- **Seguro**: plugins de terceiros rodam em sandbox com permissões limitadas.
- **Auditável**: manifestos assinados; versão fixada.

## 3. Tipos de plugin

| Tipo | Descrição | Custo | Uso |
|------|-----------|-------|-----|
| **Native** | compilado dentro da plataforma (built-in) | ~0 | providers essenciais (Let's Encrypt, Traefik, podman, SQLite) |
| **Bundled** | entregue com o binário, ativado sob demanda | tamanho do código | GitHub, GitLab, Cloudflare, Hetzner, S3, e-mail... |
| **External** | plugin de terceiros (binário/bundle assinado) | processo sandbox | extensões da comunidade |

**Decisão v1:** providers de alta frequência são Native/Bundled (zero custo extra); third-party
é External com sandbox. Nenhum processo extra em idle (plugins não ativos não rodam nada).

## 4. Portas que plugins podem implementar

| Porta | Plugins exemplos |
|-------|------------------|
| `GitProvider` | GitHub, GitLab, Bitbucket, Gitea |
| `DnsProvider` | Cloudflare, Route53, Hetzner DNS, Google Cloud DNS, Porkbun |
| `CloudProvider` | Hetzner, AWS, Azure, GCP, Vultr, DigitalOcean |
| `ObjectStore` | S3, R2, GCS, MinIO, local |
| `NotifyChannel` | e-mail, Slack, Discord, Telegram, webhook |
| `IdpProvider` | OIDC, SAML, Google, GitHub, Entra ID |
| `AiProvider` | OpenAI, Anthropic, local LLM |
| `McpProvider` | Model Context Protocol servers |
| `MonitoringExporter` | OTLP, Prometheus remote write, StatsD |
| `AcmeProvider` | Let's Encrypt, ZeroSSL, Private CA |
| `ProxyProvider` | Traefik, Caddy, NGINX, Envoy |
| `RuntimeDriver` | Podman, Docker, containerd, Kubernetes |

## 5. Arquitetura do host de plugins

```
Core (pluginhost)
  ├── Registry        (catálogo de plugins, versões, manifestos)
  ├── Resolver        (baixa/instala sob demanda)
  ├── Loader          (native: link dinâmico/símbolos; external: subprocess)
  ├── Sandbox         (external: seccomp, rlimits, namespaces, no root)
  ├── Permissions     (declaradas no manifesto; aprovação do admin)
  └── Lifecycle       (enable/disable/upgrade; eventos)
```

### 5.1 Manifesto de plugin

```yaml
id: plugins.cloudflare-dns
version: 1.4.0
runtime: bundled   # native | bundled | external
port: DnsProvider
permissions: [network:443, secrets:read-cloudflare-token]
signature: <assinatura da chave da plataforma/editor>
```

### 5.2 Carregamento

- Ativação: configuração de um provider (ex.: adicionar Cloudflare DNS) → resolver baixa →
  loader carrega → permissões validadas → plugin registra handlers de porta.
- Desativação: remover provider → plugin descarrega → zero processo/RAM/CPU.

## 6. Segurança de plugins

- **Trust**: plugins oficiais assinados com chave da plataforma; third-party assinados por
  editor verificado.
- **Permissões mínimas**: manifesto declara o que precisa; admin aprova.
- **Sandbox externo**: subprocess com `NoNewPrivileges`, seccomp default-deny, rlimits,
  PID/network namespaces quando aplicável.
- **Timeout e retry** nas chamadas de plugin.
- **Audit**: `plugin.installed`, `plugin.enabled`, `plugin.permission_granted`.

## 7. MCP (Model Context Protocol)

- Plugins MCP permitem conectar ferramentas LLM à plataforma (ex.: assistente que consulta
  deployments).
- Implementado como plugin `McpProvider` (cliente MCP embutido) — sob demanda.

## 8. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Plugin malicioso | assinatura, sandbox, permissões, revisão |
| Plugin com bug derruba core | external rodando fora do processo core |
| Dependência de plugin bloqueia core | portas com timeout/circuit breaker; core nunca depende de plugin para funções essenciais |
| Version drift | manifesto com semver; compatibility matrix |
| Crescimento de disco (plugins) | diretório com quota + GC |

## 9. Decisões

- **Decisão:** portas como contrato; plugins nunca tocam o core.
- **Decisão:** Native/Bundled para essenciais; External com sandbox para terceiros.
- **Decisão:** carregamento 100% sob demanda; nada roda em idle.
- **Decisão:** manifestos assinados e permissões declaradas.

## 10. Referências

- RFC-0010 (Plugin System).
- Portas: [`05-arquitetura-geral.md`](05-arquitetura-geral.md) §2.3.
- Segurança: [`16-seguranca.md`](16-seguranca.md).
