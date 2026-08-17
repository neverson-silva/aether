# RFC-0008 — Segurança

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P8, P9
- **Dependências:** RFC-0004, RFC-0006

---

## 1. Objetivo

Definir a arquitetura completa de segurança: rootless, least privilege, secrets, encryption,
SSH, API keys, OIDC/SSO, MFA, audit logs. Segurança por padrão.

## 2. Escopo

**Dentro:** modelo de usuários/processos, criptografia de secrets (KEK/DEK), comunicação
mTLS core↔agent, API keys, RBAC (detalhe em RFC-0018), autenticação (local/OIDC/MFA por fase),
audit, hardening (seccomp/AppArmor/SELinux), supply chain.
**Fora:** políticas de firewall do provedor (são do usuário); auditoria de terceiros.

## 3. Responsabilidades

- Definir e impor isolamento por usuário systemd (rootless).
- Criptografar secrets em repouso e em trânsito.
- Gerir identidade (CA interna, tokens, API keys).
- Autorizar ações (RBAC enforced no core).
- Registrar audit.
- Garantir hardening de units (seccomp, NoNewPrivileges, capabilities).

## 4. Arquitetura

```
[Internet] → TLS → [Proxy rootless (CAP_NET_BIND_SERVICE)]
                     → [App containers rootless]
                     → [Core (aether user)]
                          ├→ [DB (SQLite/PG)]
                          └→ [Agent (mTLS)] → [Runtime OCI rootless]

Chaves: KEK (host/TPM) → DEK → secrets (AES-256-GCM)
CA interna: aether-ca → certs core/agent (mTLS)
```

## 5. Fluxos

### 5.1 Login local

```
1. POST /auth/login (rate-limited)
2. verifica Argon2id hash
3. emite JWT curto + refresh (rotação)
4. audit.login
```

### 5.2 Injeção de secret em app

```
1. deployment resolve refs de secrets
2. agent obtém ciphertext (core) → decript com DEK
3. escreve env file (0600) para a unit
4. unit inicia com env resolvido; zero logs com valor
```

### 5.3 mTLS core↔agent

```
1. instalador gera CA interna
2. core e agent recebem certs assinados
3. conexão gRPC valida cert + CN
4. fail → conexão recusada (audit)
```

## 6. Interfaces

```go
type SecretCrypto interface {
    Encrypt(ctx, plaintext []byte) ([]byte, error) // AEAD
    Decrypt(ctx, ciphertext []byte) ([]byte, error)
    RotateDEK(ctx) error
}
type KeyManager interface { KEK() ([]byte, error) } // host, TPM/HSM
type Identity interface {
    Authenticate(ctx, creds) (Session, error)
    Authorize(ctx, principal, action, resource) error
}
type ApiKeyService interface {
    Create(ctx, scopes []string) (Token, error)  // mostra uma vez
    Verify(ctx, token string) (*ApiKey, error)
}
```

## 7. Eventos

Emitidos: `audit.login`, `audit.rbac_change`, `secret.rotated`, `apikey.created`,
`apikey.revoked`, `security.bruteforce_blocked`.
Consumidos: (cross-cutting) — nenhum domínio é dono.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| App container rootless | sem privilégio no host |
| Secret no banco | cifrado (nunca em claro) |
| API key vazada | revogação imediata |
| Tentativa de força bruta | bloqueio + alerta |
| Compliance | audit exportável |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Rootless | segurança | complexidade de rede |
| KEK/DEK | rotação barata de DEK | gestão de chave |
| mTLS | forte | setup extra |
| MFA em fase 2 | v1 mais simples | risco aceito em v1 |

## 10. Decisões

- **D-001:** rootless em toda a plataforma.
- **D-002:** secrets AES-256-GCM com KEK/DEK.
- **D-003:** core↔agent via mTLS (sem SSH de controle).
- **D-004:** RBAC enforced no core.
- **D-005:** MFA/OIDC em fase 2+.
- **D-006:** audit append-only; sem API de delete.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| KEK no host comprometido | baixa | TPM/HSM (enterprise) |
| Secret em log | baixa | nunca logar valores; scrubbing |
| Rootless rede | média | tuning slirp/pasta documentado |
| mTLS misconfig | baixa | validação de CN + CA pin |

## 12. Alternativas descartadas

- **Docker Engine root**: descartado (superfície crítica; daemon privilegiado).
- **SSH como transporte de controle**: descartado (chaves espalhadas; sem mTLS).
- **Segredos em claro no env**: descartado (viola P9).
- **BCrypt para hash**: descartado (Argon2id é superior).
