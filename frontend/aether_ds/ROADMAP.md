# Aether Design System Roadmap

## Visão

Este roadmap define a superfície de componentes do Aether Design System. Ele cobre desde fundamentos tipográficos até experiências complexas de operação de infraestrutura. O objetivo não é reproduzir o shadcn/ui literalmente: usamos seu inventário atual como baseline de cobertura e reescrevemos a linguagem, semântica, API, motion e estados para o contexto de um PaaS.

O Aether precisa funcionar para três ritmos diferentes:

- **Leitura rápida:** status, métricas, alertas, tabelas e eventos.
- **Construção:** formulários, configuração, filtros, seleção de recursos e deploy.
- **Operação:** logs, comandos, ações destrutivas, ambientes, realtime e recuperação de falhas.

Cada componente deve ser avaliado por utilidade operacional, não por quantidade de props. Complexidade visual só entra quando melhora entendimento, orientação ou controle.

## Princípios do roadmap

- Construir fundamentos antes de componentes que dependem deles.
- Resolver primeiro os caminhos mais frequentes do PaaS.
- Usar HTML nativo quando ele já entrega semântica e comportamento corretos.
- Usar `@base-ui/react` como base real de comportamento para Button, Toolbar e futuras Dialog, Popover, Menu, Select, Combobox e Tooltip; não manter a dependência apenas como possibilidade teórica.
- Usar `tailwind-variants` para variants tipadas, centralizadas e documentadas.
- Preferir APIs prop-driven para componentes simples e médios; composition é exceção para estruturas complexas.
- Cada componente deve ter um arquivo de implementação próprio; arquivos monolíticos como `foundation.tsx` são proibidos.
- Implementar estado, teclado, foco, overflow, loading, erro e reduced motion junto com o componente.
- Toda etapa deve produzir stories que demonstrem intenção, não apenas aparência.
- Um componente só é promovido para a API pública quando sua anatomia e seu contrato estiverem estáveis.
- A árvore do Storybook deve usar apenas `Foundations`, `Components`, `Forms`, `Navigation`, `Overlay`, `Data`, `Feedback` e `Patterns`; intenção de produto é story, não categoria.

## Legenda

- **P0:** fundação ou bloqueador de vários componentes.
- **P1:** componente recorrente para o produto principal.
- **P2:** componente importante para fluxos complexos ou escala.
- **P3:** componente especializado, avançado ou de exploração.
- **Base:** primitive sem estilo ou com estilo mínimo.
- **Compound:** API composta por partes nomeadas.
- **Pattern:** solução de fluxo ou composição, não apenas um elemento.

## Status de entrega

O roadmap usa quatro estados para evitar que exportação ou existência de uma story seja confundida com prontidão de produção:

- **Production-ready:** contrato, comportamento, acessibilidade, temas e estados principais cobertos.
- **Implemented:** implementação funcional do contrato previsto, ainda podendo receber endurecimento de escala ou integração.
- **Scaffold:** anatomia e caminho visual existem, mas faltam comportamentos relevantes do contrato.
- **Missing:** item previsto ainda não possui implementação pública.
- **Blocked by dependency:** o contrato depende de uma primitive ou decisão ainda não estabilizada.

Estado atual após a auditoria:

- **Production-ready:** Button, Input, Field, Badge, Avatar, Typography, Calendar, DatePicker, DateRangePicker, TimePicker, DateTimePicker, NativeSelect, Attachment, Message, MessageScroller, Bubble, Marker, Gauge, CodeEditorLite, Toast, Sonner, Dialog, Popover, Tooltip e primitives de tema.
- **Implemented:** Chart, DiffViewer, Spotlight, HoverCard, DataTable, Select, Combobox, FileUpload, NotificationStack e os componentes de fluxo com contratos básicos.
- **Scaffold:** DataGrid, FormBuilder, DeploymentComposer, VariableEditor, ResizableDashboard, TimelineScrubber, VirtualizedList, DragAndDrop e RealtimeActivitySurface.
- **Missing:** nenhum item de componente listado como ausente na auditoria anterior. Os itens que ainda aparecem como scaffold continuam exigindo evolução antes de serem promovidos.
- **Blocked by dependency:** nenhum bloqueio técnico conhecido; melhorias de virtualização, grid drag e rich editor devem ser feitas sem quebrar a API pública atual.

Uma implementação só pode ser promovida quando cumprir o contrato completo da seção correspondente e tiver sido validada em light, dark, reduced motion, teclado, conteúdo extremo e estados de erro/loading/empty aplicáveis.

## Fase 0 — Fundação visual e comportamental (implementada)

### P0.1 Tipografia

**Tipo:** Base

Escalas de display, heading, body, label, caption e code com Inter e JetBrains Mono. Deve cobrir truncation, wrap, conteúdo longo, números tabulares, links, inline code, keyboard keys e hierarquia de documentação.

**Variants:** `level`, `tone`, `weight`, `truncate`, `mono`, `align`.

**States:** default, muted, disabled, danger, warning, success, link, inline e code.

### P0.2 Tokens e Theme Provider

**Tipo:** Base

Primitives, semântica, contracts de componente, light/dark, escopo de tema, persistência opcional, `color-scheme` e preferência do sistema.

**Critérios:** nenhum componente pode depender de valor bruto; mudança de tema não pode alterar layout nem remover foco.

### P0.3 Layout primitives

**Componentes:** `Box`, `Stack`, `Inline`, `Grid`, `Container`, `Bleed`, `Divider`, `VisuallyHidden`.

**Variants:** direção, gap, alinhamento, distribuição, largura e responsividade. Não criar abstrações para cada layout de uma página.

### P0.4 Ícones e ícones de estado

Biblioteca padrão: `@phosphor-icons/react`. A família oficial deve ser a única fonte de ícones do Aether, com exportação de tipos `Icon`, `IconProps` e `IconWeight` pela biblioteca.

Definir catálogo de ícones operacionais para deploy, rollback, service, environment, branch, database, container, server, cloud, terminal, code, activity, warning, error, success, lock, user, team, settings, search, filter, sort, copy, external link e navigation.

Definir regra de weights: `regular` como default, `bold` para ênfase ou ação selecionada, `fill` para estados sólidos, `duotone` apenas quando a camada adicional melhorar leitura e `thin` apenas em contextos de baixa densidade. Validar tamanho, alinhamento, contraste, optical weight e estado de foco. Ícones não podem ser o único canal de feedback.

Componentes devem receber ícones por props (`icon`, `leadingIcon`, `trailingIcon`) e cuidar internamente de tamanho, gap, alinhamento e `aria-hidden`. Não criar uma segunda família de ícones para uma tela específica. O pacote legado `phosphor-react` não deve ser usado; o pacote oficial é `@phosphor-icons/react`.

### P0.5 Motion system

Durações, easing, entrada, saída, hover, press, focus, skeleton shimmer, popover, dialog, drawer, command palette, toast, troca de tema e transições de superfícies.

Toda animação precisa ter versão reduzida ou estática. Movimento deve comunicar causa, continuidade ou mudança de estado.

## Fase 1 — Primitivas essenciais

### Decisão de API da fase

Componentes simples e médios devem ser consumidos por props. O consumidor não deve precisar montar uma árvore de subcomponentes para usar um botão, input, badge, avatar, empty state ou item comum.

Exemplos de contrato esperado:

```tsx
<Button icon={PlayIcon} iconPosition="start" loading={isDeploying}>
  Deploy
</Button>

<Input
  label="Service name"
  leadingIcon={SearchIcon}
  error={errors.name}
  clearable
  onClear={clearName}
/>
```

Ícones, labels, descrições, ações e estados devem ser props tipadas quando a estrutura for previsível. A implementação controla alinhamento, tamanho, gap, acessibilidade e estados visuais.

Composition API fica reservada para componentes cuja estrutura seja genuinamente composta, como Dialog, Menu, Tabs, Table avançada e Sidebar. Mesmo nesses casos, o roadmap deve prever presets prop-driven para os cenários mais frequentes.

### P0.6 Button

Variants `primary`, `secondary`, `quiet`, `danger`, `success`, `outline`; tamanhos `sm`, `md`, `lg`; largura full; ícone leading/trailing; loading; disabled; pressed.

**API preferencial:** `icon`, `iconPosition`, `loading`, `loadingLabel`, `fullWidth` e `as`. Não exigir `Button.Icon`, `Button.Label` ou wrappers para o uso normal.

**Implementação:** usar `@base-ui/react/button` para a primitive interativa e classes Tailwind registradas no `@theme`, como `bg-button-primary`, `text-button-secondary-foreground` e `border-button-secondary-border`. Valores `var(...)` arbitrários não devem aparecer na definição do componente.

Deve preservar label durante loading, impedir submissão duplicada, suportar `asChild` ou renderização como link somente quando a semântica continuar correta.

### P0.7 Button Group

Grupo de ações relacionadas, orientação horizontal/vertical, attached e separated, item ativo, overflow e navegação por teclado.

**Implementação:** usar `@base-ui/react/toolbar` para roving focus e comportamento de teclado.

### P0.8 Link

Link interno, externo, visited, disabled visual, external indicator, underline e foco. Diferenciar navegação de ação.

### P0.9 Icon Button

Botão de ação sem label visível, com `aria-label` obrigatório, tooltip opcional, tamanhos e estados de pressed/loading.

### P0.10 Badge

Status compacto, label, count, dot, icon, removable e live. Variants de intenção: neutral, info, success, warning, danger e accent.

### P0.11 Avatar

Imagem, iniciais, fallback, grupo, presença, tamanho, status, loading e erro de carregamento.

### P0.12 Kbd

Tecla individual e combinação de atalhos, adaptação para Mac/Windows/Linux, foco em command palette e navegação.

### P0.13 Separator

Horizontal, vertical, decorativo ou semântico. Não usar borda de card para qualquer separação.

### P0.14 Label

Label visível, required, optional, descrição associada, erro associado, disabled e composição com Field.

### P0.15 Skeleton e Spinner

Skeleton de texto, avatar, card, tabela e layout; spinner inline, button e página. Evitar skeleton que imita conteúdo inexistente de forma enganosa.

### P0.16 Progress

Progress determinado e indeterminado, percentual, label, status, erro, paused e progresso em etapas.

### P0.17 Alert e Inline Message

Informação persistente, success, warning, danger, neutral, ação inline, dismissible e mensagem associada a campo ou seção.

### P0.18 Empty State

Estado vazio inicial, sem resultados, sem permissão, falha de carregamento, recurso removido e estado filtrado. Deve sempre explicar contexto e próximo passo.

## Fase 1 — Formulários e entrada de dados (implementada)

### P1.1 Input

Text, search, email, password, number, URL, copyable, clearable, prefix, suffix, loading e validation state.

**API preferencial:** `label`, `description`, `error`, `leadingIcon`, `trailingIcon`, `clearable`, `onClear`, `loading`, `size` e `type`. Não exigir composição para label, hint, ícone ou mensagem de erro.

### P1.2 Textarea

Resize controlado, contador, autosize, code-like, erro, warning, descrição e conteúdo longo.

### P1.3 Input Group

Composição de input com prefix, suffix, button, icon, unit, command e validation message sem quebrar foco ou leitura.

### P1.4 Field

Primitive de formulário para label, description, control, error, hint, required, optional, orientation e estado disabled. Base para todos os campos.

### P1.5 Checkbox

Checked, unchecked, indeterminate, disabled, invalid, description, group e seleção em massa.

### P1.6 Radio Group

Seleção exclusiva, orientação, roving tabindex, descrição por opção, erro e opção desabilitada.

### P1.7 Switch

Booleano, loading, disabled, label à esquerda/direita, descrição e mudança assíncrona.

### P1.8 Toggle e Toggle Group

Pressionado, despressionado, single, multiple, toolbar, disabled, roving focus e seleção persistente.

### P1.9 Slider

Single, range, step, marks, tooltip de valor, input numérico conectado, keyboard control e valores formatados.

### P1.10 Select

Single select, multi-select, groups, descriptions, disabled options, loading, error, clear, empty e portal.

### P1.11 Select Search

Select com campo de busca interno, filtragem local, destaque de correspondência, no results, recent values e clear. Para catálogos remotos, usar o componente separado `AsyncSearchInput`.

### P1.11A Async Search Input

Busca remota prop-driven com `loadOptions(query)`, debounce, mínimo de caracteres, loading, erro, stale response protection, empty state, descrições de resultado, seleção por teclado e popup Base UI. Nunca exigir que o consumidor carregue o catálogo completo.

### P1.12 Combobox

Input + listbox, seleção single/multiple, freeform opcional, criação de item, async options, empty, loading, disabled options e keyboard navigation.

### P1.13 Date Picker

Data única, input manual, calendário, min/max, disabled dates, timezone, locale, error, clear e keyboard input.

### P1.14 Date Range Picker

Intervalo, início/fim, hover preview, presets, comparação, timezone, crossing month/year, invalid range e modo compacto para filtros.

### P1.15 Time Picker e Date Time Picker — Production-ready

12/24 horas, timezone, segundos, intervalo, teclado, locale, invalid time e integração com date range.

### P1.16 Calendar

Mês, ano, range, múltiplos meses, eventos, hoje, disabled, outside days, keyboard navigation e navegação rápida.

### P1.17 Input OTP

Código de autenticação, paste, auto advance, error, resend state, countdown e leitura assistiva.

### P1.18 File Upload e Attachment — Production-ready

Dropzone, seleção nativa, upload progress, preview, retry, cancel, erro de tipo/tamanho, múltiplos arquivos e remoção.

### P1.19 Form Actions

Barra de ações de formulário, dirty state, save, discard, pending, success, error e confirmação de navegação.

## Fase 3 — Overlay, menu e navegação contextual (implementada)

Todos os itens P1.20 a P1.32 desta fase possuem implementação React independente, story própria, API prop-driven nos casos comuns, Base UI para comportamento complexo, tokens semânticos, motion de entrada/saída e estados de teclado, foco, disabled, empty e reduced motion quando aplicável.

### P1.20 Tooltip

Tooltip para descoberta, delay, posicionamento, keyboard focus, touch fallback, conteúdo rico limitado e reduced motion.

### P1.21 Popover

Conteúdo contextual, anchor, collision handling, focus management, dismiss, nested popover e posicionamento responsivo.

### P1.22 Hover Card

Preview não essencial, delay, pointer grace area, fallback para teclado e touch sem esconder informação principal.

### P1.23 Dialog

Modal, non-modal quando necessário, title, description, actions, scroll interno, focus trap, escape, nested dialog e motion.

### P1.24 Alert Dialog / Confirm Dialog

Decisão de alto impacto, consequência explícita, ação destrutiva, loading de confirmação, erro e cancelamento seguro.

### P1.25 Drawer

Drawer lateral ou inferior, mobile bottom sheet, drag opcional, focus, scroll, snap points, dismiss controlado e motion espacial obrigatório. A entrada deve partir da borda de origem, o backdrop deve fazer fade, o fechamento deve preservar a presença até o fim da transição e swipe/drag devem acompanhar o ponteiro sem quebrar focus management.

### P1.26 Sheet

Superfície de edição ou detalhe, tamanhos, header/footer fixos, scroll, dirty state e ações persistentes.

### P1.27 Dropdown Menu

Ação contextual, grupos, submenu, checkbox item, radio item, shortcuts, destructive action e roving focus.

### P1.28 Context Menu

Menu por clique direito ou gesto equivalente, fallback de teclado, target contextual e ações dependentes do recurso.

### P1.29 Menubar

Navegação desktop com menus, roving tabindex, submenu, shortcuts e suporte a orientação.

### P1.30 Navigation Menu

Navegação ampla, grupos, active route, descriptions, mobile fallback e foco previsível.

### P1.31 Command Palette

Busca global, ações, navegação por teclado, grupos, comandos recentes, favoritos, loading assíncrono, empty state, breadcrumbs, subcomandos, atalhos e ações destrutivas.

Para o Aether, a command palette deve suportar:

- Ir para projeto, serviço, ambiente, deployment e log.
- Executar ações como deploy, rollback, scale e restart.
- Filtrar por tipo, status, ambiente e owner.
- Mostrar contexto, shortcut e risco da ação.
- Trabalhar offline com ações disponíveis localmente.
- Abrir resultados em nova região sem perder o contexto de busca.

### P1.32 Menus de usuário e workspace switcher

Conta, organização, projeto, ambiente, permissões, convite, sair e preferência de tema. Devem explicar o contexto atual e nunca depender apenas de avatar.

## Fase 2 — Estrutura de aplicação (implementada)

### P1.33 Card

Card básico, interactive, selectable, linked, metric, resource, danger, glass e elevated. Definir quando card é apropriado e quando uma lista ou tabela é melhor.

### P1.34 Item e Item Group

Linha ou bloco composto por media, title, description, metadata, actions e footer. Base para recursos, resultados de busca e configurações.

### P1.35 Breadcrumb

Hierarquia, truncation, overflow menu, current page, responsive collapse e aria-current.

### P1.36 Pagination

Offset, cursor, page size, first/last, disabled, loading, compact mobile e descrição para leitor.

### P1.37 Tabs

Manual/automatic activation, underline/pill, scroll horizontal, lazy content, deep link, disabled tab e keyboard navigation.

### P1.38 Accordion e Collapsible

Single/multiple, controlled/uncontrolled, nested, lazy content, disabled, animated height e reduced motion.

### P1.39 Sidebar

Sidebar expandida, collapsed icons, mobile sheet, sections, active item, nested navigation, command shortcut, badge, loading, permission-aware item e resizable width.

### P1.40 Top Bar e App Header

Breadcrumb, workspace, environment selector, global search, command trigger, notifications, theme switcher e user menu.

### P1.41 Resizable

Painéis redimensionáveis para logs, editor, métricas e detalhes. Deve ter min/max, persistência opcional, teclado e mobile fallback. O handle deve suportar pointer drag real em mouse e touch, pointer capture, estado visual durante dragging, cursor, limites, emissão contínua de largura e alternativa por teclado. A largura não pode depender apenas de `input[type=range]` nem de eventos de mouse isolados.

### P1.42 Scroll Area

Scroll estilizado, horizontal, vertical, shadow indicators, keyboard, nested scroll e virtualized content boundary.

### P1.43 Sheet Sidebar e Mobile Navigation

Fluxo de navegação móvel equivalente ao desktop, sem duplicar itens ou perder estado ativo. O drawer precisa de entrada e saída suaves em sua direção espacial, fade do backdrop, feedback de trigger, fechamento por Escape e outside press, preservação de foco e fallback correto para reduced motion.

## Fase 4 — Dados, métricas e infraestrutura (implementada)

Todos os itens P1.44 a P1.59 desta etapa possuem implementação React independente, story própria, contratos tipados, estados operacionais, semântica para dados, motion controlado e suporte a light/dark quando aplicável. O identificador histórico dos itens permanece P1.x para preservar as dependências do roadmap.

### P1.44 Table

Tabela semântica, alinhamento, densidade, sticky header, row states, empty, loading, error, responsive strategy, copyable cells e ações.

### P1.45 Data Table

Sorting, filtering, column visibility, column resize, pinning, row selection, bulk action, pagination, cursor loading, URL state e virtualization.

### P1.46 Data Grid

Grid de alta densidade para logs, métricas e recursos, com keyboard navigation, range selection, inline edit, column virtualization e performance controlada.

### P1.47 Filter Bar

Filtros simples, advanced filters, chips, saved views, clear all, active count, URL serialization, mobile drawer e loading de opções.

### P1.48 Sort Control

Ascending, descending, none, multi-sort, keyboard, accessible description e persistência na URL.

### P1.49 Metric Card

Valor principal, unidade, delta, período, trend, target, status, sparkline, stale, loading e empty.

### P1.50 Chart — Implemented

Linha, área, barra, stacked, donut, gauge, scatter e composed chart. Tooltips acessíveis, legenda, range, crosshair, loading, empty, no data, erro e modo reduzido.

### P1.51 Chart Tooltip e Legend

Hover, focus, keyboard equivalent, múltiplas séries, unidades, precisão numérica e exportação de contexto.

### P1.52 Progress Ring e Gauge — Production-ready

Percentual, threshold, status, indeterminate, label central, screen reader value e motion controlado.

### P1.53 Timeline

Eventos de deployment, audit log, incidentes e mudanças, com status, actor, timestamp, agrupamento, expansão e realtime.

### P1.54 Log Viewer

Monoespaçado, line numbers, wrap, follow tail, pause, search, filter, copy, download, severity, timestamps, virtualization, loading e desconexão.

### P1.55 Code Block e Code Editor Lite — Production-ready

Comando copiável, syntax highlight, line selection, diff, expand/collapse, erro por linha e ações de execução.

### P1.56 Copy Button

Copy success, failed, permission denied, fallback, tooltip e conteúdo sensível com reveal controlado.

### P1.57 Status Dot e Runtime Status

Healthy, deploying, degraded, failed, paused, unknown, offline, live pulse e status textual obrigatório.

### P1.58 Resource Tree

Árvore de projetos, serviços, ambientes e recursos; expand/collapse, lazy loading, selection, search, context menu, badges e empty branches.

### P1.59 Audit Log

Actor, ação, recurso, timestamp, diff, request ID, filtros e detalhe expandido.

## Fase 6 — Feedback, notificações e resiliência (implementada)

Os itens P1.60 a P1.66 possuem implementação React independente e stories. Sonner foi separado de NotificationStack nesta etapa; queue avançada, agrupamento e atualização transacional continuam sendo critérios de promoção quando ainda não estiverem cobertos pelo contrato atual.

### P1.60 Toast

Feedback transitório com title, description, action, dismiss, duration, persistent, queue, pause on hover, live region e prioridade.

### P1.61 Sonner / Notification Stack

Stack de notificações, posição por viewport, agrupamento, progress timer, atualização da mesma notificação, rich content e limite de simultaneidade.

Toast é feedback curto de uma ação. Notification é informação que pode exigir consulta posterior. Não misturar os dois contratos.

### P1.62 Banner

Aviso persistente de manutenção, degradação, billing, permissão, migração e indisponibilidade.

### P1.63 Inline Error

Erro de campo, erro de seção, erro de recurso e erro global com retry, request ID e ação de suporte.

### P1.64 Offline Indicator

Offline, reconnecting, stale data, queued mutation, sync success e sync conflict.

### P1.65 Loading Boundary

Loading de página, seção, card, tabela, overlay e ação. Evitar bloquear toda a aplicação para carregar uma área independente.

### P1.66 Error Boundary UI

Erro recuperável, erro inesperado, retry, report ID, detalhes técnicos expandíveis e fallback mínimo.

## Fase 7 — Componentes de fluxo e produto (parcial: implemented + scaffold)

Os itens P2.1 a P2.14 possuem implementação React independente, story própria e API prop-driven. FormBuilder, DeploymentComposer, VariableEditor, DataGrid e partes de DiffViewer permanecem scaffold até cumprirem os comportamentos avançados descritos em cada item. Os componentes não carregam regras de negócio, autorização, chamadas HTTP ou persistência; recebem dados e callbacks do produto consumidor.

### P2.1 Wizard

Etapas, progress, validação por etapa, back/next, save draft, resume, erro, revisão final, abandono e mobile.

### P2.2 Questionnaire

Perguntas condicionais, progresso, autosave, validação, branching, revisão e resultado.

### P2.3 Form Builder

Campos dinâmicos, grupos, dependências, drag/reorder, preview, schema e validação.

### P2.4 Resource Picker

Seleção de projeto, serviço, ambiente ou recurso com tree, search, recent, permissions, empty e async loading.

### P2.5 Environment Switcher

Dev, staging, production, branch/preview, protection, warning e confirmação para ações sensíveis.

### P2.6 Deployment Composer

Configuração de deploy, source, environment, variables, secrets, review, summary, progress, logs, success e rollback.

### P2.7 Variable Editor

Key/value, secret reveal, masking, validation, duplicate key, scope, bulk edit, import/export e diff.

### P2.8 Command Runner

Comando, target, permission, confirmation, output stream, cancel, retry, exit code, timeout e copy.

### P2.9 Diff Viewer

Side-by-side, unified, syntax, additions/removals, collapsed unchanged, line actions e accessible summary.

### P2.10 Approval Flow

Requester, approvers, status, policy, comment, approve, reject, expired e audit trail.

### P2.11 Bulk Action Bar

Seleção de recursos, count, actions permitted, destructive confirmation, pending progress, partial failure e clear selection.

### P2.12 Saved View

Nome, filtros, colunas, sort, owner, shared/private, favorite, rename, duplicate, delete e default view.

### P2.13 Activity Feed

Eventos agrupados, filtros, realtime, unread, pagination, loading e empty.

### P2.14 Changelog e Release Notes

Versão, categoria, impacto, migration note, links, expandable details e unread state.

## Fase 8 — Interação avançada e exploração (parcial: implemented + scaffold)

Todos os itens P3.1 a P3.10 possuem implementação React independente e story própria. ResizableDashboard, TimelineScrubber, VirtualizedList, DragAndDrop e RealtimeActivitySurface continuam scaffold até a interação avançada deixar de ser simulada ou fixa. Os componentes implementados devem manter teclado, reduced motion, RTL e estados vazios como critérios de promoção.

### P3.1 Carousel

Slides, controls, dots, autoplay opcional, pause, keyboard, swipe, reduced motion e accessible labels.

### P3.2 Aspect Ratio

Media, preview, chart, video e responsive constraints.

### P3.3 Direction Provider

LTR/RTL, espelhamento de ícones e navegação por teclado. Nenhum componente deve assumir que esquerda e direita são invariáveis.

### P3.4 Resizable Dashboard

Widgets reordenáveis, grid, resize, add/remove, save layout, reset, mobile stacking e empty dashboard.

### P3.5 Spotlight / Global Search

Busca por entidades, fuzzy matching, scopes, recent, commands, permissions, keyboard shortcuts e zero results acionável.

### P3.6 Timeline Scrubber

Janela temporal, zoom, markers, incidents, playback, range selection e timezone.

### P3.7 Virtualized List

Listas de milhares de recursos ou logs com accessibility fallback, scroll anchoring, dynamic row height e loading boundary.

### P3.8 Drag and Drop

Reorder, move, upload, keyboard alternative, preview, invalid target, cancel e reduced motion.

### P3.9 Multi-select Resource Explorer

Tree + table + preview, seleção em massa, search, filters, keyboard navigation e command actions.

### P3.10 Realtime Activity Surface

Live cursor/status, event stream, reconnect, pause, unread, jump to latest, backfill e conflict.

## Componentes shadcn/ui cobertos pelo baseline

O inventário oficial atual do shadcn/ui inclui, entre outros: Accordion, Alert, Alert Dialog, Aspect Ratio, Attachment, Avatar, Badge, Breadcrumb, Bubble, Button, Button Group, Calendar, Card, Carousel, Chart, Checkbox, Collapsible, Combobox, Command, Context Menu, Data Table, Date Picker, Dialog, Direction, Drawer, Dropdown Menu, Empty, Field, Hover Card, Input, Input Group, Input OTP, Item, Kbd, Label, Marker, Menubar, Message, Message Scroller, Native Select, Navigation Menu, Pagination, Popover, Progress, Questionnaire, Radio Group, Resizable, Scroll Area, Select, Separator, Sheet, Sidebar, Skeleton, Slider, Spinner, Switch, Table, Tabs, Textarea, Toast, Toggle, Toggle Group, Tooltip e Typography. A referência oficial deve ser consultada antes de fechar cada fase: [shadcn/ui Components](https://ui.shadcn.com/docs/components).

O Aether adiciona ao baseline: Icon Button, Link, Field extensions, Date Range Picker, Time Picker, Date Time Picker, Select Search, Resource Picker, Environment Switcher, Filter Bar, Log Viewer, Runtime Status, Deployment Composer, Variable Editor, Command Runner, Diff Viewer, Approval Flow, Bulk Action Bar, Saved View, Offline Indicator e superfícies realtime.

## Matriz de estados obrigatória

Cada componente deve avaliar quais estados se aplicam:

- Default.
- Hover.
- Focus-visible.
- Active, pressed ou expanded.
- Selected ou checked.
- Disabled.
- Readonly.
- Loading.
- Pending.
- Success.
- Warning.
- Error.
- Empty.
- No results.
- Stale.
- Offline.
- Permission denied.
- Content long.
- Overflow.
- Light mode.
- Dark mode.
- Reduced motion.
- Mobile viewport.

## Sistema de animação

### Entrada e saída

Overlays entram com opacity + deslocamento ou escala mínima. Não usar zoom agressivo em dialogs. Drawers respeitam sua direção espacial. Toasts entram da posição onde permanecerão.

### Feedback

Loading usa shimmer, pulse ou spinner conforme o contexto. Success e erro podem usar microtransição de ícone, mas devem manter texto. Estados live podem ter pulse discreto, nunca uma luz piscando sem necessidade.

### Interação

Hover é rápido e reversível. Press deve ter resposta imediata. Focus é persistente e claramente visível. Mudanças de seleção devem ser rápidas o bastante para manter sensação de controle.

### Performance

Animar `transform` e `opacity` quando possível. Evitar layout thrashing, blur exagerado em listas grandes e sombras animadas em superfícies numerosas. Componentes complexos precisam de fallback visual estático.

## Ordem de entrega recomendada

1. Tokens, Typography, layout, Icon, Button, Link, Label e Field.
2. Input, Textarea, Checkbox, Radio, Switch, Select, Combobox e Date Picker.
3. Tooltip, Popover, Dialog, Drawer, Dropdown, Command Palette e Toast.
4. Card, Item, Tabs, Accordion, Breadcrumb, Sidebar e Navigation.
5. Table, Data Table, Filter Bar, Pagination, Metrics, Chart e Timeline.
6. Log Viewer, Code Block, Diff Viewer, Resource Tree e Runtime Status.
7. Deployment Composer, Environment Switcher, Variable Editor, Approval Flow e Bulk Actions.
8. Realtime, virtualização, drag and drop e dashboard configurável.

## Definition of Done

Um componente entra no roadmap como entregue quando:

- Tem problema, consumidor e fronteira definidos.
- Possui API pública tipada e variants implementadas com `tailwind-variants` quando aplicável.
- Usa API prop-driven por padrão; qualquer Composition API precisa de justificativa de complexidade estrutural.
- Consome tokens semânticos e funciona nos dois temas.
- Tem composição com HTML nativo ou Base UI justificada.
- Tem stories para os estados relevantes e conteúdo extremo.
- Tem documentação de uso, não uso e acessibilidade.
- Não desloca layout inesperadamente em loading, erro ou conteúdo longo.
- Tem comportamento de teclado, foco e reduced motion definidos.
- Não introduz regra de negócio no componente.
- Está exportado somente depois de o contrato estar estável.

## Sequenciamento e dependências

Componentes de alto nível devem depender de primitives estabilizadas. `Command Palette` depende de Input, Dialog, List/Item, Kbd e Popover. `Combobox` depende de Field, Input, Popover e Listbox. `Date Range Picker` depende de Calendar, Popover, Button e Field. `Data Table` depende de Table, Checkbox, Button, Dropdown, Input, Select e Pagination. `Deployment Composer` depende de Field, Select, Combobox, Tabs, Alert, Progress, Dialog, Toast e Log Viewer.

Não iniciar um componente complexo apenas porque ele é visualmente interessante. Começar quando seus blocos dependentes têm contratos suficientes para que a complexidade fique no fluxo, e não em workarounds de primitives instáveis.

## Registro de decisões

Toda decisão que alterar a anatomia, o modelo de estado ou a API de uma família de componentes deve registrar:

- Problema e usuários afetados.
- Alternativas consideradas.
- Dependência de primitive externa.
- Impacto em light/dark e tokens.
- Impacto em motion e acessibilidade.
- Estratégia de migração.
- Critério para promover, manter experimental ou remover.
