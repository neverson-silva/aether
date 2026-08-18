# Auditoria Profunda da UI + Plano de Migração para Aceternity UI

> Documento de diagnóstico e planejamento. **NENHUMA alteração de código foi feita.**
> Fonte: `web/` — 92 arquivos TSX, ~14.5k LOC, React 18.3 + Vite 5 + TanStack Router/Query + Tailwind v4.

---

## 1. Estado atual (arquitetura visual)

**Stack**: React 18.3 · Vite 5 · TanStack Router (file-based) + TanStack Query · react-hook-form + zod · axios · Tailwind **v4** (`@tailwindcss/vite`, `@theme` em `web/src/styles.css`) · sem shadcn/Radix · sem Framer Motion/Motion · sem Aceternity · sem i18n.

**Design system atual** (`src/styles.css` + `src/components/ui/`):
- Tokens `@theme` Material-3-like (dark-first): `surface*`, `surface-container*`, `on-surface*`, `primary #568dff`, `error #ffb4ab`, `outline-variant #424655`, `--radius-DEFAULT .5rem`, fontes Inter/JetBrains Mono, `font-label-caps/code-md/body-md` utilitários próprios.
- **18 componentes** em `ui/`: `button`, `card`, `modal` (+`ConfirmDialog`), `form` (`Input/Select/Field`), `table`, `status` (`StatusPill`), `toast`, `feedback` (`Popover/Spinner/EmptyState/CodeBlock`), `card-menu`, `metrics`, `runtime-status`, `cn`, e os primitives de página `AppPage/AppSection/AppToolbar/AppBadge/AppStatCard/AppEmptyState/AppSkeleton/AppCard/AppLink`.
- **Terminal real**: `@xterm/xterm` com addons `fit`, `search`, `web-links`, `unicode11` (`-components/Terminal.tsx`) — já é um terminal funcional de verdade (não demo).
- **Logs**: `LiveLogs.tsx` (SSE) com strip ANSI, JSON pretty-print, classificação level/tag, presença; `DeploymentLogModal.tsx` (poll 2.5s).

---

## 2. Problemas encontrados (por severidade)

### CRÍTICO
1. **Deploy log sem produto**: `DeploymentLogModal` é um `<div>` raw com cores hardcoded (`bg-[#0a0a0a]`, `text-[#d1d5db]`), **sem auto-scroll, pause, follow, search, virtualização** — uma linha por deploy com milhares de linhas trava o DOM. É a tela mais usada do produto.
2. **Cores hardcoded em massa**: ~**350+ hex inline** em TSX (sucesso `#4ade80` ×58, warn `#fbbf24` ×36, info `#60a5fa` ×16, `#050505`/`#0a0a0a` code-bg ×20, `#a78bfa`…). Cada tela inventa sua paleta → impossível re-temar.

### ALTO
3. **Sidebar custom com estilo básico** (`shell.tsx`): sem estado ativo refinado, sem colapso animado, sem badges/busca no nav; header com `glass-panel` fixo.
4. **Wizards raw** (`ApplicationWizard`, `CreateServiceModal`, `Compose/Template/DatabaseWizard`): divs com `className` gigantes, `style={{ fontVariationSettings… }}`, toggles manuais — 5 implementações quase iguais de "card selecionável".
5. **Cards/badges/loaders duplicados**: padrões `bg-surface-container rounded-xl border border-outline-variant` repetidos ~200× manualmente; spinners/`animate-spin` reinventados.
6. **`style={{...}}` inline** em dezenas de pontos (ícones Material Symbols com `fontSize`/`fontVariationSettings`).

### MÉDIO
7. **Acessibilidade frágil**: icon-buttons sem `aria-label`, toggles sem `role`, modal raw do wizard sem foco-trapping (o `Modal` do kit tem Escape mas não focus trap), contraste inconsistente (`#d1d5db` sobre `#0a0a0a` ok; `on-surface-variant/50` às vezes baixo).
8. **Responsividade parcial**: grids `grid-cols-1 xl:grid-cols-3` ok, mas tabelas sem overflow-strategy unificada, terminal sem mobile strategy documentada, sidebar mobile com overlay raw.
9. **Motion quase inexistente**: só `animate-fade-in/modal-pop` e `pulse-dot`; hovers inconsistentes; sem microinterações de status.

### BAIXO
10. **Feedback inconsistente**: toasts removidos em massa (por pedido), EmptyStates bons (`AppEmptyState`) mas aplicação desigual; `DeploymentLogModal` "Loading logs..." cru.

---

## 3. Componentes raw mapeados (atenção)

| Componente | O que faz | Onde | Genérico? | Duplicado? | Aceternity equiv. |
|---|---|---|---|---|---|
| `shell.tsx` sidebar+header | nav global | todas as rotas | sim | — | **Sidebar** (`ui.aceternity.com/components/sidebar`) |
| `ApplicationWizard` step 2 (Build Method) | seleção de cards | wizard | sim | ×5 wizards | **Focus Cards / Tabs** |
| `DeploymentLogModal` | log de deploy | deploy page | sim | — | **Terminal** + custom (log viewer) |
| `LiveLogs` rows | classifica linhas | app page | parcial | — | custom (viewer) |
| wizards (inputs/toogles) | forms | criação | sim | ×5 | **shadcn/Radix** inputs+toggle |
| `AppStatCard`/`MetricCard` | métricas | dashboards | sim | ok | **Bento Grid** (opcional) |
| `CardMenu` | menu de contexto | cards | sim | — | **DropdownMenu** (Radix) |
| spinner/loading states | loaders | global | sim | ×8 variações | **Loader** (aceternity) + skeleton |
| `status.tsx` StatusPill | badges | global | sim | ok | shadcn Badge + custom dot |
| `modal.tsx` | dialog | global | sim | — | Radix Dialog (base) |
| `toast.tsx` | feedback | global | sim | — | **sonner** (shadcn) |
| inline icons `material-symbols-outlined` | ícones | global | sim | via fontSize inline | manter (sistema de ícones ok) |

---

## 4. Aceternity mapping (Atual → Aceternity → decisão)

| Atual | Aceternity | Decisão |
|---|---|---|
| Sidebar custom | Sidebar | ✅ **adotar** (colapsável, com ícones + conteúdo animado) |
| Deploy log raw | Terminal | 🟡 **parcial**: usar a estética do Terminal Aceternity como referência visual, mas manter/evoluir o **viewer próprio** (virtualizado, com pause/follow/search) |
| Terminal xterm | Terminal (demo) | ❌ **não substituir** — xterm é real (pty+ANSI+search); apenas re-envolver em shell premium (header live/pause/search/copy) |
| Cards selecionáveis (wizards) | Focus Cards / Hover Border | ✅ usar **Hover Border Glow** + primitives shadcn |
| MetricCards | Bento Grid | 🟡 **opcional** — só se a densidade justificar |
| Loaders/spinners | Loader | 🟡 adotar só skeleton + loader discreto (sem "showcase") |
| Empty states | — | ✅ manter `AppEmptyState` (já bom) |
| Tabs | Tabs (aceternity) | ✅ adotar para tab-bars de página (com conteúdo animado) |
| Backgrounds de página | Spotlight / Aurora | 🟡 **usar com parcimônia**: 1-2 backgrounds sutis (login/onboarding), nunca em listas |
| CodeBlock | CodeBlock | ✅ adotar para previews de Dockerfile/nginx (wizard) |
| Timeline | Timeline | 🟡 deploy history (opcional) |

**Regra**: Aceternity alimenta **primitives e microinterações**; não vira decoração em listas/abas de alta frequência.

---

## 5. Terminal — análise e plano de redesign

**Hoje**: xterm real (funciona: resize via WS, search addon, shell selector, reconexão, status dot). Problemas: **não é "produto"** — sem barra de controle fixa, sem pause/follow para logs de deploy, sem copy/clear visíveis, sem atalhos documentados, estética crua (container genérico).

**Plano**:
1. **Shell premium** (barra sticky): `● LIVE` indicator + `[Pause] [Clear] [Copy] [Search]` + shell select + status (connecting/reconnecting/disconnected) + botão reconectar.
2. **Manter xterm** com addons já presentes + `fit` no resize (ResizeObserver) + `scrollback` configurável.
3. **Auto-focus** no mount, `Ctrl+Shift+F` focus search, `Ctrl+L` clear.
4. **Mobile**: altura responsiva (`calc(100dvh - header)`), touch scroll preservado, toolbar colapsada em ícones.
5. **Perf**: sem re-render do React por frame (eventos só atualizam o DOM do xterm); toolbar separada via memo.
6. **Acessibilidade**: `aria-live` para estado de conexão, foco no input após mount, contraste do tema xterm com tokens (não hardcoded).

---

## 6. Logs — análise e plano de redesign

**Hoje**: SSE (`/apps/:id/logs`), `LiveLogs` classifica (ANSI strip, JSON pretty, level/tag), mas **renderiza tudo** (sem windowing) e o `DeploymentLogModal` (poll) é cru.

**Plano — Log Viewer Premium** (`LogViewer` componente próprio):
- **Virtualização** (windowing ~200 linhas, `react-virtuoso` ou windowing manual) — **obrigatório** para milhares de linhas.
- **Controles sticky**: `LIVE ●` + `[Pause] [Resume] [Clear] [Download]` + `Search` (highlight, match counter, Enter/Shift+Enter) + filtros `INFO|WARN|ERROR` + colapso de `[tag]`.
- **Follow mode**: auto-scroll só quando o usuário está no fundo (detect scroll-top); pausa automática ao rolar para cima; badge "N novas linhas" para retomar.
- **Linhas**: timestamp + level com cor de token (não hex), tag colorida, JSON expandível (`<details>`/toggle), copy-line/copy-block no hover, `whitespace-pre-wrap` + quebra de linhas longas (ou horizontal scroll opcional).
- **Streaming**: buffer append + batch (16ms frame) + `useDeferredValue`; conexão com indicador live/offline + reconexão automática.
- **Deploy log**: migrar o `DeploymentLogModal` para o `LogViewer` com **auto-refresh** + botão `[Download .log]` + erro destacado.
- **Atalhos**: `⌘F` search, `Space` pause, `C` clear.

---

## 7. Design system proposto

```
Design Tokens (styles.css @theme)        ← fontes canônicas (NÃO mudar primary/radius)
      ↓
Primitives (Radix + cn + tokens)         ← Dialog, Dropdown, Tabs, Toggle, Tooltip, Label, Slot
      ↓
UI Components (shadcn-style, nosso kit)  ← Button, Input, Select, Card, Badge, Table, Skeleton, Toast(sonner), Modal
      ↓
Aether Components (Aceternity + custom)  ← Sidebar, TerminalShell, LogViewer, CodeBlock, FocusCard, StatusDot, EmptyState
      ↓
Domain Components                        ← DeploymentStatus, EnvEditor, ServiceCard, WizardStep, HealthCheckBadge
      ↓
Pages
```

- **Cores**: consolidar em tokens semânticos no `@theme`: `--color-success`, `--color-warning`, `--color-info`, `--color-code-bg`, `--color-code-fg` — **erradicar** os ~350 hex inline (os 5 mais usados: `#4ade80`→success, `#fbbf24`→warning, `#60a5fa`→info, `#050505`/`#0a0a0a`→code-bg, `#a78bfa`→purple-accent opcional).
- **Tipografia**: manter escala existente (`font-display-lg/headline/body/code/label-caps`); adicionar `text-[12px]` code como token `code-xs`.
- **Radius/shadows/spacing**: manter canônicos (`.5/.625/.75rem`) — **proibido mudar** (AGENTS.md); definir `shadow-*` de 3 níveis.
- **Motion**: tokens `--duration-*` e `--ease-*`; usar apenas: entrance leve (fade+y 4px), hover borders, status pulse, tab content swap; **nada contínuo**.
- **Motion lib**: `motion` (framer-motion v11) como única lib de animação.

---

## 8. Plano de migração

### P0 — Core UX (terminal, logs, nav, deploy)
1. Instalar/configure: `motion`, `@radix-ui/*` (dialog, dropdown, tabs, toggle, tooltip, scroll-area), `sonner`, `react-virtuoso`, Aceternity (CLI `npx aceternity-ui init` — components-sidebar) com Tailwind v4.
2. **Tokens semânticos** + script de limpeza dos hex mais comuns.
3. **`LogViewer`** (virtualizado) → substituir `DeploymentLogModal` e integrar em `LiveLogs`.
4. **`TerminalShell`** (barra premium) em volta do xterm.
5. **Sidebar** Aceternity (colapsável) + header refinado.
6. Deploy page: status timeline + log viewer embutido; tab-bar com Tabs.

### P1 — Design System
7. Primitives Radix (Dialog/Dropdown/Tabs/Toggle/Tooltip/ScrollArea) atrás do kit.
8. Unificar `Button/Input/Select/Card/Badge/Table/Skeleton` (shadcn-style, tokens).
9. Wizards → `FocusCard`/`HoverBorder` + forms unificados; `CodeBlock` para previews.
10. Toast → sonner (mantendo `useToast` como facade).

### P2 — Polish
11. Empty/loading/error states consistentes (Skeleton + EmptyState + AlertBlock).
12. Microinterações: hover borders em cards, glow sutil no primary, tab transitions, status pulses.
13. Backgrounds sutis (login/onboarding com Aurora/Spotlight discreto).
14. Remover componentes raw substituídos + CSS morto (`styles.css` utilitários órfãos) + `spikes/` refs.

---

## 9. Riscos

| Risco | Mitigação |
|---|---|
| **Performance** (logs/terminal) | virtualização + batch + memo; nenhuma animação em listas; `will-change` controlado |
| **Regressão funcional** | FASE 11: APIs/hooks/WS/SSE/polling/routing intactos; migração é presentation-layer; testes `tsc` + `npm run build` a cada fase; revisão visual por tela |
| **Acessibilidade** | Radix dá focus-trap/aria; auditoria de contraste pós-migração; `focus-visible` em todos os novos interativos |
| **Dependências** | Aceternity + Radix + motion + virtuoso adicionam bundle — código-split por rota (Vite), lazy nos componentes pesados (sidebar/logs); monitorar tamanho do chunk |
| **"Showcase de animações"** | regra dura: motion só resolve problema de UX; revisão crítica pós-P2 |
| **Tokens** | NÃO alterar canônicos (primary/radius/fonts — AGENTS.md); só ADICIONAR tokens semânticos |
| **consistência** | hierarquia tokens→primitives→ui→aether→domain→pages; proibir hex inline a partir do P1 (lint) |

---

## 10. Resultado esperado

- Primeira impressão: "produto de infraestrutura premium" — sidebar colapsável com navegação refinada, deploy page com terminal/log viewer de verdade (live, virtualizado, pause/follow/search/filtros), tabs suaves, cards com hover-border sutil.
- Consistência: zero hex inline; toda cor vinda de tokens semânticos; um único `Button/Input/Card/Badge/Skeleton` no app.
- Performance: logs com 100k+ linhas fluido (windowing), terminal sem re-render de React, bundle com code-split.
- Arquitetura: novo dev cria tela seguindo tokens→primitives→ui→aether; menos duplicação; CSS morto removido.
- Aceternity presente como fonte de primitives/microinterações, **não** como tema dominante.

---

*Documento de auditoria — aguardando aprovação para iniciar a Fase de Implementação (P0).*
