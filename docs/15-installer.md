# 15 — Installer

> **Status:** Instalação, atualização, rollback, desinstalação.
> **RFC relacionada:** [`RFC-0009`](rfc/RFC-0009-installer.md)
> **Princípios atendidos:** P2 (mínimo necessário), P6 (simplicidade operacional), P9.

---

## 1. Propósito

Um instalador inteligente que roda em **servidor limpo**, detecta o ambiente, instala **apenas
as dependências necessárias**, cria serviços e deixa a plataforma pronta. Também cuida de
atualização, rollback e desinstalação.

## 2. Experiências-alvo

```bash
# instalar
curl -fsSL https://get.aether.sh | sh

# atualizar
aether update

# desinstalar
aether uninstall
```

## 3. Fases da instalação

```
Fase 0 — Bootstrap (script bash mínimo)
   ├── detecta SO/distro/arch/init (systemd? glibc?)
   ├── verifica UID=0, arquitetura suportada
   ├── baixa binário `aether` (assinado) + instalador interno
   └── executa `aether install`
Fase 1 — Detect & plan
   ├── detecta recursos (CPU/RAM/disco)
   ├── detecta podman/buildah/skopeo presentes e versões
   ├── detecta conflitos (docker instalado? portas 80/443 usadas?)
   └── gera plano (o que instalar, o que pular)
Fase 2 — Install deps (mínimas)
   ├── instala podman, crun, conmon, buildah, skopeo, fuse-overlayfs
   ├── habilita user@.service (systemd user instances)
   └── (nenhuma imagem baixada)
Fase 3 — Provision
   ├── cria usuários: aether, aether-agent, aether-proxy (system, sem login)
   ├── cria diretórios em /var/lib/aether e /var/log/aether
   ├── gera identidade (CA interna para mTLS core↔agent, chave mestra)
   ├── escreve units systemd: aether-core, aether-agent, aether-proxy
   └── daemon-reload + enable
Fase 4 — Init
   ├── inicializa banco (SQLite) + migrações
   ├── gera admin bootstrap token (impresso uma vez)
   ├── configura firewall/portas (80/443/22?) conforme política
   └── health check: tudo `active (running)`
Fase 5 — Done
   ├── imprime URL, token, próximos passos
   └── (opcional) faz post da telemetria de instalação (só se consentido)
```

## 4. Instalação de dependências — política

- Instala **somente o que falta** (idempotente).
- Versões mínimas exigidas (ex.: podman ≥ 4.6, crun ≥ 1.8).
- Usa o gerenciador de pacotes da distro (`apt`, `dnf`, `zypper`); fallback: binários estáticos
  oficiais do ecossistema (rpm/ostree) se a distro não tiver versão mínima.
- **Nenhum download de imagem de container durante a instalação.**

## 5. Atualização (`aether update`)

### 5.1 Modelo

- Binário único + migrações de banco versionadas.
- Atualização = `aether update` (ou disparada pelo core quando a rede permite; autoupdate
  desligável por padrão — P8).

### 5.2 Fluxo

```
1. verifica versão disponível (checksum + assinatura)
2. baixa binário novo para staging (/var/lib/aether/updates/)
3. valida checksum/assinatura
4. backup automático (SQLite VACUUM INTO + event log) em /var/lib/aether/backups/pre-update/
5. swap atômico (rename) de binário
6. migrações de schema (transacionais, idempotentes)
7. recarrega units systemd (daemon-reload, restart core)
8. propaga para agentes: agent.updated (cada agente baixa/valida/reinicia)
9. report: sucesso → novo estado; falha → rollback automático
```

### 5.3 Zero-downtime

- Core escuta via socket systemd (`ListenStream`); na troca, connections ativas continuam até
  o handoff.
- Migrações transacionais: se falhar, nada muda.
- UI serve SPA estático; swap de assets é atômico.

### 5.4 Rollback de versão

```
aether rollback [versão]
→ restaura binário anterior (staging preservado)
→ aplica migração reversa (por versão)
→ recarrega units
```

## 6. Desinstalação

```
aether uninstall [--purge]
→ remove units, usuários, sockets
→ (--purge) remove dados em /var/lib/aether e /var/log/aether
→ app units do usuário: PARADAS mas NÃO removidas por padrão (migração segura)
```

## 7. Instalação em ambientes variados

| Ambiente | Suporte | Notas |
|----------|---------|-------|
| Debian/Ubuntu | v1 | apt |
| Fedora | v1 | dnf |
| RHEL/CentOS Stream | v1 | dnf; selinux policy |
| openSUSE | v1 | zypper |
| Container (Docker-in-Podman?) | v2 | não suportado em v1 (self-host = host) |
| ARM (aarch64) | v1 | imagens/binários arm64 |

## 8. Identidade e segurança no bootstrapping

- CA interna (`aether-ca`) gerada na instalação: emite certs para core↔agent (mTLS).
- Chave mestra (KEK) gerada; usada para criptografar secrets (ver [`16-seguranca.md`](16-seguranca.md)).
- Bootstrap token impresso uma vez (acesso inicial ao admin).

## 9. Riscos e mitigação

| Risco | Mitigação |
|-------|-----------|
| Distro sem versão mínima de podman | binários estáticos oficiais |
| Portas 80/443 ocupadas | detectar e orientar (ou proxy em porta alta + redirect) |
| Instalação interrompida | idempotência em todas as fases; retomada |
| Update corrompe banco | backup automático + migrações transacionais + rollback |
| Dependency conflict (docker presente) | não instalar/desabilitar docker; documentar coexistência |
| SELinux | policy customizada (Fedora/RHEL) |

## 10. Decisões

- **Decisão:** instalador bash mínimo + binário interno (`aether install/update/uninstall`).
- **Decisão:** zero download de imagem na instalação.
- **Decisão:** atualização atômica com backup automático e rollback.
- **Decisão:** autoupdate off por padrão; P8.

## 11. Referências

- RFC-0009 (Installer).
- Metas operacionais: [`03-metas-engenharia.md`](03-metas-engenharia.md) §3.
- Segurança: [`16-seguranca.md`](16-seguranca.md).
