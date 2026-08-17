# 13 — Observabilidade

> **Status:** Logs, métricas, tracing, alertas, timeline, audit.
> **RFC relacionada:** [`RFC-0007`](rfc/RFC-0007-observability.md)
> **Princípios atendidos:** P2 (sem desperdício), P5 (eventos), P8 (dados do usuário).

---

## 1. Propósito

Prover visibilidade completa da plataforma e das aplicações — logs, métricas, tracing, alertas,
timeline, audit — com **desperdício zero de recursos**. A observabilidade deve ser *reagente*:
só coleta o que há subscriber.

## 2. Princípios de design

1. **Custo zero por padrão**: sem subscriber → sem coleta.
2. **Derivar do que já existe**: eventos do bus alimentam timeline/audit (não duplicar).
3. **Estruturas compactas**: logs estruturados, métricas agregadas em memória.
4. **Privacidade**: telemetria só com consentimento (P8).
5. **Nada cresce**: rotação, retenção, limites.

## 3. Logs

### 3.1 Fontes

| Fonte | Coleta |
|-------|--------|
| Logs do core/agent/proxy | journald / arquivos com rotação |
| Logs de aplicações | conmon → arquivos (`/var/log/aether/apps/<app>/*.log`) com rotação |
| Logs de build | diretório do build, retido N dias |

### 3.2 Formato

- JSON estruturado compacto: `{ts, level, src, msg, fields{...}}`.
- `level` de logs: `debug|info|warn|error` (default info).
- Sem parsing pesado no hot path.

### 3.3 Streaming

- Leitura via **sockets unix** (sem acumular em memória).
- UI consome via SSE; CLI via `aether logs`.
- Follow mode com backpressure.

### 3.4 Retenção e rotação

| Parâmetro | Default |
|-----------|---------|
| Rotação por tamanho | 10 MB por arquivo |
| Rotação por tempo | diária |
| Compressão | zstd (arquivos antigos) |
| Retenção | 14 dias ou 100 MB por serviço (configurável) |
| Journald | `SystemMaxUse` limitado |

### 3.5 Busca

- Índice mínimo por (app, timestamp, nível) — sem serviço de search.
- FTS5 (SQLite) opcional para query textual em logs retidos (pequena fração de custo).

## 4. Métricas

### 4.1 Coleta

- Host: `/proc`, `cgroup v2` (`memory.current`, `cpu.stat`, `io.stat`) — leitura barata.
- Containers: `podman stats` **sob demanda** (nunca polling contínuo).
- Plataforma: contadores internos (eventos, deploys, cache hits, etc.).

### 4.2 Agregação

- Janelas em memória (5 s / 60 s / 300 s) com rotação; eviction por idade.
- Sem armazenamento de séries longas por padrão (quem quiser, exporta).

### 4.3 Exposição

- Endpoint `/metrics` (formato Prometheus) **ativado apenas quando há subscriber** (scraper)
  ou sob demanda.
- Se nenhum scraper existe → sem coleta ativa → CPU ~0 (P2).

### 4.4 Métricas por app (para usuário)

- CPU%, RAM, IO, restarts, uptime — calculadas sob demanda no dashboard.

## 5. Tracing

- **Leve e amostrado**: spans para operações longas (deploy, backup, build, migração) com
  sampling configurável (default: 10% em produção, 100% em staging).
- Propagação: header `X-Aether-Trace`.
- Export: OTLP opcional (via plugin) para quem já tem OTel; **default: sem export** (zero custo).

## 6. Alertas

- **Avaliação por evento** (não polling): regra `quando <condição sobre evento/série> então <ação>`.
- Exemplos de regras v1:
  - `service.crashed` ≥ 3 em 10 min → alerta
  - `cert.renewal_failed` ≥ 3 → alerta
  - `deployment.failed` → alerta
  - Disk > 85% → alerta (métrica sob demanda)
  - Health check falhou → alerta
- Canais (via plugin/notify): e-mail, Slack, Discord, Telegram, webhook.
- Estados: `fired` → `resolved`; deduplicação por regra+recurso+janela.

## 7. Timeline

- Timeline por recurso = eventos do bus filtrados por aggregate (zero custo extra).
- UI: timeline de um app mostra todos os deploys, crashes, certs, scale events.

## 8. Audit

- Ações administrativas registradas em tabela append-only (quem, o quê, recurso, IP, quando).
- Nunca UPDATE/DELETE via API.
- Inclui: login/logout, RBAC changes, deploys manuais, restores, configurações de cert.
- UI de consulta + exportação.

## 9. Sem desperdício de recursos (resumo)

| Fonte de desperdício | Como evitamos |
|----------------------|---------------|
| Coletor contínuo | coleta sob demanda |
| Scraper interno | sem scraping próprio |
| Armazenamento de séries | agregado em memória; sem store longo |
| Logs duplicados | cada log tem um dono |
| Parse em hot path | JSON estruturado sem reparse |
| Busca pesada | FTS5 opcional e limitado |
| Tracing sempre ligado | amostragem configurável |

## 10. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Dashboard sem dados quando sem subscriber | explicitar "ativar monitoramento" para o usuário |
| Métricas sob demanda podem perder picos | coleta em janelas quando ativo; documentar limitação |
| Alertas em eventos podem perder estados | reconciliar em timer de baixa frequência (ex.: 60 s) apenas quando há regra ativa |

## 11. Decisões

- **Decisão:** logs estruturados com rotação; streaming via SSE.
- **Decisão:** métricas sob demanda; `/metrics` só com subscriber.
- **Decisão:** alertas avaliados em eventos + reconciliação de baixa frequência.
- **Decisão:** timeline/audit derivados do event bus.

## 12. Referências

- RFC-0007 (Observability).
- Event Bus: [`12-event-bus.md`](12-event-bus.md).
- Análise de consumo: [`04-analise-consumo-recursos.md`](04-analise-consumo-recursos.md).
