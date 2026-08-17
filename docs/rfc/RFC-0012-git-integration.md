# RFC-0012 — Integração Git

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P5, P7, P9
- **Dependências:** RFC-0011 (Deployments)

---

## 1. Objetivo

Definir a integração com sistemas Git (GitHub, GitLab, Bitbucket) como fonte de aplicações,
gatilhos de deploy (webhooks) e preview deployments.

## 2. Escopo

**Dentro:** porta `GitProvider`, autenticação (OAuth/tokens), clone/fetch, detecção de commit,
webhooks de entrada, gatilhos, suporte a preview.
**Fora:** CI completo; UI específica de cada provider.

## 3. Responsabilidades

- Autenticar com providers (OAuth app, PAT).
- Clonar/atualizar fonte em workspace.
- Detectar mudanças (commit/branch/tag) via webhook ou on-demand.
- Emitir eventos de trigger para Deployments.
- Aplicar políticas de branch (deploy só em `main` etc.).
- (fase 2+) Preview por PR/push de branch.

## 4. Arquitetura

```
Git (domínio)
  GitProvider (porta)
   ├── GitHubDriver  (bundled)
   ├── GitLabDriver  (bundled)
   └── BitbucketDriver (bundled)
        │
Webhook listener → valida assinatura → event source.updated
```

## 5. Fluxos

### 5.1 Configurar provider

```
1. usuário conecta (OAuth) → armazena token (cifrado)
2. cria GitSource (repo, branch, path)
3. testa conexão (fetch HEAD)
```

### 5.2 Deploy por push

```
1. webhook (X-Hub-Signature validado) → event webhook.received
2. GitProvider.FetchHEAD → compara ref
3. se branch elegível → source.updated → Deployments.Start
```

### 5.3 Preview

```
1. webhook PR (opened/updated/closed)
2. opens → PreviewDeployment (branch) → build → rota temporária
3. closed → teardown (remove rota+cert)
```

## 6. Interfaces

```go
type GitProvider interface {
    Authorize(ctx, authReq) (TokenRef, error)
    RepoInfo(ctx, ref TokenRef, repo string) (*RepoInfo, error)
    FetchHEAD(ctx, source GitSource, token TokenRef) (CommitInfo, error)
    Clone(ctx, source GitSource, token TokenRef, dest string) error
    VerifyWebhook(ctx, payload, signature string, secret []byte) error
    ListBranches/PRs...
}
```

## 7. Eventos

Emitidos: `webhook.received`, `source.updated`, `commit.detected`, `pr.opened`,
`pr.closed`, `git.auth_failed`.
Consumidos: (Deployments) `source.updated`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| push em main | auto-deploy |
| PR aberto | preview subido |
| PR fechado | preview removido |
| token expirado | alerta `git.auth_failed` |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| OAuth + PAT suportados | flexibilidade | gestão de tokens |
| Webhook validado | segurança | setup de webhook |
| On-demand fetch (sem polling) | P2 | latência entre push e webhook (mínima) |

## 10. Decisões

- **D-001:** GitHub/GitLab/Bitbucket em v1; Gitea via plugin.
- **D-002:** webhooks validados (HMAC); sem polling de providers.
- **D-003:** fetch sob demanda (deploy) + webhook.
- **D-004:** política de branch configurável.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Webhook spoofing | baixa | validação de assinatura |
| Token revogado | baixa | alerta + re-auth |
| Rate limit do provider | baixa | cache + fetch sob demanda |

## 12. Alternativas descartadas

- **Polling de repos**: descartado (P2).
- **Git via SSH com chave por app**: descartado em v1 (gestão de chaves); reavaliar.
- **Um provider só**: descartado (paridade).
