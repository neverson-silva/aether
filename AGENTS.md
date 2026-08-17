# AGENTS.md — Regras de Trabalho no Aether

## Inglês First — NUNCA Português no Código/UI

Este app (frontend e backend) é **english-first**:

- Todo o código, logs, mensagens de erro, comentários, nomes de variáveis/funções/rotas,
  mensagens de evento (ex.: `deploy.queued`, notificações), labels, placeholders, tooltips,
  mensagens de toast e qualquer texto exibido ao usuário devem estar em **inglês**.
- **PROIBIDO** conter palavras em português no código-fonte, UI ou textos gerados pelo app,
  **a menos que o usuário peça explicitamente**.
- Se ao escrever código você gerar texto em português (ex.: mensagem de erro, label de
  botão, título de modal), reescreva para inglês antes de finalizar.

## Design System, Sidebar e Botões: NÃO MEXER (a menos que Explicitamente Solicitado)

O design system (tokens de cor, radius, tipografia, fontes) e todos os componentes de UI
(sidebar, botões, app-link, card-menu, org-switcher, modais, inputs, tabelas) são parte
da identidade visual do produto.

**PROIBIDO**, salvo solicitação explícita do usuário:
- Alterar `web/src/styles.css` (tema/`@theme`, cores, `--radius-*`, fonts).
- Alterar a estrutura, labels ou estilização da sidebar (`web/src/components/shell.tsx`).
- Alterar cores/radius/variants de botões (`web/src/components/ui/button.tsx`), links
  (`app-link.tsx`) ou qualquer componente em `web/src/components/ui/`.
- Adicionar/remover fontes no `package.json` (ex.: trocar Inter por outra) sem pedido.
- "Reaproveitar" o tema de uma página mockup/HTML de referência para sobrescrever o
  tema global do app. O tema global NÃO deve ser mudado como efeito colateral de
  refatorar uma tela.

Quando refatorar uma tela específica:
- Use SOMENTE os tokens existentes do design system.
- Não introduza cores/radius hardcoded que divergem do tema.
- Foque no conteúdo/estrutura da tela em questão; não altere o shell global.

Os tokens canônicos (não mudar):
- `primary`: azul Material (`#b0c6ff` no dark).
- `--radius-DEFAULT`: `0.5rem`, `--radius-lg`: `0.625rem`, `--radius-xl`: `0.75rem`.
- Font do app: Inter (`--font-*`).

### Exceções
SE e SOMENTE SE o usuário pedir explicitamente (ex.: "mude o primary", "reset o radius",
"altera o botão X"), pode-se alterar. Sempre confirme o escopo antes.

## Banco de Dados — NUNCA Limpar

- Regra permanente: NUNCA `DROP/CREATE/TRUNCATE/DELETE` dados reais de produção.
- Nunca `podman rm -f $(podman ps -aq)` nem `--filter name=aether-` (pega infra).
- Sempre operar containers por nome específico.
- Ler banco é permitido; escrever só com autorização explícita.

## Arquitetura Realtime / Event-Driven — Proibido Polling

A plataforma é **event-driven e realtime**. O frontend NÃO deve buscar estado por polling.

### Como o realtime funciona
- **Postgres é a fonte de verdade** dos estados; o Redis é o barramento realtime:
  pub/sub (`notify:org:<org>`) para fanout + Streams para event log (`ev:org:<org>`, seq por org)
  e fila de deploys (`q:deployments:*`, consumer groups).
- **Frontend**: um WebSocket único (`/api/v1/ws/realtime`) por sessão, via
  `RealtimeProvider`/`NotificationProvider`; bootstrap inicial por REST e atualizações
  incrementais por eventos WS. Protocolo: subscribe por escopo autorizado, `seq` para
  replay (persistido em `localStorage`), eventos **efêmeros** (ex.: `deploy.build.log`,
  `app.state`) com seq=0 que não vêm no replay.
- Servidor envia ping/heartbeat; o cliente deve responder `{"op":"ping"}` a cada ~25s
  (read timeout do hub é 45s). O `Timeout` middleware NÃO deve ser aplicado a paths `/api/v1/ws/`.
- Backend: `AETHER_RUNTIME_BACKEND=redis` em produção e no dev (`dev.sh`/`install.sh`);
  `memory` só para testes.

### Regra anti-polling
- **PROIBIDO** adicionar polling/`setInterval`/`refetchInterval` no frontend para dados
  que já são entregues por evento WS (deploys, app states, notificações).
- Polling é permitido SOMENTE para telemetria de baixa frequência ou fallback quando o
  WS está desconectado, e deve ser revisado com o usuário antes.
  Já aceitos: net-q 15s, presence 30s/10s, host-stats 2s, SSE de follow de logs,
  fallback de notificações 30s (só quando offline).
- Dúvida → consulte o usuário antes de introduzir polling.

## Containers / Infra

- API, frontend, postgres e redis rodam em containers podman (via `install.sh`).
- `podman` é a única dependência do host.
- Após mudanças de código: rebuild via `./install.sh start` (builda imagem + reinicia).
- Máquina podman: 4GB (se OOM em builds, aumentar).

## Testes

- Suite: `AETHER_TEST_DATABASE_PORT=5433 AETHER_API_TEST_DATABASE_PORT=5433 go test ./internal/... -count=1 -p 1 -timeout 25m`
- Postgres de teste: container `aether-test-pg` na porta 5433 (não derrubar).
- Redis de teste: container `aether-redis-test` na porta 6380.
- Rodar `go build ./...` + `go vet ./internal/...` antes de finalizar.
