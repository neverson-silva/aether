# RFC-0031 — Real-Time Notification Infrastructure

- **Status:** Draft — Especificação
- **Data:** 2026-08-05
- **Dependências:** RFC-0005 (Event Bus)

---

## 1. Objetivo

Construir uma infraestrutura completa de notificações em tempo real para a plataforma,
permitindo que todos os usuários de uma organização recebam eventos de deploy e operações
instantaneamente (se logados) ou ao retornar (via centro de notificações).

Esta RFC é um **pré-requisito arquitetural** para RFC-0030 (Service Management).

## 2. Componentes

```
Event Bus (outbox PostgreSQL)
  │
  ▼
Notification Engine
  │
  ├──▶ Bell Store (tabela notifications)
  │
  └──▶ SSE Hub (fan-out por organização)
         │
         ▼
       Clients (EventSource no browser)
         │
         ├──▶ Toast (sooner) — tempo real
         └──▶ Bell Icon (header) — badge + dropdown
```

### 2.1 Notification Engine

Subscreve ao Event Bus (`deployment.*`, `server.*`, `backup.*`) e transforma cada evento
em uma notificação formatada:

1. Extrai `org_id` do payload do evento
2. Formata a mensagem humana (ex: "✅ api-gateway deployed · 47s")
3. Armazena na tabela `notifications` (Bell Store)
4. Publica no SSE Hub para entrega imediata aos clientes conectados

### 2.2 Bell Store

**Tabela `notifications`:**
```sql
CREATE TABLE notifications (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  user_id TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,              -- deployment.ready | deployment.failed | server.registered | ...
  message TEXT NOT NULL,           -- mensagem formatada (humana)
  payload JSONB NOT NULL DEFAULT '{}',
  read INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_notif_org_read ON notifications(org_id, read, created_at DESC);
```

**API:**
- `GET /api/v1/notifications` — últimas 50 (query: `?before=<cursor>` para paginação)
- `GET /api/v1/notifications/unread-count` — `{"count": 3}`
- `POST /api/v1/notifications/{id}/read` — marca como lida
- `POST /api/v1/notifications/read-all` — marca todas como lidas da org

### 2.3 SSE Hub

- Endpoint: `GET /api/v1/events/stream` (autenticado via JWT Bearer)
- Identifica a organização pelo token JWT (`claims.OrgID`)
- Mantém um mapa `map[orgID][]chan Event` em memória
- Cada cliente conectado recebe um channel
- Eventos publicados pelo Notification Engine são enviados para todos os canais da org
- Heartbeat a cada 15s (comentário SSE `: heartbeat`) para manter a conexão
- Formato:
  ```
  event: notification
  data: {"type":"deployment.ready","message":"✅ api-gateway deployed · 47s","service_id":"...","project_id":"...","ts":"2026-08-05T12:00:00Z"}

  ```

### 2.4 Client (Frontend)

**SSE Connection:**
```typescript
// Em Shell ou App root, ao autenticar:
const es = new EventSource(`/api/v1/events/stream`, {
  // Token enviado via cookie httpOnly ou header customizado via polyfill
});
es.addEventListener('notification', (e) => {
  const data = JSON.parse(e.data);
  addToast(data);                    // sooner no canto inferior direito
  incrementBellBadge();              // atualiza o badge do sino
});
es.onerror = () => reconnect();     // backoff exponencial
```

**Reconexão:**
- Backoff: 1s → 2s → 4s → 8s → 16s → 30s (max)
- Após 5 falhas consecutivas, ativa polling fallback (`GET /unread-count` a cada 30s)
- Ao reconectar com sucesso, volta para SSE

**Toast (Sooner):**
| Tipo do evento | Aparência | Duração |
|---|---|---|
| `deployment.ready` | Verde, ícone ✅, service name + duration | 8s auto-dismiss |
| `deployment.failed` | Vermelho, ícone ❌, clicável → /services/:id/deployments | Persistente |
| `deployment.queued / .building` | Amarelo/neutro, ícone ⏳ | 5s auto-dismiss |
| `server.registered` | Azul | 5s auto-dismiss |
| `backup.finished` | Verde | 5s auto-dismiss |

## 3. UX do Bell Icon

**Header (shell):**
```
┌─────────────────────────────────────────────┐
│  Aether > Default > api-gateway     🔔 3  👤│
└─────────────────────────────────────────────┘
```

**Dropdown:**
```
┌──────────────────────────────────────┐
│  Notifications              Mark all │
│  ─────────────────────────────────── │
│                                      │
│  ✅ api-gateway deployed             │
│     47s · port 3000 · just now       │  ← bold (não lida)
│                                      │
│  ❌ worker-emails failed             │
│     OOM · 5m ago                     │  ← bold (não lida)
│                                      │
│  ✅ postgres-main backup finished    │
│     2.4GB · 1h ago                   │  ← normal (lida)
│                                      │
│  🆕 server worker-1 registered       │
│     3h ago                           │
└──────────────────────────────────────┘
```

- Badge: círculo com número (some quando 0)
- Scroll: max 400px, scroll interno
- Clicar em notificação → marca como lida + navega (abre o service detail no deploy)
- "Mark all": marca todas como lidas, reseta o badge

## 4. Eventos Emitidos

### 4.1 Deploy Events (emitidos pelo Core.Deploy)

| Evento | Payload | Exemplo de mensagem |
|---|---|---|
| `deployment.queued` | serviceID, name, project, env, triggeredBy, commit/image | **api-gateway** deploy queued · triggered by neversonbs13@gmail.com |
| `deployment.building` | serviceID, name, buildMethod, commit | Building **api-gateway** · nixpacks · commit a84f9b2 |
| `deployment.starting` | serviceID, name, containerID[0:12], server | Starting **api-gateway** · container d57663… |
| `deployment.healthcheck` | serviceID, name, attempt, maxAttempts | Health check **api-gateway** · GET / · 3/30 |
| `deployment.ready` | serviceID, name, durationMs, port, url | ✅ **api-gateway** deployed · 47s · port 3000 |
| `deployment.failed` | serviceID, name, error, durationMs | ❌ **api-gateway** failed · port 3000 already in use |

### 4.2 Server Events

| Evento | Payload |
|---|---|
| `server.registered` | serverID, name, host |
| `server.marked_unhealthy` | serverID, name |
| `server.recovered` | serverID, name |

### 4.3 Backup Events

| Evento | Payload |
|---|---|
| `backup.started` | backupID, dbName, engine |
| `backup.finished` | backupID, dbName, size, durationMs |
| `backup.failed` | backupID, dbName, error |

## 5. Segurança

- SSE stream requer autenticação (JWT). Token pode ser passado via cookie `aether_token` (HttpOnly) — o EventSource nativo não suporta headers custom, então o endpoint lê o cookie.
- Eventos contêm `org_id` — o SSE Hub filtra por organização (nunca vaza eventos entre orgs).
- O payload NÃO contém valores de secrets ou connection strings.
- Bell Store: usuários só veem notificações da sua organização.

## 6. Migração

```sql
-- migration 11
CREATE TABLE IF NOT EXISTS notifications (
  id TEXT PRIMARY KEY,
  org_id TEXT NOT NULL,
  type TEXT NOT NULL,
  message TEXT NOT NULL,
  payload TEXT NOT NULL DEFAULT '{}',
  read INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notif_org_read ON notifications(org_id, read, created_at DESC);
```

## 7. Checklist de Implementação

- [ ] Tabela `notifications` + índices (migration 11)
- [ ] `POST /api/v1/notifications/{id}/read` + `read-all` + `unread-count`
- [ ] `NotificationEngine` — subscriber do Event Bus, formatador de mensagens
- [ ] `SSE Hub` — fan-out por organização, heartbeat 15s
- [ ] `GET /api/v1/events/stream` — endpoint SSE que lê cookie de auth
- [ ] Bell Icon + badge + dropdown no shell header
- [ ] Toast (sooner) system upgrade no ToastProvider — suporte a níveis (success/error/info) com auto-dismiss
- [ ] Client SSE connection com reconexão backoff + polling fallback
- [ ] Enriquecer `deployment.*` eventos com triggeredBy, serviceName, etc. (já parcialmente implementado via FireDeployStarted / FireWebhookEvent)
- [ ] Wire NotificationEngine aos eventos de deploy (Core → Bus → Engine)
- [ ] Teste E2E: duas sessões em browsers diferentes (mesma org) → deploy na sessão A → toast aparece na sessão B em <2s
- [ ] Teste offline: deploy com usuário deslogado → ao logar, bell mostra badge com o evento
