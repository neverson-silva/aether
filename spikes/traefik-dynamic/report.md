# SPIKE-TRAEFIK — Config dinâmica via API em memória

> **Status:** Concluído ✓
> **Data:** 2026-08-02
> **Host:** macOS arm64, Traefik 3.7.10 (binário nativo), Go 1.26.5
> **Hipótese (H5):** a configuração do proxy pode ser aplicada **em memória** (provider HTTP,
> servida pelo Networking Engine), sem escrita de arquivo por deploy e sem restart — validando
> a RFC-0002 (config dinâmica em memória) e a mitigação D4.

---

## 1. Método

- **`config-server` (Go)**: simula o Networking Engine servindo o `ProxyConfig` em memória em
  `:18090/traefik`. Adiciona uma segunda rota (`spike2.local`) **ao vivo** — representando um
  deploy que adiciona um domínio.
- **Traefik**: static config (`traefik.yml`) com entrypoint `web` (:18081), API dashboard
  (:18082) e provider `http` apontando para o config-server (pollInterval 1s).
- **Backend**: `python http.server` em :18099 (app fake).
- **Validações**: (1) rota inicial responde; (2) rota adicionada ao vivo responde sem restart;
  (3) nenhum arquivo dinâmico escrito em disco; (4) API rawdata mostra os routers ativos.

## 2. Resultado

| Etapa | Resultado |
|-------|-----------|
| Rota inicial `spike.local` → backend | ✅ respondeu (diretório do backend) |
| Rota `spike2.local` adicionada AO VIVO | ✅ respondeu, **sem restart e sem arquivo** |
| Arquivos dinâmicos criados no deploy | ✅ **nenhum** (`traefik.yml` antes = depois) |
| API rawdata (routers ativos) | ✅ `router-a@http`, `router-b@http` — config aplicada em memória |

## 3. Análise

1. **H5 CONFIRMADA.** O Traefik aplicou `router-b` segundos após o config-server alterar o
   payload em memória — pollInterval 1s foi suficiente. A mudança de rota não tocou disco
   além do `traefik.yml` estático (que nunca é reescrito em deploy).
2. **O padrão funciona para a arquitetura**: o Networking Engine mantém o `ProxyConfig` em
   memória e o provider HTTP do Traefik o consome. Isso elimina o I/O por deploy que os
   concorrentes pagam (provider file do Coolify/Dokploy → D4).
3. **Custo do provider http**: pollInterval de 1s = 1 GET leve/s do Traefik ao config-server —
   CPU desprezível. Alternativa ainda mais barata: `pollInterval: 5s` (latência de deploy
   config tolerável) — configuração do platform.
4. **rawdata via API** (`/api/rawdata`) é a fonte de verdade observável para o dashboard de
   rotas do painel — sem custo extra.

## 4. Conclusão

**H5 CONFIRMADA.** O padrão "config em memória via provider HTTP" é viável, barato e elimina
I/O por deploy. A RFC-0002 está correta ao definir config dinâmica em memória e API para
status/rawdata.

## 5. Recomendações de ADR (para RFC-0002)

- **Confirmado**: provider HTTP como mecanismo de config dinâmica; `pollInterval` default 5s
  (configurável) com refresh imediato via `traefik reload`/API se necessário.
- Networking Engine deve expor: (a) endpoint de config dinâmica (o "provider http"), (b)
  leitura de `/api/rawdata` para reconciliação/status.
- Reconciliation no boot: Traefik parte do provider http (nada em disco) → nenhum estado
  persistido no proxy (exatamente a RFC-0002).
- Nota: `pollInterval` é polling **de infra, no proxy** — não é polling do core (P5 preservado).

## 6. Rerun

```bash
cd spikes/traefik-dynamic && bash run.sh
```
