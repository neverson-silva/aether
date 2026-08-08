# SPIKE-ACME — Integração com servidor ACME real (Pebble)

> **Status:** ✅ Concluído — Certificate Engine validado contra o servidor ACME de teste
> da Let's Encrypt (Pebble), com validação HTTP-01 real.
> **Data:** 2026-08-03
> **Ambiente:** Lima VM Debian 13 (arm64) — Pebble 2.x compilado do source,
> pebble-challtestsrv (DNS fake: tudo → 127.0.0.1, AAAA desabilitado).

---

## 1. Topologia do teste

```
Aether (VM)                        Pebble (VM)
  cert engine ──ACME(HTTPS:14000)──►  CA de teste (minica)
  challenge server :5002 ◄──HTTP-01 real── Pebble valida
                                          │
  challtestsrv :8053 (DNS: spike.acme.local → 127.0.0.1)
```

- `AETHER_ACME_DIR=https://127.0.0.1:14000/dir`
- `AETHER_CHALLENGE_ADDR=0.0.0.0:5002` (porta HTTP-01 configurada do Pebble)
- `SSL_CERT_FILE=<minica.pem>` (confiança no CA de teste)

## 2. Resultado

```
t1: pending → t5: issued
```

- **Emitido**: `fullchain.pem` (CN=Pebble Intermediate CA) + `privkey.pem.enc` (AES-256-GCM, 0600)
- Pebble log: `authz ... set VALID by completed challenge` e `Issued certificate serial ...`
- Renovação automática (RenewLoop diário) cobriria o vencimento de 7 dias do Pebble.

## 3. Bugs reais encontrados e corrigidos (plataforma)

| # | Bug | Correção |
|---|-----|----------|
| B1 | `parseECKey` só aceitava PEM, mas a conta é armazenada em DER | aceitar DER e PEM |
| B2 | Conta local criada antes de um Register bem-sucedido → `accountDoesNotExist` no CA | `ensureAccountRegistered`: Register sempre (201 ou 200-existente seta o KID) |
| B3 | Authz já `valid` (revalidação) → `Accept` 400 | skip do Accept quando authz valid |
| B4 | `GetOrder` sobrescreve `order.URI` (Pebble não manda Location em GET) → WaitOrder("") | salvar `orderURI` antes do GetOrder |
| B5 | Pebble finaliza assíncrono (status `processing`, sem certURL) → x/crypto `CreateOrderCert` quebra | fallback: WaitOrder + FetchCert pelo CertURL |

## 4. Implicações para produção (Let's Encrypt real)

- LE real retorna `valid` síncrono no finalize → caminho primário; fallback B5 nunca dispara.
- HTTP-01 usa porta 80 (padrão) — validado o mesmo código com `:5002` no teste.
- Conta ACME cifrada em repouso; renovação D-30 com jitter (RFC-0003).
- **Próximo passo**: repetir este fluxo contra LE staging/real usando o domínio + VPS do usuário.

## 5. Rerun (VM Lima)

```bash
cd /tmp/pebble && sudo systemd-run --unit=pebble --property=WorkingDirectory=/tmp/pebble \
  /tmp/pebble/pebble -config test/config/pebble-config.json -dnsserver 127.0.0.1:8053
sudo systemd-run --unit=cts /tmp/pebble-challtestsrv -dnsserver :8053 -doh "" \
  -http01 "" -https01 "" -tlsalpn01 "" -defaultIPv4 127.0.0.1 -defaultIPv6 ""
# aether com AETHER_ACME_DIR=https://127.0.0.1:14000/dir e AETHER_CHALLENGE_ADDR=0.0.0.0:5002
```
