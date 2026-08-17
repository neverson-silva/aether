# RFC-0002 — Networking Engine

- **Status:** Draft
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P1, P2, P4, P5, P9
- **Dependências:** RFC-0003 (Certificate Engine)

---

## 1. Objetivo

Definir o **Networking Engine**: gestão de domínios, HTTPS/TLS, load balancing, middlewares,
service discovery e a **abstração do proxy** (Traefik é um provider; Caddy/NGINX/HAProxy/Envoy
podem ser providers futuros).

## 2. Escopo

**Dentro:** modelo de domínio (Domain, Route, Middleware, LoadBalancer), porta `ProxyProvider`,
config dinâmica em memória, service discovery, HTTP/3.
**Fora:** emissão/renovação de certificados (RFC-0003); firewall do host; DNS da infra.

## 3. Responsabilidades

- Modelar e aplicar rotas de domínios para apps/serviços.
- Aplicar middlewares (rate limit, forward auth, headers, rewrite, redirect, basic auth).
- Balancear entre réplicas; health do upstream.
- Descobrir serviços e registrar backends.
- Configurar o proxy provider **em memória** (sem arquivo em disco por deploy).
- Reconciliar estado desejado no boot do proxy.

## 4. Arquitetura

```
  Domain / Route / Middleware (domínio)
        │
  NetworkingEngine
        │  porta ProxyProvider
        │
  ┌─────┴─────┬──────────┬──────────┐
Traefik     Caddy       NGINX      Envoy   (providers)
  │ (v1)      (v2?)      (fase 4)  (fase 4)
  │  config dinâmica via API (em memória)
  ▼
  proxy escuta :80/:443
  service discovery: registros de backend por evento
```

## 5. Fluxos

### 5.1 Adicionar domínio (TLS via Cert Engine)

```
1. domain.add → valida (RBAC, DNS sugerido)
2. cria Route(domain, app, port, path, middlewares)
3. CertEngine.ensure(domain)  →  cert ref
4. ProxyConfig.update(route + cert ref)
5. provider.Apply(config)  (memória)
6. domain.added + proxy.reloaded
```

### 5.2 Service discovery

```
1. app start → service.registered (backend info)
2. Networking marca backend ativo
3. provider.Apply (rota inclui backend ativo)
4. app stop → service.stopped → marca inativo
```

### 5.3 Falha de proxy / reconcile

```
1. proxy down (unit restart)
2. Networking detecta (evento) e reaplica ProxyConfig no boot
3. estado desejado é a fonte (idempotente)
```

## 6. Interfaces

```go
type ProxyProvider interface {
    Name() string
    Apply(ctx context.Context, cfg ProxyConfig) error
    Status(ctx context.Context) (*ProxyStatus, error)
    Reload(ctx context.Context) error
    Certs(ctx context.Context) ([]ProxyCert, error) // refs de certs ativos
}

type ProxyConfig struct {
    Routes      []Route
    Middlewares []Middleware
    TLSOptions  TLSOptions  // min version, redirect, http3
    Defaults    ProxyDefaults
}

type Route struct {
    ID string; Host string; Path string
    Backends []Backend      // target service:port
    MiddlewareRefs []string
    TLS *TLSEndpoint       // ref ao cert
    LB LBPolicy
}
```

## 7. Eventos

Emitidos: `domain.added`, `domain.removed`, `route.updated`, `middleware.changed`,
`proxy.reloaded`, `proxy.down`, `backend.registered`, `backend.unregistered`.
Consumidos: `app.deployed`, `service.started`, `service.stopped`, `cert.issued`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| app.com → app | rota criada, TLS, roteamento |
| Wildcard *.app.com | DNS-01 + cert wildcard (RFC-0003) |
| Rate limit na rota | middleware aplicado |
| 2 réplicas | LB round-robin; health upstream |
| app stop | backend desregistrado; 503 sem derrubar plataforma |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Traefik como v1 | rico, dinâmica em memória, HTTP/3 | RAM ~30-60 MB; ecossistema próprio |
| Config em memória | zero I/O em disco por deploy | state em memória precisa reconcile |
| Abstração provider | antilock-in | custo de mapeamento de sintaxe |

## 10. Decisões

- **D-001:** Traefik = provider v1.
- **D-002:** config em memória; reconcile no boot.
- **D-003:** certs sempre geridos pelo Cert Engine (nunca pelo provider).
- **D-004:** HTTP/3 opt-in por rota.
- **D-005:** porta 80/443 via `CAP_NET_BIND_SERVICE` no unit do proxy.

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Provider em memória perde estado no crash | média | reconcile idempotente no boot |
| Rate limit default derruba app | média | conservador + visibilidade |
| Muitos domínios → muitos certs handshakes | baixa | SAN grouping; staples |

## 12. Alternativas descartadas

- **Caddy como v1**: mais leve, mas menos rico em middlewares e service discovery; reavaliar em v2.
- **NGINX como v1**: reload de arquivos a cada config; pior para dinâmica.
- **Deixar o proxy gerenciar certs**: descartado (RFC-0003 — soberania de certs).
- **Provider file em disco**: descartado (I/O por deploy; conflito com metas).
