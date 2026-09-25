# Elisyum Component Catalog

## Scope

This catalog identifies the component families present in `frontend/aether_ds` and records the intended product use that Elisyum must cover. The source package is a behavioral and capability reference only. Elisyum implementations must be independently designed from `DESIGN.md`, use semantic tokens, wrap Base UI where appropriate, and expose new contracts.

No source code, markup, CSS, stories, tests, snapshots, comments, or documentation wording is reused from `aether_ds`.

`NativeSelect` is intentionally excluded from the Elisyum public surface by product decision. The layered `Select` contract is the supported select API.

The implementation rule is one Storybook story file per public component or tightly bounded component family. Stories must demonstrate meaningful states, keyboard behavior where relevant, dark and light themes, and reduced-motion-safe behavior. Storybook MCP validation is required for each implementation batch.

The surface rule is mandatory across every component: in dark mode, inputs, textareas, select triggers, comboboxes, date controls, time controls, editors and other editable fields must be lighter than the opaque surface immediately behind them. A field inside a modal, drawer or sheet must remain lighter than that overlay. Use the semantic `field` and `field-hover` tokens; never substitute a darker surrounding surface token for an input background.

## Visual review strategy

The modernization pass categorizes the public surface by how much visual change it needs while preserving its behavioral contract:

| Strategy | Components |
| --- | --- |
| Keep | AspectRatio, DirectionProvider, Label, Kbd, Separator, Marker, ScrollArea |
| Refine | Button, ButtonGroup, IconButton, Link, Input, Textarea, Checkbox, RadioGroup, Switch, Toggle, ToggleGroup, Progress, ProgressRing, Spinner, Skeleton, Tooltip, Breadcrumb, Tabs, Badge, Alert |
| Redesign | Card, MetricCard, RuntimeStatus, EmptyState, Field, InputGroup, Select, Combobox, Calendar, DatePicker, DateRangePicker, DateTimePicker, Table, DataTable, Pagination, Sidebar, AppHeader, NotificationStack, Questionnaire, FormActions |
| Rebuild | LogViewer, CommandRunner, DeploymentComposer, Timeline, ResourceTree, ResizableDashboard, Wizard, Application, ApprovalFlow, RealtimeActivitySurface, DataGrid, DiffViewer |

The strategy describes visual and compositional debt, not permission to break a public contract. Every category inherits the same Elysium surface, geometry, state and motion rules from `DESIGN.md`.

## Foundations and composition

| Component | Intended use |
| --- | --- |
| ElisyumProvider | Configure theme, density, motion, transparency, contrast, locale, timezone, and token overrides for an application or isolated preview. |
| DirectionProvider | Scope left-to-right or right-to-left layout direction without changing component meaning. |
| Typography | Apply semantic display, body, label, metric, code, and log text roles. |
| AspectRatio | Reserve stable media or preview geometry. |
| Layout | Compose Box, Stack, Inline, Grid, Container, Bleed, Divider, and VisuallyHidden structures. |
| Separator | Express visual or semantic boundaries between content regions. |
| ScrollArea | Provide an intentional scroll viewport with visible affordances. |
| Skeleton | Represent pending content while preserving final layout shape. |
| Spinner | Show short-lived indeterminate work without replacing useful context. |
| Kbd | Display keyboard shortcuts and command hints. |
| Label | Name controls and preserve accessible field relationships. |
| Marker | Attach a small semantic marker to a resource, event, or status. |
| Card | Group related content into a bounded surface. |
| Item and ItemGroup | Build repeated resource rows with identity, metadata, actions, and selection. |
| Avatar | Represent a person, team, service, or fallback identity. |
| Badge | Present compact categorical or status information. |
| Button | Trigger an action with clear intent, loading, disabled, and destructive states. |
| IconButton | Trigger an icon-led action with an accessible name and compact target. |
| ButtonGroup | Group related actions while preserving hierarchy and keyboard access. |
| Link | Navigate to an internal or external destination with product-consistent emphasis. |
| CopyButton | Copy a value and provide success or failure feedback. |
| Bubble | Present a compact conversational or contextual message. |
| Accordion | Organize related sections whose details can be expanded independently. |
| Carousel | Browse a bounded set of related visual or content panels. |

## Form and input controls

| Component | Intended use |
| --- | --- |
| Field | Combine label, control, description, validation, and supporting content. |
| Input | Capture a single-line value with validation and loading states. |
| Textarea | Capture multi-line text with predictable resizing and limits. |
| InputGroup | Combine an input with leading, trailing, or adjacent affordances. |
| NumberField | Edit numeric values with keyboard, step, bounds, and formatting behavior. |
| InputOTP | Capture segmented one-time passcodes with paste and focus progression. |
| Checkbox | Select independent boolean options or a mixed parent state. |
| RadioGroup | Select exactly one option from a visible set. |
| Switch | Toggle an immediately applied setting. |
| Toggle | Toggle a single tool or view mode while preserving pressed semantics. |
| ToggleGroup | Choose one or many related modes from a compact control group. |
| Slider | Adjust a bounded scalar value or range. |
| Select | Choose one option from a layered list. |
| SelectSearch | Search and select from a large option set. |
| Combobox | Combine text entry, filtering, and option selection. |
| AsyncSearchInput | Search remote resources with loading, empty, error, and stale states. |
| Calendar | Display and select dates in a navigable calendar grid. |
| DatePicker | Select a single date with a calendar surface. |
| DateRangePicker | Select a bounded start and end date. |
| DateTimePicker | Select a date and time as one value. |
| TimePicker | Select or edit a time with locale-aware constraints. |
| FileUpload | Select files with validation, progress, retry, and removal states. |
| DragAndDrop | Move or reorder items with pointer and keyboard alternatives. |
| FormActions | Align submit, cancel, reset, and secondary form actions. |
| FormBuilder | Render typed field definitions into a coherent form surface. |
| FilterBar | Compose query filters with removable and editable filter tokens. |
| Questionnaire | Guide a user through a structured set of questions and answers. |
| ResourcePicker | Choose an application resource from a searchable collection. |
| MultiSelectResourceExplorer | Search, navigate, and select multiple related resources. |
| VariableEditor | Edit environment or service variables with value visibility and validation. |

## Navigation and overlays

| Component | Intended use |
| --- | --- |
| AppHeader | Provide the global product header, context, and high-level actions. |
| Sidebar | Provide persistent product navigation with collapsed and responsive modes. |
| OrganizationSwitcher | Change the active organization or workspace. |
| EnvironmentSwitcher | Change the active deployment environment or runtime context. |
| UserMenu | Expose account, preferences, support, and sign-out actions. |
| Breadcrumb | Show the current location and parent navigation path. |
| Tabs | Switch between related views while preserving context. |
| NavigationMenu | Expose grouped product destinations and active location. |
| Menubar | Provide a desktop command menu with grouped top-level actions. |
| DropdownMenu | Present an anchored list of actions or navigation choices. |
| ContextMenu | Present actions for the item or region under pointer or keyboard context. |
| CommandPalette | Search and execute product commands from a global surface. |
| Spotlight | Highlight a focused task, announcement, or guided product area. |
| Popover | Anchor contextual content to a trigger without modal focus trapping. |
| Tooltip | Explain an unfamiliar control without carrying essential information alone. |
| HoverCard | Preview related content after a deliberate pointer or focus interaction. |
| Dialog | Present a focused task that requires temporary attention. |
| AlertDialog | Confirm a consequential action with explicit decision semantics. |
| Modal | Provide a generic modal surface for product-specific content. |
| Drawer | Present contextual content entering from a screen edge. |
| Sheet | Present a responsive task surface that can adapt from panel to full-screen. |
| Collapsible | Reveal or hide secondary content while preserving its relationship to a trigger. |

## Feedback, status, and notifications

| Component | Intended use |
| --- | --- |
| Alert | Communicate an important persistent status or next action. |
| Banner | Communicate broad product or workspace status across a page region. |
| InlineError | Place concise validation or recovery guidance beside a control. |
| Message | Render a single user, system, or operational message. |
| MessageScroller | Browse a bounded stream of messages while preserving the latest context. |
| LoadingBoundary | Coordinate pending content with a useful loading fallback. |
| ErrorBoundaryUI | Give a failed render a recoverable and understandable presentation. |
| OfflineIndicator | Surface disconnected, reconnecting, and recovered network state. |
| RuntimeStatus | Show the lifecycle and health of an application or service. |
| Progress | Show determinate or indeterminate progress in a linear region. |
| ProgressRing | Show progress in a compact circular region. |
| Gauge | Show a bounded metric against a target or threshold. |
| EmptyState | Explain the absence of content and expose a useful next action. |
| ToastProvider and Toast | Provide transient, accessible feedback with action and promise states. |
| Sonner | Mount the single notification channel used by the design system. |
| NotificationStack | Present a persistent or grouped set of notifications. |

## Data, observability, and technical surfaces

| Component | Intended use |
| --- | --- |
| Table | Present structured rows and columns with accessible headers. |
| DataTable | Add sorting, selection, pagination, and empty or loading states to tabular data. |
| DataGrid | Support dense interactive tabular data with keyboard navigation and column behavior. |
| VirtualizedList | Render long collections while preserving item identity and scroll performance. |
| Pagination | Navigate through a bounded result set. |
| SortControl | Express an active column sort and its direction. |
| SavedView | Save, select, and manage a reusable query or table configuration. |
| AuditLog | Present an ordered record of actor, action, target, and outcome. |
| ActivityFeed | Present chronological product activity with identity and context. |
| RealtimeActivitySurface | Display live operational events, connection state, and replay boundaries. |
| Timeline | Present ordered lifecycle events and their status. |
| TimelineScrubber | Navigate through a time-based event or media sequence. |
| Chart | Visualize time series or categorical metrics with accessible summaries. |
| ChartTooltipLegend | Explain chart series and focused values. |
| MetricCard | Present a headline metric, comparison, trend, and status context. |
| LogViewer | Browse technical logs with follow, filtering, timestamps, and copy actions. |
| DiffViewer | Compare two text or configuration revisions with line-level context. |
| CodeBlock | Present readable, copyable, syntax-aware technical text. |
| CodeEditorLite | Edit small code or configuration values with useful keyboard behavior. |
| ResourceTree | Navigate hierarchical infrastructure or project resources. |
| Resizable | Resize adjacent panels while preserving usable minimum dimensions. |
| ResizableDashboard | Arrange and resize dashboard panels within a stable responsive grid. |
| Attachment | Show an attached file or artifact with status and actions. |
| Changelog | Present product releases and noteworthy changes. |

## Operational workflows and product patterns

| Component | Intended use |
| --- | --- |
| ApprovalFlow | Represent requested, reviewed, approved, rejected, and delegated decisions. |
| DeploymentComposer | Configure and review a deployment before execution. |
| CommandRunner | Enter, execute, and inspect an operational command with safe feedback. |
| BulkActionBar | Expose actions for the current selection without hiding selection context. |
| SavedView | Persist a meaningful operational view configuration for later reuse. |
| Wizard | Guide a multi-step task with validation, progress, and recoverable navigation. |
| Application | Demonstrate a composed application surface and its product-level layout responsibilities. |
| Forms | Demonstrate composed form patterns and field relationships. |
| Foundation | Demonstrate tokens, type roles, and foundational states. |

## Implementation phases

1. Foundations, providers, layout, typography, surfaces, buttons, links, and status primitives.
2. Form controls and field composition, including keyboard-intensive Base UI adapters.
3. Navigation and overlays, including focus management, dismissal, and responsive surfaces.
4. Data, observability, logs, charts, and virtualization contracts.
5. Operational composites and workflow patterns.
6. Cross-component verification for themes, density, reduced motion, accessibility, and production packaging.

Each phase must update the public exports, include a Storybook story for every implemented public component, run the Storybook MCP story instructions and preview checks, and pass typecheck, package build, and Storybook build.
