# 09 — Certificate Engine

> **Status:** Núcleo de identidade de rede.
> **RFC relacionada:** [`RFC-0003`](rfc/RFC-0003-certificate-engine.md)
> **Princípios atendidos:** P1, P2, P5, P9.

---

## 1. Propósito

O **Certificate Engine** é o componente **único e soberano** responsável por certificados TLS
na plataforma. Ele controla **emissão, renovação, revogação, histórico, auditoria e alertas**.
Let's Encrypt (e outros ACME providers) são **apenas providers**.

**Regra inegociável:** o proxy nunca controla certificados. O proxy apenas *consome* os
certificados emitidos pelo Certificate Engine (referências). Isso garante que a troca de proxy
não quebre a cadeia de certificados e que a política de renovação seja centralizada e auditável.

## 2. Por que um Certificate Engine próprio (não delegar ao proxy)

| Razão | Explicação |
|-------|-----------|
| Soberania da renovação | Traefik/autocert renovam com sua própria lógica e não expõem histórico/auditoria |
| Consistência multi-proxy | Se trocarmos o provider de proxy, certs continuam os mesmos |
| Wildcard + DNS-01 | Requer integração com providers de DNS (Cloudflare, Route53...) — não é papel de proxy |
| Auditoria e compliance | Registro de quem emitiu, quando, qual domínio, qual provider |
| Alertas proativos | Falha de renovação = evento de alerta (antes do vencimento) |
| Controle de custo | Um único processo de renovação com jitter (não N proxies renovando) |

## 3. Arquitetura

```
Certificate Engine (Core)
   │
   ├── Policy Manager   (regras: providers, contas, renovação, retenção)
   ├── Account Manager  (contas ACME)
   ├── Challenge Runner (HTTP-01 via proxy; DNS-01 via plugin DNS)
   ├── Store            (certs + chaves criptografadas; histórico)
   └── Notifier         (eventos de emissão/renovação/falha)
   │
   └── Provider (porta ACME)
         ├── Let's Encrypt (default)
         ├── ZeroSSL (futuro)
         └── Private CA (futuro, enterprise)
```

### 3.1 Porta do Provider ACME

```go
type AcmeProvider interface {
    Register(ctx, email) (Account, error)
    OrderCertificate(ctx, Order) (OrderResult, error)      // inicia ordem
    CompleteChallenge(ctx, Challenge, proof) error          // http-01/dns-01
    GetCertificate(ctx, Order) (CertBundle, error)
    Revoke(ctx, cert, reason) error
}
```

### 3.2 Porta DNS (para DNS-01)

```go
type DnsProvider interface {
    UpsertTxtRecord(ctx, name, value, ttl) error
    DeleteTxtRecord(ctx, name, value) error
}
```

Implementações: Cloudflare, Route53 (AWS), Hetzner DNS, Google Cloud DNS, e outros (via plugins).

## 4. Fluxos

### 4.1 Emissão (HTTP-01)

```
domain.added → CertEngine.ensure(domain)
→ Policy: provider de conta
→ Account: registra/seleciona conta ACME
→ Order: cria ordem (domínio)
→ Challenge http-01: gera token; informa proxy (rota temporária)
→ Proxy serve challenge → ACME valida → Order finaliza
→ CertEngine baixa cert bundle → criptografa e persiste
→ Atualiza refs no proxy (config dinâmica em memória)
→ events: cert.issued
```

### 4.2 Emissão (DNS-01 / wildcard)

```
wildcard domain.added → CertEngine.ensure
→ Order wildcard → Challenge dns-01
→ DnsProvider.UpsertTxtRecord (plugin) → aguarda propagação (poll curto com timeout)
→ ACME valida → cert → persiste → refs no proxy
→ cleanup: DeleteTxtRecord
```

### 4.3 Renovação

- Timer determinístico: renova a partir de **D-30** (expiry) com **jitter aleatório** por domínio
  para não sobrecarregar ACME.
- Se falhar: retries com backoff; após N falhas → evento `cert.renewal_failed` + alerta.
- NUNCA deixar expirar: renew agressivo perto do vencimento.

### 4.4 Revogação

- Manual (UI/API), ou automática (se chave comprometida).
- Notifica ACME (revoke) e remove refs do proxy.

## 5. Storage de certificados

- Chaves privadas: **criptografadas** (AES-256-GCM; chave derivada de master key) em
  `/var/lib/aether/keys/certs/<domain>/privkey.pem.enc`.
- Cert público: texto (legível).
- Histórico: tabela `cert_events` (issued/renewed/revoked/failed) — auditável.
- Permissões: apenas `aether-core` escreve; proxy lê apenas via refs (caminhos com 0600),
  ou o core injeta certs no proxy em memória (default: injeta refs/path com ACL).

## 6. Renovação de múltiplos domínios / SAN

- Um cert pode cobrir múltiplos domínios (SAN) quando o usuário agrupar domínios por app.
- Wildcard cobre subdomínios; renew do wildcard via DNS-01.

## 7. Política e retenção

| Parâmetro | Default | Nota |
|-----------|---------|------|
| Renew window | D-30 | configurável |
| Retry backoff | 1h, 2h, 4h... máx 24h | |
| Max failures antes de alerta | 3 | |
| Retenção de histórico | 1 ano | audit |
| Manter certs revogados | sim (por 30 dias) | diagnóstico |

## 8. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Rate limit do Let's Encrypt | jitter, agrupamento por SAN, DNS-01 para wildcards |
| Falha de renovação silenciosa | alerta por evento; renew agressivo perto do vencimento |
| Chave privada comprometida | revogação imediata + rotação |
| Provider de DNS indisponível | retry + fallback para HTTP-01 (quando possível) |

## 9. Decisões

- **Decisão:** Certificate Engine soberano; proxy consome refs.
- **Decisão:** Let's Encrypt como provider padrão; ACME como porta universal.
- **Decisão:** chaves privadas criptografadas em repouso.
- **Decisão:** DNS-01 obrigatório para wildcard; HTTP-01 default para domínio único.

## 10. Referências

- RFC-0003 (Certificate Engine).
- Domínio Certificates: [`06-dominios-sistema.md`](06-dominios-sistema.md) §2.9.
- Segurança de chaves: [`16-seguranca.md`](16-seguranca.md).
