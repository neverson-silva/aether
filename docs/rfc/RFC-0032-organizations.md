# RFC-0032 — Enterprise Organizations & Multi-Tenancy

## Status: Implemented (backend core + org switcher)

## Objective
Tornar a plataforma organization-centric, com multi-tenancy real:

- Organizations com owner único, roles RBAC (owner/admin/member/viewer) e Global Admin.
- Restrição de acesso por projeto (project assignments): Member/Viewer enxergam
  apenas projetos atribuídos.
- Troca de organização sem novo login (header `X-Aether-Org`).
- Audit log por organização.
- Personal Organization automática no primeiro login ("<Name>'s Organization").

## Arquitetura

```
Organization
    └── Project (organization_id) → Service → Deployments/Domains/Volumes/Env/Logs
```

- Projetos e Apps já possuíam `org_id`; a migração enriquece o modelo.
- Tokens continuam compactos (userId + org default + globalRole); a organização
  ativa de cada request é resolvida no middleware `auth` via `X-Aether-Org`
  e validada contra `members`.

## Migração 21 (idempotente, ADD COLUMN IF NOT EXISTS)
- `orgs`: + slug, description, avatar, color, updated_at, deleted_at (soft delete).
- `users`: + global_role.
- `projects`: + slug, description, color, updated_at, deleted_at.
- `project_assignments(org_id, user_id, project_id)`.
- `audit_logs(org_id, user_id, action, resource_type, resource_id, details)`.
- Backfill: slug gerado a partir do nome; projetos existentes já possuem org_id
  (nenhuma perda de dado).

## Permissão (avaliação centralizada)
1. Global ADMIN → ALLOW.
2. Não é membro da org → DENY.
3. Owner → ALLOW.
4. Member/Viewer → apenas projetos atribuídos (`project_assignments`).

Aplicado em:
- `auth` middleware (valida membership + resolve org ativa).
- `projectForOrg` / `canAccessProject` / `appForOrg` (handlers de projeto/service).
- `handleListProjects` (filtra projetos atribuídos para Member/Viewer).

## API
- `GET/POST /api/v1/organizations`, `GET/PATCH/DELETE /organizations/:id`
- `GET/POST /organizations/:id/members`, `PATCH/DELETE /members/:userId`
- `POST/DELETE /organizations/:id/members/:userId/projects/:projectId`
- `GET /organizations/:id/audit`
- `GET /api/v1/me` agora retorna `organizations` (multi-org) + `global_role`.

## Frontend
- `OrgProvider` (context) + `OrgSwitcher` (dropdown estilo GitHub) no shell.
- API client envia `X-Aether-Org` automaticamente; trocar org invalida o cache
  (React Query) e a tela de projetos atualiza sem reload.
- Páginas `/organizations/new` e `/organizations/$id` (Overview/Members/Projects/Audit).
- Invite member com role + selector de projetos.

## Testes
- `TestMultiTenancyProjectScoping`: cria org/projeto, convida membro com
  atribuição, valida que o membro vê apenas o projeto atribuído e leva 403 em
  org estranha.
- Migração idempotente: testes de banco com versão 21.

## Pendências futuras
- URL `/org/:slug` + breadcrumb hierárquico.
- Convites pendentes (invitation table), transferência de ownership.
- Caching de membership (atualmente 1 query por request), quotas/billing.
