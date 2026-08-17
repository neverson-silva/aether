# RFC-0015 — Multi Server e Clusters

- **Status:** Implementado (v1)
- **Autor:** Aether Architecture Team
- **Data:** 2026-08-02
- **Princípios:** P5, P9, P10
- **Dependências:** RFC-0005 (Event Bus), RFC-0006 (Runtime Driver API)

---

## 1. Objetivo

Definir o modelo multi-servidor: 1 servidor central (core) + N servidores workers (agents),
formação de clusters, roteamento de workloads, tolerância a falhas. A arquitetura cresce sem
reescrita (P10).

## 2. Escopo

**Dentro:** modelo de Server/Agent/Cluster, registro e mTLS, distribuição de eventos,
scheduler de workloads, afinidade, heartbeat, upgrade de agentes, failover do scheduler.
**Fora:** HA de banco (fase 5); load balancer externo.

## 3. Responsabilidades

- Registrar servidores/agentes (com mTLS).
- Distribuir eventos entre nós (core↔agent via gRPC streaming).
- Agendar workloads para servidores elegíveis.
- Aplicar afinidade/labels (ex.: "rodar GPU nesta máquina").
- Detectar falhas (heartbeat + timeouts) e reagir.
- Atualizar agentes (propagação de versão).
- Garantir 1 scheduler ativo por cluster (leader election).

## 4. Arquitetura

```
        Central (core + banco + proxy opcional)
        │  gRPC mTLS (streaming de eventos)
   ┌────┴────┬──────────┬────────────┐
 Worker A  Worker B  Worker C    ...
 (agent)    (agent)   (agent)

Scheduler: agenda Deployment → servidor elegível → eventos distribuídos
Clusters: agrupamento lógico com labels e afinidade
```

## 5. Fluxos

### 5.1 Registrar servidor

```
1. `aether server add` → gera token de inscrição (curto)
2. agent (instalado no worker) conecta ao core com token + mTLS handshake
3. core emite certs; registra server; server.heartbeat inicial
4. server.registered
```

### 5.2 Deploy em cluster

```
1. deployment.created
2. Scheduler: filtra servidores (labels/recursos/health)
3. escolhe target (least-loaded, afinidade)
4. evento deployment.assigned → agent target
5. agent executa (build/schedule/health) → eventos de status → core projeta
```

### 5.3 Falha de nó

```
1. heartbeat ausente por timeout
2. server.marked_unhealthy
3. workloads com replicas → reagendados para outro nó
4. alerta + audit
```

### 5.4 Leader election (HA de core)

```
1. core nodes concorrem por lock distribuído (no event log / etcd se presente)
2. vencedor = scheduler ativo; perdedores em standby
3. falha → eleição nova
```

## 6. Interfaces

```go
type AgentClient interface {
    Send(ctx, events []Event) error
    Recv(ctx) (<-chan Event, error)
    Ping(ctx) (*Heartbeat, error)
    ExecRemote(ctx, spec) (RemoteResult, error)
}

type Scheduler interface {
    Place(ctx, workload Workload) (ServerID, error)
    Rebalance(ctx, policy) error
}
```

## 7. Eventos

Emitidos: `server.registered`, `server.heartbeat`, `server.marked_unhealthy`,
`server.recovered`, `agent.upgraded`, `cluster.formed`, `deployment.assigned`,
`leader.elected`.
Consumidos: `deployment.created`, `deployment.cancelled`, `server.removed`.

## 8. Casos de uso

| Caso | Resultado |
|------|-----------|
| 2 workers | deploy vai para o menos carregado |
| Nó cai | workloads reagendados |
| Adicionar nó | cluster cresce sem reconfig |
| Manutenção | drain + reagendamento |

## 9. Trade-offs

| Decisão | Prós | Contras |
|---------|------|---------|
| Eventos distribuídos (não REST polling) | consistência, reativo | infra de streaming |
| mTLS | segurança | setup de certs |
| Scheduler simples (least-loaded) | previsível | não otimiza custo/energia |
| Leader election | HA | +1 camada |

## 10. Decisões

- **D-001:** 1 core + N agents; comunicação por HTTP/2 mTLS (JSON sobre HTTP mTLS; gRPC documentado como evolução — mesmo protocolo de eventos, sem polling) — implementado.
- **D-001b:** registro por token single-use (24h) + CA interna (crypto/x509) emitindo certs; registro aceita cert do servidor (InsecureSkipVerify) — o token autentica o enrollment (padrão docker swarm).
- **D-002:** scheduler least-loaded com labels/afinidade.
- **D-003:** heartbeat com timeout; reagendamento para replicas.
- **D-004:** leader election no HA de core (fase 5).
- **D-005:** agentes atualizam por propagação de versão.

## 10.5 Notas de implementação (v1)

- Migração 5: tabelas `servers`, `server_tokens`, `server_commands`; `apps.server_id` e `deployments.server_id`.
- Endpoints do agente (mTLS): `POST /agent/v1/register|heartbeat|events|exec`, `GET /agent/v1/commands` (long-poll); heartbeats a cada 5s, watchdog marca unhealthy após 30s.
- Scheduler: least-loaded entre servidores `healthy`; `apps.server_id` força o alvo; sem agentes saudáveis → deploy local.
- Deploy remoto v1: apps de **imagem** (pull → network/volume → run → health); git builds permanecem locais (limitação documentada).
- Logs do deploy remoto transmitidos ao core e gravados no mesmo arquivo de log do app.
- CLI: `aether server token|list|rm`, `aether agent --core --token --name [--labels]`.
- Limitação: registros intermitentes para agentes em outra máquina exigem certificado com SAN do hostname/IP do core (cert inclui localhost + 127.0.0.1 + hostname da máquina).

## 11. Riscos

| Risco | Prob. | Mitigação |
|-------|-------|-----------|
| Rede particionada | baixa | timeouts + reconciliação |
| Split-brain scheduler | baixa | leader election + lock |
| Agent desatualizado | baixa | upgrade propagado |
| Latência de evento | baixa | streaming persistente |

## 12. Alternativas descartadas

- **Kubernetes como base do cluster**: descartado em v1 (overhead; é driver futuro).
- **REST polling agent→core**: descartado (P5).
- **Sem scheduler (tudo local)**: descartado (impossível multi-server).
