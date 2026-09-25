# AGENTS.md — Elisyum Design System

## Scope

This directory is the home of the Elisyum Design System. Elisyum is a complete, independent reimplementation of the capabilities represented by `frontend/aether_ds`, with additional foundations and product patterns required by the Aether platform.

The system is dark-mode first and must support light mode as a first-class theme. It is intended for production interfaces, not a small component exercise or a thin visual wrapper around an existing library.

This document applies to every file under `frontend/elisyum_ds`.

## Non-negotiable directive

Rebuild every required component and foundation in `frontend/elisyum_ds` from first principles using Tailwind CSS and Base UI.

The target is capability parity and a stronger product language, not source parity. Elisyum may preserve the same user-facing intent, accessibility expectations, state coverage, provider responsibilities and operational workflows as Aether DS, while using new APIs, new composition, new markup, new styling, new tokens and new implementation decisions.

No line of code may be copied from `frontend/aether_ds` or from another Aether Design System implementation.

## Relationship with Aether DS

`frontend/aether_ds` is a behavioral and capability reference only. It can be inspected to understand:

- Which component families the product needs.
- Which interaction states and accessibility contracts must exist.
- Which provider responsibilities are expected by consumers.
- Which operational workflows need reusable patterns.
- Which gaps Elisyum should solve or extend.

It is not an implementation dependency, visual source, token source or file template.

Elisyum must not:

- Import from `frontend/aether_ds` or `@aether/design-system`.
- Re-export Aether DS components through a new package.
- Symlink, alias or wrap Aether DS to avoid implementing a component.
- Copy and rename a component, hook, provider, test, story, fixture or utility.
- Copy CSS, Tailwind class compositions, token values, markup hierarchy or snapshots.
- Copy comments, documentation wording, examples or test descriptions.
- Mechanically port the old directory structure and change names afterward.

When a capability is shared, write an independent contract first, then implement it independently. “Same idea” means equivalent product intent and reliable behavior; it does not mean identical source, structure or visual output.

## Product and design authority

Use the repository-level `DESIGN.md` as the primary visual and interaction constitution for Elisyum. It defines the product identity, dark-first theme, light-mode requirements, typography, tokens, surfaces, motion, accessibility, operational states and component expectations.

Use `FRONTEND_PRODUCT_MAP.md` to understand the Aether product areas, navigation, routes, services and workflows that the design system must support.

Use `frontend/aether_ds` only to discover capability requirements and compatibility risks. If its behavior conflicts with `DESIGN.md`, the Elisyum direction in `DESIGN.md` wins unless an explicit product contract requires compatibility.

## Technology foundation

### Tailwind CSS

Tailwind is the composition and implementation layer for layout, responsive behavior, spacing, typography, state variants and component styling.

Elisyum owns the semantic design tokens. Components must consume semantic CSS variables and Tailwind mappings instead of scattering raw colors, arbitrary shadows, one-off radii or unrelated pixel values throughout component files.

Dark surface layering is a hard invariant: editable controls must always use the semantic `field` surface, which is lighter than the opaque surface behind them. This remains true inside normal work surfaces, dialogs, drawers, sheets, popovers and other overlays. Use `field-hover` for interaction feedback; never use a darker surface token for inputs or textareas.

The Tailwind setup must:

- Expose Elisyum semantic tokens as utilities.
- Support dark mode as the default runtime mode.
- Support light mode without changing component anatomy or API.
- Support focus-visible, disabled, selected, invalid, loading and data-state variants.
- Preserve responsive behavior and density without duplicating components per breakpoint.
- Keep component styles local and composable rather than creating global selectors for individual screens.

### Elysium modernization rules

- Build hierarchy from context rails, typography, alignment and separators before adding cards.
- A card represents an independent object or decision surface; do not wrap every metric, form section or status in a card.
- Use the L0–L6 surface ladder in `DESIGN.md`; normal content is not elevated paper.
- Inputs, textareas, selects, comboboxes and date controls must remain visibly lighter than their surrounding surface in dark mode.
- Use signal edges, markers and operational text for selection and status. Do not use badges or saturated fills as the default representation of every state.
- Borders must communicate a relationship. Do not add a border merely because a component is interactive.
- Use the Elysium geometry relationships; do not apply one radius or pill treatment indiscriminately.
- Technical identifiers, quantities, timestamps, ports and logs use technical typography and stable alignment.
- The default accent is reserved for action and selection. Teal, decorative gradients, glow and neon developer styling are forbidden.
- Every interactive hit area must expose the correct cursor, immediate pressed feedback, visible focus and a reduced-motion path.

Tailwind does not define the product identity by itself. The design system owns the token names, semantic roles and component contracts that Tailwind consumes.

### Base UI

Use Base UI primitives for accessible behavior where a headless primitive is appropriate, including dialogs, popovers, menus, selects, comboboxes, tabs, tooltips, navigation surfaces and other keyboard-intensive interactions.

Base UI provides behavior and accessibility infrastructure. Elisyum owns:

- The public component API.
- The visual language and semantic tokens.
- Composition boundaries.
- Product-specific states.
- Focus and dismissal policy.
- Motion, layering and responsive behavior.
- Documentation and test coverage.

Do not expose raw Base UI primitives as the design system API when an Elisyum component contract is needed. Wrap them with deliberate names and stable composition. Do not reproduce Base UI internals.

### Storybook MCP

When working on UI components, always use the `elisyum-storybook-mcp` MCP tools whenever they are available. Start Storybook with `npm run storybook` when a running server is needed. The project MCP endpoint is `http://localhost:6006/mcp`.

Before answering or taking action on a component API:

- Query `docs-list` to find documented components.
- Query `docs-show` for the component before using its props, states or examples.
- Query `docs-show-story` when `docs-show` does not provide enough detail.
- Never invent component properties, including common-sounding properties such as `shadow`.
- Use only properties explicitly documented or shown in stories.
- Query `get-storybook-story-instructions` before creating or updating stories.
- Use `test-run` to validate Storybook stories and interaction or accessibility behavior whenever the testing toolset is available.

If the MCP server is unavailable, state that limitation and use the local Storybook, typecheck, build and test commands as the fallback validation path.

### Sonner

The notification system uses Sonner. The official package name is `Sonner`; all documentation and APIs in this directory must use that spelling.

Elisyum must provide one coherent toast experience with the same essential product capability expected by current consumers:

- Success, information, warning and error feedback.
- Loading and promise-based operations.
- Updating, dismissing and replacing notifications.
- Action affordances when a recovery or follow-up action exists.
- Dark and light theme synchronization.
- Accessible live-region behavior and readable durations.
- Consistent placement, layering and z-index above dialogs and sheets.
- Reduced-motion behavior.

Mount one provider-level Toaster. Avoid per-page toaster instances and duplicated notification channels.

## Provider architecture

Providers are part of the design system contract and must be reimplemented independently. They must provide the same necessary capabilities without copying the old provider code.

### ElisyumConfigProvider

Create a typed configuration provider for product-wide design-system configuration. It should own defaults and scoped overrides for concerns such as:

- Theme mode and system-theme resolution.
- Density and component scale.
- Motion and transparency preferences.
- Accessibility preferences that affect presentation.
- Sonner defaults such as position, duration and close behavior.
- Locale, timezone and formatting defaults when components need them.
- Brand and icon configuration where consumers need controlled variation.

Configuration must be explicit, type-safe and overridable without requiring consumers to fork components. The provider must not become a repository for server state, authentication state, deployment state or arbitrary business data.

### Theme provider

The theme provider must make dark mode the default, expose light mode, and support system preference where product configuration enables it. It must apply the theme through a stable root attribute or class and keep token semantics consistent across themes.

Theme changes must not cause component APIs or DOM meaning to change. The same status must remain understandable through text, iconography, shape, contrast and semantics in both themes.

### Sonner provider

Compose Sonner once with Elisyum configuration and theme state. The provider must be usable from application roots, stories, tests and isolated component previews without requiring application-specific runtime state.

The provider contract must support a single source of truth for toast configuration. Components should request notifications through the supported Elisyum notification surface rather than creating ad hoc toaster instances.

### Provider composition

The public provider composition must be predictable for consumers. The intended root composition is conceptually:

1. Configuration defaults and overrides.
2. Theme resolution and token application.
3. Overlay and portal behavior.
4. Sonner configuration and Toaster mounting.

The exact component names and implementation are up to Elisyum. They must be newly designed rather than copied from Aether DS.

## Architecture

Organize the system around explicit layers:

- `src/foundations`: semantic tokens, themes, Tailwind integration, typography, motion, elevation, icon policy and foundational CSS.
- `src/providers`: configuration, theme, overlays, notifications and other system-level context.
- `src/primitives`: independently implemented low-level controls and Base UI adapters.
- `src/components`: stable reusable components with focused contracts.
- `src/composites`: composed controls such as search fields, filter bars, data toolbars and resource summaries.
- `src/patterns`: reusable page and workflow patterns for operational product surfaces.
- `src/hooks`: public hooks only when behavior is genuinely reusable and has a clear contract.
- `src/icons`: icon policy and any Elisyum-specific icon wrappers.
- `src/testing`: test helpers and accessibility fixtures that belong to Elisyum itself.
- `src/index.ts`: the supported public API.

Keep dependencies flowing toward foundations and behavior primitives. Avoid circular imports, hidden global state and deep imports into internal implementation files.

Prefer small, composable APIs. A component with many boolean props, incompatible slots or screen-specific exceptions should be split into explicit compositions.

## Component scope

Elisyum must cover the full component surface required by the Aether product, then extend it where the product needs more expressive operational interfaces.

### Foundations

Implement and document:

- Color and semantic theme tokens.
- Typography roles and technical text styles.
- Spacing, sizing, radius and elevation scales.
- Focus rings, selection indicators and validation semantics.
- Motion durations, easing and reduced-motion behavior.
- Surface, overlay and portal layering.
- Icon sizing, alignment and accessibility rules.
- Responsive and density rules.

### Core primitives

Provide deliberate Elisyum contracts for the usual interface primitives, including:

- Button and icon button.
- Link and navigation link.
- Input, textarea and field.
- Checkbox, radio and switch.
- Select and combobox.
- Menu, context menu and command menu.
- Dialog, alert dialog, drawer and sheet.
- Popover and tooltip.
- Tabs and segmented controls.
- Badge, chip, avatar and status indicator.
- Card, panel, separator and scroll area.
- Table, pagination and data controls.
- Progress, spinner and skeleton.
- Alert, callout, empty state and error state.
- Sonner-backed toast and notification surfaces.

### Operational composites

Support the Aether mental model with higher-level building blocks such as:

- Resource header with identity, status, metadata and primary actions.
- Lifecycle and health state pair.
- Deployment timeline and deployment log viewer.
- Service variable editor and secret-value presentation.
- Domain and certificate status editor.
- Metrics panel and telemetry summary.
- Terminal workspace and command surface.
- Key-value editor for environment and configuration data.
- Confirmation flow for destructive operations.
- Async operation status with queued, running, succeeded, failed and cancelled states.
- Split panes, resizable panels and contextual inspectors.
- Filter bar, query controls and command palette.

### Elisyum extensions

Elisyum should go beyond parity where it improves the product:

- Make async and realtime state visually coherent across deployments, backups, restores and domains.
- Make failure recovery and next actions explicit instead of exposing only an error label.
- Support dense technical views without losing hierarchy or accessibility.
- Provide graceful states for disconnected, stale, unknown, partially available and permission-limited resources.
- Provide reusable patterns for infrastructure relationships, logs, events, credentials and resource dependencies.
- Add composable inspector and command surfaces that preserve context during high-impact operations.

The extension must still respect the product model. More components are not automatically better; add a pattern when it reduces repeated decisions and has a stable boundary.

## Component contracts

Every public component must define, through types and tests:

- Its semantic role.
- Its supported composition model.
- Its keyboard and pointer behavior.
- Its focus and dismissal behavior.
- Its controlled and uncontrolled state model where applicable.
- Its loading, error, empty and disabled behavior.
- Its accessible name and description requirements.
- Its theme and density behavior.
- Its responsive behavior.
- Its motion and reduced-motion behavior.
- Its escape hatches, if any, and why they are safe.

Do not hide product state inside visual variants. Prefer explicit state names such as `status`, `phase`, `intent`, `loading`, `disabled`, `selected` or `invalid` over ambiguous combinations of styling flags.

Use semantic component names and stable public exports. Internal implementation names may change during development without becoming consumer contracts.

## State completeness

No component is complete if it only demonstrates its resting state. When relevant, cover:

- Default.
- Hover.
- Pressed.
- Focus-visible.
- Selected.
- Disabled.
- Loading.
- Empty.
- Invalid.
- Success.
- Warning.
- Error.
- Queued.
- Deploying.
- Degraded.
- Disconnected.
- Stale.
- Unknown.

Operational states must use more than color. Pair color with text, icons, labels, shape, position or motion so that the meaning survives color-vision differences, reduced motion and light-mode changes.

## Dark-first and light-mode support

Dark mode is the default Elisyum experience. It must be designed intentionally, not produced by inverting a light theme.

In dark mode, fields are intentionally lighter than their container. A modal or sheet may be darker than its inputs, and a normal work surface may be darker than its controls. The relationship is semantic and must remain intact when themes, density or scoped token overrides change.

Light mode is supported and must be treated as a complete product surface:

- Every semantic token requires a considered light value.
- Text, borders, focus rings and controls must remain legible.
- Elevation must remain visible without relying on dark shadows.
- Disabled and secondary states must remain distinguishable.
- Overlays, dialogs, menus and toasts must preserve contrast.
- Charts, status indicators and technical data must remain interpretable.

Do not create separate component families such as `DarkButton` and `LightButton`. Themes change semantic values, not component meaning or anatomy.

## Accessibility

Accessibility is part of the implementation contract, not a later audit. Every interactive component must be usable with keyboard and pointer input and must expose correct semantics to assistive technology.

Required checks include:

- Keyboard navigation and activation.
- Visible focus-visible treatment.
- Correct labels, descriptions and error association.
- Correct dialog, menu, select, combobox and tab semantics.
- Focus trapping and focus restoration for modal surfaces.
- Escape and outside-interaction behavior.
- Live-region behavior for async feedback and notifications.
- Sufficient contrast in dark and light themes.
- Usable target sizes and touch behavior.
- Reduced-motion behavior.
- No information conveyed only by color, hover or animation.

Use Base UI behavior where it improves correctness, but verify the resulting Elisyum composition instead of assuming a primitive solves every product requirement.

## Motion and interaction quality

Use motion to communicate causality, hierarchy and spatial relationships. Prefer restrained, interruptible transitions over decorative effects. Menus and popovers should preserve their relationship with their trigger. Drawers and sheets should enter from their anchored edge. Async operations should expose progress or phase changes without making the user wait for animation.

Every transition must have a reduced-motion path. Never make loading, success or failure understandable only through animation.

## Dependency and reuse policy

Allowed reuse:

- Public third-party packages installed as declared dependencies.
- Base UI primitives used according to their license and public API.
- Tailwind CSS and its official integration mechanisms.
- Sonner through its public API.
- Product requirements and documented behavior discovered from the repository.

Forbidden reuse:

- Source files from `frontend/aether_ds`.
- Internal or generated output from Aether DS.
- Copied component structure with renamed symbols.
- Copied CSS or Tailwind recipes.
- Copied tests, stories, snapshots or fixtures.
- Re-exporting the old package under Elisyum names.

If a design or implementation decision is inspired by an existing system, express it as a new Elisyum contract and implement it independently. Third-party license obligations remain mandatory.

## Verification

Before considering a component or provider complete:

1. Verify there is no import, alias, symlink or re-export from Aether DS.
2. Verify the implementation uses Elisyum semantic tokens and Tailwind integration.
3. Verify Base UI behavior is wrapped by an Elisyum API where applicable.
4. Verify provider composition works in the application root, isolated stories and tests.
5. Verify Sonner is mounted once and follows theme, layering and accessibility rules.
6. Verify dark and light themes.
7. Verify keyboard, focus, screen-reader semantics and reduced motion.
8. Verify all relevant state variants, including operational failure states.
9. Run the package typecheck, tests and production build.
10. Review the public API for unnecessary flags, accidental business logic and deep-import leakage.

When visual regression tooling is available, cover both themes and the important responsive states. When accessibility tooling is available, run it against every public interactive pattern and not only the happy path.

## Completion checklist

A contribution to Elisyum is ready only when:

- The component or provider has a clear purpose and boundary.
- The implementation is original and contains no copied source lines from Aether DS.
- The public contract is typed and intentionally named.
- Tailwind and semantic tokens are used consistently.
- Base UI is used for behavior where appropriate without leaking raw internals.
- Configuration, theme and Sonner behavior remain coherent.
- Dark mode is the default and light mode is complete.
- States, failures and async transitions are represented.
- Accessibility and reduced motion are verified.
- The component works outside one screen or one business-flow implementation.
- Tests and production build pass.
- No generated artifact, compatibility shortcut or old-package dependency is being used to bypass the clean-room implementation.
