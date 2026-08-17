# 08 — Networking Engine

> **Status:** Núcleo de rede.
> **RFC relacionada:** [`RFC-0002`](rfc/RFC-0002-networking-engine.md)
> **Princípios atendidos:** P1, P2, P4, P5, P9.

---

## 1. Propósito

O **Networking Engine** administra toda a exposição de aplicações à rede: domínios, HTTPS,
TLS/termination, balanceamento, middlewares, service discovery. Ele abstrai o proxy — **Traefik
é apenas um Provider** (implementação padrão). No futuro: Caddy, NGINX, HAProxy, Envoy sob a
mesma interface.

## 2. Por que abstrair o proxy

1. **Antilock-in**: Traefik/Caddy/NGINX mudam de ecossistema a cada ciclo; a plataforma não pode
   depender de um.
2. **Semântica única**: o domínio fala "domínio", "rota", "middleware"; o provider traduz para
   a sintaxe do produto.
3. **Migração**: trocar provider = trocar driver, não reescrever a plataforma.
4. **Custo/recursos**: podemos escolher o provider mais leve para o caso (ex.: Caddy é mais
   simples; Traefik mais rico; NGINX familiar).

## 3. Interface do Provider de Proxy

```go
type ProxyProvider interface {
    Name() string
    Apply(ctx, ProxyConfig) error       // config dinâmica em memória
    GetStatus(ctx) (ProxyStatus, error) // versão, backends, certs carregados
    Reload(ctx) error
    Certificates() ([]ProxyCert, error) // certs ativos (somente refs/estado)
}
```

`ProxyConfig` é uma lista de rotas, middlewares, TLS refs e load balancers — **neutra**.

## 4. Traefik como Provider padrão — por quê

| Critério | Traefik | Caddy | NGINX | Envoy |
|----------|---------|-------|-------|-------|
| Config dinâmica em memória (API) | excelente | boa (JSON API) | média (reload de config) | excelente (xDS) |
| Service discovery nativo | excelente | boa | requer lua/código | excelente |
| Middlewares prontos (auth, rate limit, headers) | muito bons | razoáveis | bons (via lua/nginx) | bons |
| HTTP/3 | suporte | suporte | parcial | suporte |
| Consumo de RAM | ~30-60 MB | ~15-30 MB | ~10-25 MB | ~50-100 MB |
| Certificate management integrado | sim (mas usamos o nosso) | sim (Autocert) | via certbot | via mTLS/CA |
| Modelo declarativo | ótimo | ótimo | declarativo de arquivos | declarativo (xDS) |

**Decisão:** Traefik em v1 — melhor equilíbrio entre dinâmica em memória, middlewares,
HTTP/3 e maturidade, com custo aceitável. Caddy é candidato para v2 se a pressão de RAM
exigir. NGINX/Envoy para casos específicos (fase 4+).

## 5. Arquitetura de aplicação de config (sem escrita em disco)

- O **Certificates** emite/mantém certs.
- O **Networking** constrói `ProxyConfig` em memória.
- O **agent/proxy** aplica via **API HTTP do Traefik** (dynamic config) — **sem arquivo em
  disco a cada deploy** (D4 mitigado).
- Reconciliação: se o proxy cair, no start ele re-registra (estado desejado reaplicado).

## 6. Fluxos principais

### 6.1 Adicionar domínio a uma app

```
[UI/API] → domain.add → Applications.domain.attached
→ Networking: cria Route(domain, target)
→ Certificates: solicita/renova cert (HTTP-01 ou DNS-01)
→ ProxyConfig atualizada → apply (API Traefik)
→ events: domain.added, cert.issued, proxy.reloaded
```

### 6.2 Terminação TLS

- Proxy termina TLS (443), entrega HTTP ao app na rede interna.
- Cert refs vêm do Certificate Engine (nunca o proxy controla emissão).

### 6.3 Service Discovery

- Cada app/serviço registra um backend no Networking ao iniciar (`service.registered`).
- Rota dinâmica → proxy redireciona por hostname/path sem reiniciar.

## 7. Domínios, HTTPS e desafios ACME

| Desafio | Uso | Provider |
|---------|-----|----------|
| HTTP-01 | domínios com apontamento pronto | Let's Encrypt (default) |
| DNS-01 | wildcard, domínios sem porta 80 | Let's Encrypt, Cloudflare DNS, Route53, etc. (via plugin dns) |

- **Wildcard**: só via DNS-01.
- **Renovação**: 30 dias antes do vencimento, com jitter para evitar thundering herd; falhas
  geram alerta (evento `cert.renewal_failed`).

## 8. Middlewares (v1)

| Middleware | Descrição |
|-----------|-----------|
| Rate Limit | limite por IP/client; configurável por rota |
| Forward Auth | delegar autenticação a serviço externo (ex.: authelia) |
| Headers | injetar/remover cabeçalhos; security headers |
| Rewrite | reescrita de paths |
| Redirect | redirecionamentos (http→https, apex→www) |
| Auth basic | proteção simples por rota |

## 9. Load Balancing

- Round-robin entre réplicas de um serviço.
- Affinity por IP (sticky) opcional.
- Health check upstream usado pelo proxy (marcar backend indisponível).

## 10. HTTP/3

- Habilitado por app (H3 + QUIC no 443); config via middleware/TLS do provider.
- Fallback transparente para HTTP/2/1.1.

## 11. Rede interna e comunicação entre apps

- Redes podman por app; service discovery por nome de rede.
- Apps em rede privada não expõem portas públicas; apenas o proxy fala com elas
  (ou porta publicada quando o usuário pedir).

## 12. Estrutura de dados (domínio Networking)

```
Domain { id, name, tls {provider, challenge}, wildcard bool }
Route { domainRef, appRef, port, path, middlewareRefs[], lb {...} }
Middleware { type, config }
ProxyConfig { routes[], middlewares[], certs[], tlsMinVersion }
```

## 13. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Provider só em memória pode perder config em restart | reconciliation: aplicar estado desejado no boot |
| Rate limit mal configurado derruba app | limites default conservadores; visibilidade |
| TLS handshake para muitos domínios | certs agrupados por SAN; OCSP staple opcional |

## 14. Decisões

- **Decisão:** abstração `ProxyProvider`; Traefik é o driver padrão.
- **Decisão:** config dinâmica em memória (sem escrita de arquivos).
- **Decisão:** terminação TLS no proxy; certs geridos pelo Certificate Engine.
- **Decisão:** HTTP/3 por opt-in.
- **Decisão:** portas 80/443 no proxy via CAP_NET_BIND_SERVICE no unit systemd do proxy.

## 15. Referências

- RFC-0002 (Networking Engine).
- Certificate Engine: [`09-certificate-engine.md`](09-certificate-engine.md).
- Domínio Networking: [`06-dominios-sistema.md`](06-dominios-sistema.md) §2.8.
