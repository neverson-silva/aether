# Elysium DS

Elysium is Aether's operational interface language: a precise, calm command surface for deploying, observing, repairing and protecting live systems. It is not a theme, a dashboard kit or a generic SaaS component library.

The system exists to shorten the distance between system state and the next safe operation. Every screen and component should make four things legible:

1. What resource is in scope.
2. What state it is in now.
3. What the operator can do next.
4. What consequence that action has.

## Diagnosis and direction

The previous visual language was technically coherent but too close to the familiar 2015–2022 SaaS grammar: repeated rounded rectangles, equal-weight borders, floating cards for unrelated facts, badge-heavy status, generic blue actions and conventional dashboard spacing. The modernization removes container-first hierarchy and replaces it with spatial, typographic and operational hierarchy.

Elysium should feel precise, engineered, calm, high-performance, technical, confident and refined. It must remain humane during failure, dense enough for expert work and quiet enough for long sessions.

The product references are principles only: Apple contributes directness, spatial continuity and restrained motion; Google Cloud contributes resource scope and hierarchy; AWS contributes operational state coverage; Azure contributes enterprise clarity; Nubank contributes confident contrast and human error language. Elysium copies none of their visual systems, colors or component anatomy.

## Elysium Visual DNA

These are the structural characteristics that should survive removal of the Aether logo:

1. **Context rails** — scope, resource identity and current location are expressed as an aligned rail before content begins. The operator always knows organization, project, environment and resource.
2. **Layered workplanes** — the canvas, work surface, contextual region, field, overlay and critical overlay use distinct tonal roles. A surface earns elevation by changing spatial ownership, not by receiving a shadow.
3. **Signal edges** — selection, focus, health and lifecycle use restrained edge treatments, markers and text. Active context is not a large saturated fill; it is an exact signal attached to the thing it changes.
4. **Technical metadata rhythm** — IDs, timestamps, versions, ports, commit hashes and resource quantities use aligned technical type and stable tabular spacing. Metadata is architecture, not gray filler.
5. **Operational state language** — lifecycle and health are separate dimensions. A deployment can be running while health is degraded. State uses marker, label, phase, progress and next action as appropriate.
6. **Composed density** — information is grouped by operational question into continuous workplanes, split regions, timelines and inspectors. Cards are used for independent objects, not as a default wrapper.
7. **Causal motion** — press, selection, disclosure, overlay, progress and completion respond at the source of the action. Motion is short, interruptible where physical, and always useful without animation.

## Composition rules

Elysium is an application environment, not a collection of widgets.

- Use alignment, whitespace, typography and separators before adding a container.
- Use a card only when the content represents an independent object, action group or decision surface.
- Use a continuous workplane for related metrics, settings and technical data.
- Use a split region when two views share a task: resource list + detail, editor + preview, timeline + inspector.
- Use an inspector when detail must preserve the user's current workspace.
- Use a modal only for focused decisions, creation flows that must stay in context, or genuinely blocking confirmation.
- Use a drawer or sheet when the parent context should remain visible.
- Use a popover or menu for short contextual choices.

The default product frame is:

1. Context rail: organization, project, environment and breadcrumb.
2. Resource header: identity, type, lifecycle, health, metadata and the primary safe action.
3. View navigation: tabs for stable views of the same resource.
4. Workplane: table, timeline, editor, stream, form or command surface.
5. Inspector: only when it preserves context and reduces navigation.

Responsive layouts change this spatial model. Side inspectors become sheets, rails become compact scope strips, action groups become menus and tables preserve their primary columns instead of shrinking every field.

## Surface architecture

Surfaces are spatial ownership, not decoration.

| Layer | Token role | Meaning |
| --- | --- | --- |
| L0 | `canvas` | application field and global negative space |
| L1 | `surface-0` | structural chrome, navigation and stable page regions |
| L2 | `surface-1` | primary workplane in normal flow |
| L3 | `surface-2` | contextual region, table head, hover and nested grouping |
| L4 | `field` | interactive control surface, always distinct from its parent |
| L5 | `surface-3` / `overlay` | menus, popovers, drawers and dialogs |
| L6 | `surface-4` / `critical-overlay` | stacked destructive or blocking surfaces |

The dark theme uses a deliberate lightness ladder. Inputs, textareas, select triggers, comboboxes and date/time controls are always slightly lighter than the opaque surface immediately behind them. A field inside an overlay remains lighter than that overlay. Light mode preserves the same relationship through value and edge contrast.

Do not use shadows to turn normal content into paper. Borders are for relationships: quiet separators, structural boundaries, interactive boundaries and focus boundaries are different roles. Elevation is reserved for transient or actively lifted surfaces.

Translucency belongs to structural chrome and temporary surfaces. Workplanes, code, logs, tables and forms remain opaque enough for long-session readability. Reduced transparency removes blur and increases opacity.

## Color language

Color carries operational meaning and remains scarce. The action accent is an indigo-ultramarine signal, not a page decoration and not teal. Information blue, success green, warning amber, danger coral and neutral state are independent channels.

| Role | Use |
| --- | --- |
| `action` | primary operation, selection edge, progress and focus relationship |
| `info` | factual information, links and discovery |
| `success` | healthy, ready, completed and verified |
| `warning` | degraded, stale, attention and partial availability |
| `danger` | invalid, failed, destructive and blocked |
| `neutral-state` | unknown, paused, unavailable and not configured |

Rules:

- Never use action color to decorate every heading or surface.
- Never encode state by color alone; pair it with text, iconography, position, shape or progress.
- Never use raw hex values in component files. Add semantic roles to `tokens.css` and `tokens.ts`.
- Avoid gradients on controls and surfaces. Data visualization may use a gradient only when it represents measured data.
- Do not introduce teal as a primary, brand or fallback accent.

## Geometry and density

Geometry is relational rather than uniformly rounded.

- `radius-sm`: compact menu rows, dense technical controls and table affordances.
- `radius-md`: fields, buttons and ordinary interactive controls.
- `radius-lg`: independent cards, resource groups and workplane boundaries.
- `radius-xl`: focused workflow surfaces and overlays only.
- `radius-pill`: status tracks, compact tags and progress tracks only.

Nested surfaces should use a smaller radius than their parent. Rows and tables should not become collections of pills. A full-bleed log, terminal or editor uses square internal geometry.

The base spacing unit is `0.25rem`. Use `2` for icon-to-label, `3` for control internals, `4` for ordinary component padding, `5–6` for related sections, `8` for page-level separation and `10–16` only for onboarding or deliberate empty states. Density is chosen by task: `dense` for inventory, `stream` for logs, `telemetry` for metrics and `focused` for decisions.

Default controls are `2.5rem`, dense controls `2.25rem`, focused controls `2.75rem`. Actionable touch targets remain at least `2.75rem` on touch-oriented layouts.

## Typography

Typography carries architecture. Use platform-shaped interface type (`-apple-system`, `BlinkMacSystemFont`, `Inter`, `Segoe UI`) and technical type (`ui-monospace`, `SFMono-Regular`, `IBM Plex Mono`, `Menlo`) for identifiers and streams.

| Role | Size | Leading | Weight | Tracking |
| --- | --- | --- | --- | --- |
| Display | `3rem` | `1.04` | `650` | `-0.035em` |
| Page title | `2rem` | `1.1` | `650` | `-0.028em` |
| Section title | `1.125rem` | `1.25` | `650` | `-0.012em` |
| Body | `1rem` | `1.5` | `400` | `0` |
| Supporting | `0.875rem` | `1.4` | `400` | `0.005em` |
| Label | `0.75rem` | `1.2` | `650` | `0.035em` |
| Metric | `2.25rem` | `1` | `650` | `-0.035em` |
| Code | `0.875rem` | `1.5` | `450` | `0` |
| Log | `0.8125rem` | `1.5` | `450` | `0.01em` |

Use labels as orientation, not decoration. Technical values align numerically and remain selectable. Long resource names truncate only when the full value remains available through accessible naming or a deliberate disclosure.

## Controls

Controls belong to one family: the same field layer, edge treatment, focus language, icon alignment, press response and density rules apply to Button, IconButton, Input, Textarea, Select, Combobox, Checkbox, Radio, Switch, Tabs, Segmented Control, Search, Command and menus.

- Primary action: one visually dominant action per decision surface.
- Secondary action: quieter neutral surface or outline relationship.
- Quiet action: text/icon action for low-risk contextual operations.
- Dangerous action: coral semantics and deliberate confirmation when irreversible.
- Toolbar action: compact, icon-led and grouped by proximity to the thing it changes.
- Inline action: low ceremony and aligned to the affected value.

Buttons are not all equal rectangles. Use contrast, placement and label specificity to establish hierarchy. Use `Deploy`, `Restart`, `Attach domain`, `Rotate secret` and `Create service` instead of generic `Submit` or `Manage`.

## State and interaction

Every interactive component supports, where relevant: default, hover, pressed, focus-visible, selected, disabled, loading, invalid, success, warning and error.

- Hover communicates availability and proximity, not selection.
- Press responds on pointer down and uses a restrained `0.985` scale or color shift.
- Selected state uses signal edge, marker, text and surface relationship; it is not hover.
- Focus is a high-contrast keyboard boundary and never relies on color alone.
- Disabled controls explain unavailability through state and affordance, not opacity alone.
- Interactive hit areas expose `cursor: pointer`; disabled and non-interactive descendants do not.
- Copy, save, deploy, retry and completion actions provide immediate local feedback and durable status.

Motion tokens: press `100ms`, fast `140ms`, standard `180ms`, popover `220ms`, overlay `280ms`; default easing `cubic-bezier(0.22, 1, 0.36, 1)`. Use critically damped motion by default. Use spring-like behavior only for physical drag, sheet or scrubber interactions. Reduced motion replaces travel with opacity and color changes, never removes meaning.

State transitions are a signature: queued → building → deploying → running, connected → reconnecting → connected, healthy → degraded → unhealthy. Show phase, progress, marker morphology and next action without making anything blink continuously.

## Operational patterns

### Resource identity

Resource headers combine identity, kind, environment, lifecycle, health, technical metadata and primary actions without becoming a hero banner. The identity pattern is compact enough for service, database, environment, deployment and domain contexts.

### Deployment

Deployment is temporal: commit, trigger, queue, build, artifact, deploy, runtime and health. Prefer timeline and phase surfaces over generic status tables. Preserve logs, cancellation, retry and rollback context.

### Logs and terminal

Logs and terminal are hero experiences for a PaaS. They use dense, selectable technical type, stable row rhythm, follow/disconnected states, copy affordances, clear command focus and restrained chrome. They are not black rounded boxes with random green text.

### Inspector

Use an inspector to examine deployments, containers, variables, domains, events and health without losing the current workspace. On small screens it becomes a sheet with the same information hierarchy.

### Empty and failure states

An empty environment, no search results, no deployments, no logs, no metrics, permission denied and connection lost are different states. Each names the context, explains the condition and exposes the next useful action when one exists. Do not use a generic oversized icon and a generic CTA.

### Wizard

Creation workflows for services, projects, databases and environments use a progress rail, explicit step descriptions, saved summaries, inline validation, review before commit and a final action named after the object being created. Allow returning to completed steps. Keep future steps quiet and unavailable until prerequisites are met. Never make `Continue` the final destructive or creating action.

## Accessibility

Every interactive surface is keyboard usable, has a visible focus-visible state, uses correct semantics and restores focus after blocking overlays. Labels, descriptions and errors are explicitly associated. Live regions announce meaningful async changes without stealing focus. Reduced motion, reduced transparency and high contrast alter presentation while preserving meaning.

Status is never color-only. Touch targets remain usable. Technical text remains selectable. Tables, logs and long resource names degrade predictably at narrow widths.

## Keep, refine, redesign, rebuild

| Strategy | Scope |
| --- | --- |
| Keep | behaviorally correct low-visual primitives: separator, label, kbd, aspect ratio, direction provider |
| Refine | button, icon button, input, textarea, checkbox, radio, switch, progress, tooltip, tabs, badge, alert and basic navigation |
| Redesign | cards, metric cards, runtime status, empty states, resource headers, pagination, tables, forms and workflow navigation |
| Rebuild | logs, terminal/command surfaces, deployment timeline, inspector patterns, resource tree, wizards and dense data workplanes |

All components inherit the same grammar. Do not independently invent a new visual treatment for each component.

## Anti-patterns and regression rules

- No default card grid for unrelated operational facts.
- No border on every wrapper; each border must communicate a relationship.
- No uniform radius applied to every component.
- No giant page title followed by a wall of equal cards.
- No badge for every state; choose the appropriate status representation.
- No generic blue button repeated across a screen.
- No raw black log surface, neon developer aesthetic or decorative infrastructure imagery.
- No gradients, glass or glow as a substitute for hierarchy.
- No native date/calendar UI as the primary visual experience.
- No native option menu as the primary Select or Combobox experience.
- No interaction whose meaning depends on animation.
- No hardcoded colors in components and no teal as a primary signal.

## Implementation and validation

Elisyum is a clean-room implementation. `frontend/aether_ds` is a capability reference only; no source, markup, CSS recipe, story, fixture or documentation wording may be copied.

Use semantic tokens from `src/foundations`, wrap Base UI behavior behind Elisyum APIs, and keep public exports stable. Every public component has a Storybook story with meaningful states. For every UI change, query `ds_mcp`, query Storybook docs and story instructions, update stories, call `stories_changed`, call `stories_preview`, then run typecheck, production build, Storybook build and available accessibility checks.

The final review asks three questions:

1. Remove the logo: does the composition still look like Aether?
2. Apply the 2018 test: is the hierarchy still a card grid, generic sidebar or uniform rounded SaaS surface?
3. Apply the Dribbble test: does it remain useful after eight hours, hundreds of resources, long logs, failures, keyboard use and a small laptop?
