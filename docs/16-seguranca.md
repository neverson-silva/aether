# 16 — Segurança

> **Status:** Arquitetura de segurança.
> **RFC relacionada:** [`RFC-0008`](rfc/RFC-0008-security.md)
> **Princípios atendidos:** P8 (dados do usuário), P9 (segurança por padrão), P1.

---

## 1. Propósito

Definir a arquitetura completa de segurança: rootless, least privilege, secrets, encryption,
SSH, API keys, OIDC/SSO, MFA, audit logs. A segurança é **por padrão**, não opcional.

## 2. Modelo de confiança

```
Trust boundaries:
  [Internet] → (TLS) → [Proxy rootless] → (rede interna) → [App containers rootless]
                          └→ [Core] → [DB SQLite/PG]
                                          └→ [Agent] → [Runtime OCI rootless]

Processos de plataforma: sem privilégio, isolados por usuário systemd.
Comunicação core↔agent: mTLS (CA interna).
Segredos: criptografados em repouso; chave mestra protegida.
```

## 3. Rootless por padrão

| Componente | Usuário | Privilégio |
|-----------|---------|-----------|
| Core | `aether` (system, no-shell) | sem root |
| Agent | `aether-agent` | sem root |
| Proxy | `aether-proxy` | sem root; `CAP_NET_BIND_SERVICE` (80/443) via systemd |
| App containers | usuário de app (ou user instance) | rootless podman |
| Build | `aether-agent` | rootless buildah |

**Por quê:** rootless reduz a superfície de ataque (exploit em container não vira root no host),
é o padrão moderno de segurança, e elimina daemon privilegiado (D2/D12 mitigados).

## 4. Least privilege

- Cada processo tem usuário próprio e diretórios próprios (nunca `chmod 777`).
- **Filesystem**: `0600` para chaves/secrets; `0700` para diretórios privados.
- **Capabilities**: `CapabilityBoundingSet` vazio para apps; proxy apenas net_bind_service.
- **`NoNewPrivileges=yes`** nas units.
- **seccomp** default-deny para apps de plataforma; **AppArmor/SELinux** profiles quando a distro
  suporta.
- **Process namespace**: agentes separados; apps em namespaces isolados.

## 5. Secrets

### 5.1 Modelo de criptografia

```
KEK (master key, do host) → criptografa DEK (data encryption key) → criptografa segredos
```

- **KEK**: derivada de entropia do host (`/var/lib/aether/keys/master.key`, `0600`, só root/aether)
  ou de HSM/TPM (enterprise). Nunca em texto.
- **DEK**: aleatória, criptografada pela KEK; pode ser rotacionada sem re-criptografar tudo.
- Algoritmo: **AES-256-GCM**; nonce único por cifra; AEAD para integridade.
- **Em repouso**: secrets armazenados cifrados no banco.
- **Em trânsito**: TLS 1.3; mTLS core↔agent.

### 5.2 Ciclo de vida de secret

- Criação: valor chega cifrado da UI/API (nunca em log).
- Referência: apps referenciam `secret_ref` (nome), nunca o valor.
- Injeção em apps: o agent resolve o ref e escreve o env com permissões restritas; não fica
  em texto em logs.
- Rotação: nova DEK (se necessário) ou novo valor; eventos `secret.rotated`.
- Revogação: remover ref; agent remove o env da unit.

### 5.3 Secret store plugável

- Default: tabela cifrada no banco (SQLite/PG).
- Plugin `SecretStore`: Vault, KMS (futuro) — via porta.

## 6. SSH

- **Nunca** ssh da plataforma para o próprio host.
- Multi-server (fase 3+): conexão core↔agent **sem SSH**: agente com mTLS (outbound) +
  gRPC. SSH **não** é usado como transporte de controle (evita chaves espalhadas).
- SSH do usuário: opcional para acesso a servidores remotos na UI/CLI (com chave gerenciada).

## 7. API Keys

- Geradas com alta entropia (`sk_...`), **hasheadas** (Argon2id) no banco; valor mostrado
  uma única vez.
- Escopos: per-user/per-org; permissões RBAC aplicadas.
- Expiração configurável; rotação; revogação imediata.
- Audit de uso.

## 8. Autenticação e Identidade

| Recurso | Fase |
|---------|------|
| Password local (Argon2id) | v1 |
| JWT curto + refresh com rotação | v1 |
| OIDC/SSO (via plugin IdpProvider) | fase 2+ |
| MFA (TOTP, WebAuthn) | fase 2+ |
| SAML (enterprise) | fase 5 |

- Rate limit de login; lockout; notificação de novos devices.
- Sessões revogáveis por usuário/org.

## 9. RBAC

Modelo: **User → Organization → Role → Permissions**.

| Papel (org) | Permissões típicas |
|-------------|--------------------|
| Owner | tudo, billing, RBAC, destroy |
| Admin | gerenciar apps, membros, backups, certs |
| Developer | criar/edit deploy; logs; metrics |
| Viewer | read-only |
| Custom | conjunto de permissões granulares |

Permissões granulares: `app.deploy`, `app.config`, `secrets.read`, `backup.create`,
`cert.manage`, `org.members`, etc.

**Decision:** RBAC avaliado no Core (authorization service), nunca no frontend; API enforce
sempre (defense in depth).

## 10. Rede e hardening de sistema

- Proxy é único ponto de entrada (80/443). Sem porta de gerenciamento exposta por padrão.
- UI/API em `:8000` (loopback) ou via domínio com auth — nunca exposta sem TLS por padrão.
- `sysctl` hardening (net.ipv4.conf.*, etc.) documentado.
- Seccomp/AppArmor/SELinux aplicados quando disponíveis.
- DNS/ACME com validação (reject non-LE challenge hosts).

## 11. Audit Logs

- Tabela append-only `audit_events`.
- Captura: autenticação, RBAC, deploys, restores, certs, secrets (sem valores), configs.
- Protegida: sem API de UPDATE/DELETE; exportação para compliance.

## 12. Supply chain e releases

- Binários assinados (cosign/sigstore); checksum publicado.
- Plugins assinados.
- Reproducible builds (CI).
- SBOM publicado por release.
- Scans de vulnerabilidade em dependências (CI + releases).

## 13. Resposta a incidentes (plano de prontidão)

- Alertas de segurança por evento (ex.: muitas falhas de login).
- Capacidade de revogar tokens/keys instantaneamente.
- Procedimento de rotação de KEK/DEK documentado.
- Backup de estado como ponto de restauração forense.

## 14. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Chave mestra no host comprometida | TPM/HSM (enterprise), rotação, backups cifrados |
| Rootless com rede complexa | documentação + tuning de slirp/pasta |
| Secret vaza em logs | política: nunca logar valores; scrubbing no log sink |
| mTLS config errado | validação de identidade por hostname+CA pin |
| DoS no proxy | rate limit default, limites de conexão |

## 15. Decisões

- **Decisão:** rootless em toda a plataforma.
- **Decisão:** least privilege por usuário de sistema + seccomp/AppArmor.
- **Decisão:** secrets cifrados (AES-256-GCM, KEK/DEK), nunca em claro.
- **Decisão:** comunicação core↔agent via mTLS (sem SSH de controle).
- **Decisão:** RBAC enforced no core; audit append-only.
- **Decisão:** MFA/OIDC em fase 2+, não bloqueiam v1.

## 16. Referências

- RFC-0008 (Security).
- Certificados: [`09-certificate-engine.md`](09-certificate-engine.md).
- Instalador (identidade inicial): [`15-installer.md`](15-installer.md).
