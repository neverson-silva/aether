# 01 — Visão Geral do Produto

> **Status:** Fundacional.
> **Escopo:** Define o produto, suas funcionalidades, personas e o contorno de v1.

---

## 1. Definição do produto

Aether Platform é uma plataforma self-hosted, open source, para implantação e operação de
aplicações em containers, com o modelo mental de um **Sistema Operacional para Aplicações**.

V1 tem **paridade funcional com Coolify e Dokploy** nas funcionalidades listadas na seção 3,
permitindo migração simples entre as plataformas. Após a paridade, funcionalidades exclusivas
serão adicionadas (seção 5 do [`18-roadmap.md`](18-roadmap.md)).

## 2. Personas

| Persona | Descrição | Necessidade central |
|---------|-----------|---------------------|
| **Indie Hacker / Solo** | Uma pessoa, uma VPS pequena | Baixo custo fixo, deploy rápido, pouco para administrar |
| **Agência** | Dezenas de clientes | Multi-tenant (Organizations), RBAC, templates, custo previsível |
| **Time de Produto** | Equipes internas | Preview deployments, Git integração, rollback, observabilidade |
| **Startup bootstrap** | Pouco budget, muito deploy | Eficiência de recursos; rodar plataforma + apps em 1 nó |
| **Enterprise** | Governança, compliance | SSO/OIDC, audit logs, multi-servidor, alta disponibilidade |

## 3. Funcionalidades de v1 (paridade com Coolify/Dokploy)

A lista abaixo é o contorno funcional de v1. Cada item detalhado possui RFC e domínio próprio.

### 3.1 Aplicações e Deployments

- **Applications**: aplicações de usuário (web, workers, cron jobs) com imagem, git source ou
  docker compose.
- **Deployments**: pipeline completo build → push → schedule → health check → promote.
- **Rollback**: voltar para deployment anterior em um clique.
- **Docker Compose**: orquestração de stacks compose nativas (mapeadas para o runtime OCI).
- **Templates**: modelos parametrizáveis de aplicações.
- **Marketplace / One Click Apps**: catálogo de aplicações prontas.
- **Databases**: provisionamento de bancos gerenciados (PostgreSQL, MySQL/MariaDB, Redis,
  MongoDB, etc.).
- **Cron Jobs**: execução agendada de tarefas.
- **Workers**: processos de trabalho de longa duração.
- **Builds**: build de imagens OCI a partir de fonte (Dockerfile, Buildpacks CNB, custom).

### 3.2 Operação

- **Logs**: streams de logs por serviço, retenção configurável.
- **Metrics**: consumo de CPU/RAM/IO por serviço e por servidor.
- **Health Checks**: checks de prontidão/liveness.
- **Monitoring e Alerts**: alertas sobre métricas e eventos.
- **Backups e Restore**: backups agendados e restauração.
- **Volumes**: gerenciamento de storage persistente.
- **Secrets / Environment Variables**: gestão segura de configuração.
- **Terminal**: acesso via UI/CLI ao runtime.

### 3.3 Rede e Certificados

- **Domains**: vinculação de domínios a aplicações.
- **SSL / Certificados**: emissão e renovação automática (Let's Encrypt e outros).
- **Traefik**: proxy padrão com abstração para múltiplos providers futuros.
- **Networking**: redes virtuais entre aplicações, service discovery.
- **HTTP/3, Load Balancing, Middlewares, Rate Limiting, Forward Auth**: recursos de proxy.

### 3.4 Fontes e Automação

- **GitHub, GitLab, Bitbucket**: integração com repositórios.
- **Webhooks**: gatilhos de deploy.
- **Preview Deployments**: ambientes temporários por PR/branch.
- **Cron Jobs / Workers / Health Checks** (já listados em 3.1/3.2).

### 3.5 Governança

- **Organizations, Teams, RBAC, Users**: multi-tenant com controle de acesso.
- **Audit Logs**: trilha de auditoria.
- **API e CLI**: automação de primeira classe.

### 3.6 Infraestrutura

- **Multi Server**: múltiplos servidores de aplicação.
- **Clusters**: agrupamento lógico de servidores.
- **Storage, Registry, Images, Networks, SSH**: primitivas de infraestrutura.

## 4. Funcionalidades EXCLUÍDAS de v1 (decisão de escopo)

| Excluída | Justificativa | Quando |
|----------|---------------|--------|
| Pipeline CI completo (jobs arbitrários) | Escopo explode; build-deploy é suficiente em v1 | Fase 3+ (integração com runners) |
| Feature flags como produto | Custo alto, demanda de marketplace | Pós-paridade |
| Serverless/FaaS | Não é paridade com Coolify/Dokploy | Fora de escopo |
| Edge/global CDN | Fora do modelo self-hosted | Fora de escopo |
| Marketplace multi-vendor | V1: catálogo próprio + plugins | Fase 4 |

## 5. Plataformas suportadas (v1)

| Suporte | Plataformas |
|---------|-------------|
| **Primário** | Linux x86_64 e aarch64 (glibc), systemd |
| **Avançado** | Distribuições: Debian, Ubuntu, Fedora, RHEL/CentOS Stream, openSUSE |
| **Descartado em v1** | macOS como servidor; Windows como servidor; FreeBSD |

## 6. Experiência de instalação (conceito)

```bash
curl -fsSL https://get.aether.sh | sh
```

O instalador (detalhado em [`15-installer.md`](15-installer.md)):

1. Detecta a distribuição, arquitetura, init system e recursos.
2. Verifica pré-requisitos (systemd, network, UID 0).
3. Instala apenas o runtime necessário (podman + crun + buildah + skopeo) se ausente.
4. Cria usuários de serviço rootless, diretórios e unidades systemd.
5. Inicializa o banco, gera identidade, executa migrações.
6. Abre a UI em `http://<host>` e imprime o token de bootstrap.
7. Deixa o sistema pronto em < 2 minutos em hardware mediano.

## 7. Experiência de atualização (conceito)

```bash
aether update
```

- Binário único substituído de forma atômica; banco migrado de forma transacional;
  unidades systemd recarregadas; agentes atualizados por propagação de eventos.
- Rollback de versão com um comando.
- Detalhes em [`15-installer.md`](15-installer.md).

## 8. Indicadores-chave do produto (KPIs)

| KPI | Meta v1 |
|-----|---------|
| Tempo de instalação limpa | < 2 min |
| RAM em idle (plano de controle) | < 120 MB |
| SSD consumido (instalação limpa) | < 300 MB |
| CPU em idle | ≈ 0 (evento-driver) |
| Tempo de update | < 60 s com zero downtime |
| Tempo de rollback | < 30 s |
| Processos residentes | ≤ 6 |
| Migração Coolify → Aether | < 1 dia |
| Migração Dokploy → Aether | < 1 dia |

Números detalhados e justificados em [`03-metas-engenharia.md`](03-metas-engenharia.md).
