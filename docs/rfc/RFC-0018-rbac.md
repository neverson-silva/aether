# RFC-0018 — Organizations, Teams e RBAC

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P9, P10
- **Dependências:** —

---

## 1. Objetivo

Definir o modelo multi-tenant: **Organizations**, **Teams**, **Users** e **RBAC** com
permissões granulares, enforced no core.

## 2. Escopo

**Dentro:** hierarquia org/team/user, papéis, permissões granulares, escopos de API key,
invites, auditoria de RBAC, políticas de isolamento entre orgs.
**Fora:** OIDC/SSO detalhado (fase 2+, referência em RFC-0008); billing.

## 3. Responsabilidades

- Modelar User/Organization/Team/RoleBinding.
- Avaliar permissões (authorize) no core (toda mutação).
- Isolar dados entre orgs (tenant).
- Gerenciar invites e associação.
- Registrar mudanças de RBAC em audit.
- Definir escopos de API key por permissão.

## 4. Arquitetura

```
User ──► Organization ──► Team
            │
        RoleBinding (user/team → role)
            │
      Permissions (granulares)
            │
   authorize() enforced no core
```

## 5. Fluxos

### 5.1 Criar org e adicionar membro

```
1. owner cria org → org.created
2. invite → e-mail → aceita → member
3. atribui papel (admin/developer/viewer/custom)
4. audit.rbac_change
```

### 5.2 Mutação com RBAC

```
1. POST /api/v1/apps/:id/deploy
2. core: resolve principal → bindings → permissions
3. authorize("app.deploy", app) → ok/403
4. executa; audit
```

## 6. Interfaces

```go
type RBACService interface {
    Authorize(ctx, principal Principal, action Action, resource Resource) error
    ListPermissions(ctx, role Role) ([]Permission, error)
    Bind(ctx, principal Principal, role Role, scope Scope) error
}

type Principal struct { UserID string; OrgID string; TeamIDs []string }
type Permission struct { Resource string; Action string }
type Scope struct { Type string; ID string } // org-wide, project, app
```

## 7. Eventos

Emitidos: `user.invited`, `member.joined`, `role.changed`, `org.created`, `team.created`,
`apikey.scope_changed`, `rbac.changed`.
Consumidos: (cross-cutting) — authorize é síncrono; eventos alimentam audit/timeline.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| Viewers não modificam | 403 |
| Developer não gerencia membros | 403 |
| API key com escopo deploy-only | restrição efetiva |
| Org A não vê Org B | isolamento por tenant |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| RBAC granular | flexibilidade | complexidade de mapear permissões |
| Enforce no core | segurança | nunca confiar no client |
| Papéis custom | poder | gestão por admins |

## 10. Decisões

- **D-001:** hierarquia User → Organization → Team.
- **D-002:** papéis padrão (owner/admin/developer/viewer) + custom.
- **D-003:** authorize() síncrono no core; nunca no frontend.
- **D-004:** API keys com escopos derivados de permissões.
- **D-005:** isolamento por tenant em todas as queries.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Escalada de privilégio | baixa | autorização sempre no core; auditoria |
| Org data leak | baixa | tenant filter obrigatório em repositórios |
| Confusão de papéis | baixa | docs + defaults seguros |

## 12. Alternativas descartadas

- **RBAC por recurso apenas (sem org)**: descartado (multi-tenant).
- **Casbin/Permify**: considerados; avaliação decidiu implementação própria leve em v1.
- **Autorização no frontend**: descartado (inseguro).
