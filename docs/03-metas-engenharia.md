# 03 — Metas de Engenharia (SLAs Internos de Recursos)

> **Status:** Critério de aceite para toda a arquitetura.
> **Regra de ouro:** Qualquer PR que aumente consumo em idle precisa de justificativa escrita
> na RFC e aprovação. As metas abaixo são verificadas em CI por benchmark automatizado.

---

## 1. Como as metas são medidas

Todas as medições usam um **cenário de referência único**:

```
Hardware de referência: VPS Linux x86_64, 4 vCPU, 8 GB RAM, 100 GB SSD NVMe
Distribuição: Debian 12 (glibc, kernel 6.x, cgroup v2)
Estado: instalação limpa de Aether, nenhuma aplicação do usuário
```

Medições:
- **RAM**: `systemd-cgtop` / cgroup do processo da plataforma + RSS somado dos processos.
- **SSD**: du no diretório de instalação + diretórios de dados.
- **CPU**: `top` em janela de 60 s em idle.
- **Processos**: contagem via cgroup.

## 2. Metas de idle (sem usuários, sem aplicações)

| Métrica | Meta v1 | Meta Fase 5 (Enterprise) |
|---------|---------|--------------------------|
| RAM total (plano de controle) | **< 120 MB** | < 256 MB (+ audit/HA) |
| SSD total (instalação limpa) | **< 300 MB** | < 500 MB |
| CPU idle (média 60 s) | **≈ 0%** (evento-driver) | < 1% |
| Processos residentes | **≤ 6** | ≤ 12 (inclui agentes) |
| Containers de suporte | **0** | 0 (continua) |
| Imagens de suporte | **0** | 0 |
| Threads do processo principal | ≤ 12 | ≤ 32 |

### 2.1 Justificativa dos números

- **RAM < 120 MB**: processo único compilado (Go/Rust) com heap pequeno + SQLite (buffers
  internos ~2–8 MB) + cache LRU em memória limitado + nenhum daemon. 120 MB é realista para um
  binário estático + runtime; bem abaixo dos 400 MB–1.5 GB dos concorrentes.
- **SSD < 300 MB**: binário (~30–80 MB), SQLite (< 10 MB), diretórios de runtime (units, sockets,
  state), plugins essenciais, e **nenhuma imagem** (proxy é binário, não imagem). 300 MB é folgado.
- **Processos ≤ 6**: `core` (1), `agent` (1), `proxy` (1), `sqlite-wal-ckpt` (não é processo, é
  thread — os 6 são: core, agent, proxy, + possíveis workers efêmeros). Na prática: core+agent+proxy = 3.
- **CPU ≈ 0**: nenhum polling; work acionado por evento; timers com backoff determinístico.

## 3. Metas de operação

| Operação | Meta v1 | Notas |
|----------|---------|-------|
| Instalação limpa (script → pronto) | **< 2 min** | sem downloads de imagem; binário único |
| Primeira inicialização | **< 5 s** | banco pequeno, migrações rápidas |
| `aether update` (sem downtime) | **< 60 s** | troca de binário + migração transacional |
| Rollback de versão da plataforma | **< 30 s** | binário anterior + migração reversa |
| Deploy de app com imagem pronta | **< 10 s** | pull + unit start + health check |
| Deploy de app com build local | **< 2 min** (app de referência) | Buildah rootless |
| Rollback de deployment | **< 30 s** | unit anterior + configuração anterior |
| Restart de app após crash | **< 5 s** | systemd restart policy |
| Reinício do host (recovery) | **< 90 s** até apps prontas | units systemd em boot |

## 4. Metas de capacidade

| Métrica | Meta v1 | Notas |
|---------|---------|-------|
| Aplicações suportadas por servidor | ≥ 200 em um nó de 4 vCPU/8 GB | aplicações pequenas |
| Deployments simultâneos por servidor | configurável (default 2) | controle de contenção de CPU/IO |
| Agentes multi-servidor | até 50 por servidor central | fase 3+ |
| Bancos de dados gerenciados | ≥ 30 | fase 2+ |
| Backup agendado simultâneo | não conflita com deploys | política de janelas |

## 5. Metas de qualidade de execução

| Métrica | Meta |
|---------|------|
| Uptime do plano de controle | ≥ 99.9% (um nó) |
| Zero-downtime em updates | obrigatório v1 |
| Build de app nunca bloqueia outro build | fila com limite de concorrência |
| Plataforma nunca OOM-kills app do usuário | políticas de cgroup separadas |
| Monitoramento nunca excede 1% de CPU extra | sem subscriber = sem coleta |

## 6. Metas de migração (paridade de produto)

| Migração | Meta |
|----------|------|
| Coolify → Aether (config + apps) | < 1 dia de trabalho |
| Dokploy → Aether | < 1 dia de trabalho |
| Aether → Coolify/Dokploy (saída) | formato declarativo exportável |

## 7. Como as metas entram em CI

- **Benchmark stage**: cada PR que toca em core/agent/proxy roda o cenário de referência e
  compara com baseline; regressão > 10% bloqueia merge (exceto com justificativa aprovada).
- **Idle profile**: teste que liga a plataforma, espera 60 s, e verifica RAM/SSD/CPU/processos
  contra as metas.
- **E2E mínimos**: install → deploy → update → rollback → restore, com cronômetros.

## 8. Não-metas (para não virar burocracia)

- Não medir latência de UI (métrica de produto, não de arquitetura).
- Não medir consumo em servidores de 128 vCPU (fora do alvo).
- Não medir throughput de proxy puro em carga máxima (o proxy delega a implementação do provider).
