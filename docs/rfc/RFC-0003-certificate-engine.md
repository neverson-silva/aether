# RFC-0003 — Certificate Engine

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P2, P5, P9
- **Dependências:** —

---

## 1. Objetivo

Definir o **Certificate Engine**: componente soberano e único para emissão, renovação,
revogação, histórico, auditoria e alertas de certificados TLS. Let's Encrypt e outros provedores
ACME são apenas providers.

## 2. Escopo

**Dentro:** contas ACME, desafios (HTTP-01/DNS-01), ordem e emissão, renovação (D-30 + jitter),
revogação, histórico, alertas, storage criptografado de chaves privadas.
**Fora:** configuração do proxy (o proxy apenas consome refs); DNS pública.

## 3. Responsabilidades

- Gerenciar contas ACME por email/CA.
- Criar ordens e resolver desafios.
- Persistir cert bundle (chave privada criptografada).
- Agendar renovações com jitter; retries com backoff; alertas.
- Servir refs de certs ao Networking/Proxy.
- Manter histórico auditável (issued/renewed/revoked/failed).
- Revogação (manual/automática).

## 4. Arquitetura

```
Certificates (domínio)
      │
CertificateEngine
  ├── PolicyManager (renew window, retries, retenção)
  ├── AccountManager
  ├── ChallengeRunner (http-01 via proxy; dns-01 via plugin)
  ├── Store (certs cifrados; histórico)
  └── Notifier
      │
      └── porta AcmeProvider / DnsProvider
            ├── Let's Encrypt (v1)
            ├── ZeroSSL (futuro)
            └── Private CA (enterprise)
```

## 5. Fluxos

### 5.1 Emissão HTTP-01

```
1. ensure(domain)
2. resolve provider+conta
3. Order(domain) → challenges
4. informar proxy: rota temporária /.well-known/acme-challenge/{token}
5. ACME valida → Order ready
6. baixa bundle → cifra → persiste
7. refs para proxy (memória)
8. cert.issued
```

### 5.2 Renovação

```
1. timer (D-30) com jitter por domínio
2. Order nova → desafio → emitir
3. sucesso → cert.renewed; troca refs no proxy
4. falha → retries (1h,2h,4h,...) → após N falhas cert.renewal_failed + alerta
5. perto do vencimento: renew agressivo (diário)
```

### 5.3 DNS-01 (wildcard)

```
1. Order wildcard → challenge dns-01
2. DnsProvider.UpsertTxtRecord
3. aguardar propagação (poll curto com timeout)
4. ACME valida → emitir
5. cleanup: DeleteTxtRecord
```

## 6. Interfaces

```go
type AcmeProvider interface {
    Register(ctx, email) (Account, error)
    NewOrder(ctx, domains []string) (OrderID, error)
    GetChallenges(ctx, orderID) ([]Challenge, error)
    CompleteChallenge(ctx, ch Challenge, proof ChallengeProof) error
    FetchCertificate(ctx, orderID) (CertBundle, error)
    Revoke(ctx, bundle CertBundle, reason RevokeReason) error
}

type DnsProvider interface {
    UpsertTxtRecord(ctx, name, value string, ttl int) error
    DeleteTxtRecord(ctx, name, value string) error
}

type CertBundle struct {
    Domain string; CertPEM []byte; KeyPEMEncrypted []byte; ChainPEM []byte
    NotBefore, NotAfter time.Time; Issuer string
}
```

## 7. Eventos

Emitidos: `cert.issued`, `cert.renewing`, `cert.renewed`, `cert.renewal_failed`,
`cert.revoked`, `cert.expiring_soon`.
Consumidos: `domain.added`, `domain.removed`, `tls.force_renew`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| app.com em app | cert válido em < 2 min |
| *.app.com | wildcard via DNS-01 |
| Falha de renovação | alerta + retries + renew agressivo |
| Comprometimento | revogação imediata + rotação |
| Migração de proxy | certs preservados (engine soberano) |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Engine soberano | controle total; auditoria; multi-proxy | +1 módulo para manter |
| Chave privada cifrada | segurança | overhead de cifra (mínimo) |
| HTTP-01 default | simples | precisa porta 80 / apontamento |

## 10. Decisões

- **D-001:** Let's Encrypt = provider padrão.
- **D-002:** chaves privadas cifradas em repouso.
- **D-003:** renew a D-30 com jitter; retries backoff; alerta após 3 falhas.
- **D-004:** wildcard só via DNS-01.
- **D-005:** proxy consome refs; nunca emite.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Rate limit LE | média | jitter, SAN, DNS-01 |
| Falha silenciosa de renovação | média | alertas + renew agressivo |
| DNS provider down (DNS-01) | baixa | retry + fallback HTTP-01 quando possível |

## 12. Alternativas descartadas

- **Delegar certs ao Traefik/autocert**: descartado (sem auditoria; acopla a provider).
- **Cert manager externo (cert-manager)**: descartado (dependência externa; modelo k8s).
- **Renovação central sem jitter**: descartado (thundering herd em LE).
