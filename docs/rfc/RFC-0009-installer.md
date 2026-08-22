# RFC-0009 — Instalador

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P2, P6, P9
- **Dependências:** —

---

## 1. Objetivo

Definir o instalador inteligente (servidor limpo), o modelo de atualização, rollback e
desinstalação.

## 2. Escopo

**Dentro:** bootstrap, detecção de ambiente, instalação de dependências mínimas, provisionamento
(usuários/units/identidade), init do banco, update, rollback, uninstall.
**Fora:** gerenciamento de apps; UI.

## 3. Responsabilidades

- Rodar em servidor limpo e deixar a plataforma pronta.
- Detectar distro/arch/init; instalar só o que falta.
- Criar usuários/units/diretórios; gerar identidade (CA, KEK, token).
- Inicializar banco + migrações.
- Executar `update` com backup, migração e rollback automático.
- Desinstalar de forma limpa (com `--purge` opcional).

## 4. Arquitetura

```
curl script (bootstrap mínimo)
   │
   ▼
aether install   (binário)
   ├── Fase detect & plan
   ├── Fase install deps (podman/crun/conmon/buildah/skopeo/fuse-overlayfs)
   ├── Fase provision (users/units/identity)
   ├── Fase init (api/db/migrations/admin token)
   └── Fase done (URL + token)
```

## 5. Fluxos

### 5.1 Instalação (passo a passo)

Ver [`15-installer.md`](../15-installer.md) §3 para o fluxo completo. Resumo de eventos:
`install.started`, `install.deps_ok`, `install.provisioned`, `install.init`, `install.complete`.

### 5.2 Atualização

```
1. `aether update`
2. baixa binário (assinado) para staging; valida checksum
3. backup automático (SQLite VACUUM INTO + event log)
4. swap atômico do binário
5. migrações transacionais
6. recarrega units; restart core
7. propaga para agentes (event agent.updated)
8. falha → rollback automático
```

### 5.3 Rollback de versão

```
aether rollback [versão]
→ restaura binário anterior
→ aplica migração reversa
→ recarrega units
```

## 6. Interfaces

```go
type Installer interface {
    Plan(ctx) (*InstallPlan, error)        // o que vai ser feito
    Install(ctx, plan) error
    Update(ctx, target string) (*UpdateResult, error)
    Rollback(ctx, version string) error
    Uninstall(ctx, purge bool) error
    Status(ctx) (*InstallStatus, error)
}
```

## 7. Eventos

Emitidos: `install.started`, `install.complete`, `update.started`, `update.complete`,
`update.failed`, `rollback.complete`.
Consumidos: (poucos) — `agent.updated`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| VPS limpa Debian | plataforma pronta < 2 min |
| Upgrade 0.1→0.2 | zero-downtime, backup automático |
| Update falha | rollback automático |
| Migrar de distro | detecção + instalação correta |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Bash bootstrap + binário | mínimo, confiável | script simples não faz tudo |
| Swap atômico de binário | simples/seguro | precisa staging dir |
| Backup automático pré-update | segurança | SSD adicional temporário |
| Autoupdate off default | P8 | usuário precisa atualizar |

## 10. Decisões

- **D-001:** bootstrap bash mínimo + `aether install`.
- **D-002:** zero download de imagem na instalação.
- **D-003:** update com backup, migração transacional e rollback automático.
- **D-004:** autoupdate off por padrão.
- **D-005:** uninstall preserva units de apps (não remove dados de app por padrão).

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Distro sem versão mínima | média | binários estáticos |
| Portas ocupadas | média | detectar e orientar |
| Update corrompe banco | baixa | backup + transações + rollback |
| SELinux | baixa | policy customizada |

## 12. Alternativas descartadas

- **Ansible/Terraform na instalação**: descartado (dependência externa; overkill).
- **Instalação via imagem Docker**: descartado (viola P2/P4; downloads).
- **Package manager (apt/dnf) para a plataforma**: descartado (atualização lenta; multi-distro).
