# 00 — Manifesto, Filosofia e Princípios

> **Status:** Fundacional — princípios inegociáveis.
> **Audiência:** Toda e qualquer pessoa que tocar na base de código.

---

## 1. Missão

Construir uma plataforma **Self-Hosted PaaS Open Source** que permita a qualquer equipe
executar, escalar e operar aplicações em containers com a mesma experiência de produto de
PaaS gerenciadas (Vercel, Railway, Fly.io, Heroku), rodando 100% na infraestrutura do próprio
usuário, com paridade funcional com Coolify e Dokploy e **consumo de recursos drasticamente
inferior**.

O produto não é um "Docker Manager". Não é um "painel administrativo". Aether é um
**Sistema Operacional para Aplicações** — `OS for Apps`.

## 2. O que significa "Sistema Operacional para Aplicações"

Um sistema operacional provê ao programa:

1. **Abstração de hardware** — o programa nunca conhece o hardware físico.
2. **Gerenciamento de processos** — o SO decide como e onde executar.
3. **Gerenciamento de recursos** — o SO arbitra CPU, memória, I/O.
4. **Sistema de arquivos** — o SO provê namespace de arquivos.
5. **Redes** — o SO provê comunicação entre processos.
6. **Segurança** — o SO isola processos e controla acesso.
7. **Serviços comuns** — o SO provê serviços reutilizáveis.

Traduzindo para Aether:

| Conceito de SO | Correspondente Aether |
|----------------|----------------------|
| Hardware | Servidores físicos/VMs/nós |
| Processos | Aplicações e seus deployments |
| Recursos | CPU/RAM/SSD/redes por ambiente |
| Sistema de arquivos | Volumes, armazenamento persistente |
| Redes | Domains, HTTPS, proxy, service discovery |
| Segurança | RBAC, secrets, isolamento, TLS |
| Serviços comuns | Logs, metrics, backups, observabilidade |

O usuário opera **conceitos de alto nível** (Applications, Projects, Deployments, Domains,
Servers, Organizations, Templates, Environments). **Containers são apenas uma implementação**
desse modelo — um detalhe do runtime, intercambiável e oculto.

### Consequência arquitetural inegociável

> **O runtime nunca deve ser conhecido pelas camadas superiores.**

Nenhuma parte do domínio de aplicação pode importar, referenciar ou depender de Podman, Docker,
containerd ou Kubernetes. Toda interação com o mundo de containers acontece exclusivamente
através da interface do **Execution Engine**.

## 3. A proposição central: eficiência de recursos

Toda infraestrutura consumida pela plataforma **reduz a capacidade disponível para os containers
do usuário**. Se a plataforma consome 1 GB de RAM e 5 GB de SSD em idle, o usuário perde exatamente
isso de capacidade produtiva.

**Aether nunca deve competir por recursos com as aplicações do usuário.** Essa frase é o critério
de design para praticamente todas as decisões deste documento.

### Metáfora de aceite

Em um servidor de 4 vCPUs / 8 GB RAM / 100 GB SSD:

- Aether instalada e ociosa deve ser **quase imperceptível**: `top` não deve mostrar processos
  Aether entre os 10 primeiros consumidores de CPU.
- O custo fixo da plataforma deve caber na "casca" do sistema: containers leves de sistema,
  um processo supervisor, um processo de API, agents dormidos.

## 4. Princípios arquiteturais inegociáveis

### P1 — OCI First, nunca Docker First

- Toda arquitetura é baseada nos padrões da **Open Container Initiative**: image spec,
  runtime spec (runc/crun), distribution spec.
- **Nunca acoplar a Podman.** **Nunca acoplar a Docker.**
- Toda comunicação com o mundo de containers passa pelo **Execution Engine** (abstração) e,
  abaixo dele, por **Runtime Drivers** (implementações).
- A primeira implementação usa Podman/Buildah/Skopeo/Quadlet/crun, mas o código de domínio
  jamais referencia essas ferramentas.

### P2 — Zero desperdício estrutural

- Nenhum serviço roda se não estiver em uso.
- Nenhuma rotina roda se não houver trabalho.
- Nenhum cache cresce sem limite.
- Nenhum log é persistido sem política de retenção.
- Toda operação periódica é **acionada por evento ou agendada com política rigorosa**, nunca
  por polling cego.

### P3 — Mínimos processos residentes

- O plano de controle deve rodar com o menor número possível de processos.
- Trabalho assíncrono vive em **workers efêmeros** (sob demanda), não em um conjunto de
  workers residentes ociosos.

### P4 — Poucos containers, e minimalistas

- A plataforma opera containers próprios apenas onde o isolamento é necessário (execução de
  workloads de usuário, proxies, builders). Onde um processo host resolver, usar processo host.
- Nenhum container "bônus" de suporte (monitor, log shipper, agent de métricas) é implantado
  sem necessidade explícita.

### P5 — Eventos como fonte primária de verdade

- Proibir polling sempre que possível.
- O estado do sistema é derivado de uma sequência de eventos.
- Componentes se comunicam por um **Event Bus** assíncrono e persistente.
- Tudo o que pode ser reativo, é reativo.

### P6 — Degradação e simplicidade operacional

- Instalação em um comando, em servidor limpo.
- Atualização atômica e reversível.
- Nenhuma operação manual de manutenção em uso normal.
- Rollback de qualquer deployment em um clique.

### P7 — Menos, mas composto

- Prefira compor bibliotecas pequenas e maduras a frameworks gigantes.
- Nada de "só mais uma imagem Postgres para o painel", "só mais um Redis para cache",
  "só mais um container de monitoramento".

### P8 — Os dados são do usuário

- Nenhum telemetria obrigatória. Nenhum dado enviado para fora sem consentimento explícito.
- Backup e restauração são direitos, não features.

### P9 — Segurança por padrão

- Rootless por padrão. Least privilege em todos os processos.
- Secrets criptografados em repouso e em trânsito.
- Auditoria completa de ações administrativas.

### P10 — Crescimento sem reescrita

- A arquitetura em camadas e módulos deve permitir crescer de "1 servidor, 1 banco SQLite"
  até "N servidores, Postgres HA, clusters" sem mudanças estruturais — apenas habilitando camadas.

## 5. O que NÃO é Aether

| Não é | Por quê |
|-------|---------|
| Um Docker Manager | Gerenciar containers é detalhe de implementação; o domínio opera aplicações |
| Um painel administrativo | Há API/CLI de primeira classe; a UI é um cliente |
| Uma PaaS gerenciada | É self-hosted; o usuário controla tudo |
| Um Kubernetes distro | K8s pode ser um *driver* futuro, nunca o modelo mental da plataforma |
| Um CI/CD | Build e deploy são primitivas; CI completo fica fora do escopo v1 |

## 6. Promessas ao usuário

1. **Você nunca vai competir com a plataforma por RAM, CPU ou SSD.**
2. **Você pode migrar do Coolify ou do Dokploy em horas**, não semanas.
3. **A plataforma pode ser instalada em hardware que os concorrentes consideram insuficiente**
   (ex.: VPS de 512 MB–1 GB).
4. **Suas aplicações continuarão rodando mesmo se a plataforma for atualizada, reiniciada ou
   o painel estiver indisponível** — porque os workloads são geridos pelo runtime OCI de forma
   declarativa, não por um daemon central da plataforma.
5. **Nada é obrigatório**: cada provider, integração e módulo é carregado sob demanda.

## 7. Como este manifesto é aplicado na engenharia

- Cada RFC deve citar explicitamente qual(is) princípio(s) atende.
- Toda decisão que aumentar consumo de recursos em idle precisa de justificativa escrita e
  aprovação, registrada na RFC.
- As metas de [`03-metas-engenharia.md`](03-metas-engenharia.md) são checadas em CI (benchmarks
  de recursos) a cada release candidato.
