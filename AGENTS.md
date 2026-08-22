# AGENTS.md — Aether Working Rules

## English First — NEVER Portuguese in Code/UI

This app (frontend and backend) is **english-first**:

- All code, logs, error messages, comments, variable/function/route names,
  event names (e.g.: `deploy.queued`, notifications), labels, placeholders,
  tooltips, toast messages and any user-facing text must be in **English**.
- **FORBIDDEN** to contain Portuguese words in the source code, UI or
  app-generated text, **unless the user explicitly asks for it**.
- If writing code produces Portuguese text (e.g.: error message, button label,
  modal title), rewrite it to English before finishing.

## FORBIDDEN to Comment Code (Go and Web) — Exception: Docblocks

This is one of the most severe rules in the project. **Comments in code are
not welcome.** They are not "free documentation": they are accumulated debt.

### The Rule

**NO comment is allowed in Go or web source code (TS/TSX/JS/CSS/SCSS),
unless explicitly requested by the user.** This includes, without exception:

- Line comments (`//`, `/* */`, `#` in CSS/SCSS where supported).
- Comments that "explain" what the code does.
- "Why" comments (even if they seem useful).
- TODO/FIXME/HACK/XXX comments.
- Authorship, date, or "last change" comments.
- Divider section comments (e.g.: `// ---- LIFECYCLE ----`).
- Example or "how to use" comments.
- Comments copied from another codebase.
- Comments that repeat the function/variable name.
- Code comments embedded in commit messages as if they were inline comments.

**Code MUST be self-explanatory**: clear variable, function, type, package and
route names, obvious structure, nothing else. If a snippet needs a comment to
be understood, rewrite it to be readable — do not comment it.

### Why

- Comments lie: code changes, the comment stays and becomes misinformation.
- Comments duplicate: 90% repeat what the code already says.
- Comments cost: each commented line is two maintenance surfaces.
- Comments wrap: they hide bad code behind an explanation.
- Comments hurt diffs, code review and refactors.
- The function name, the types and the tests already document behavior.

### The Only Exception: Docblocks

At MOST, **docblocks** are allowed — and only when they document the **public
contract** of an exported symbol (what it is, what it expects, what it
returns), never **how** it works:

**Go — docblocks allowed ONLY for:**
- Exported symbols (types, functions, methods, constants, package variables).
- The package clause (one sentence: what the package is).
- Start with the symbol name (`// Stats returns...`), official Go convention.
- Max ~5 lines. One sentence is ideal.
- **FORBIDDEN** docblock on: unexported symbols, private functions, struct
  fields, local variables, code blocks, internal logic.

**Web (TS/TSX) — docblocks allowed ONLY as:**
- JSDoc (`/** ... */`) on exported components and exported hooks, describing
  the contract (props, return) — not the implementation.
- Max ~5 lines.
- **FORBIDDEN** on: unexported functions, inline callbacks, state, effects,
  styles, inline JSX, internal component logic.

**CSS/SCSS — NO comment at all, not even docblocks.** Descriptive class and
token names are enough.

**FORBIDDEN in any language:**
- Docblock on an unexported symbol.
- Docblock that explains "how" (internal logic, algorithm, iteration).
- Docblock that repeats the symbol name without adding contract.
- Docblock with long examples, ASCII diagrams or change history.

### Explicit User Exception

The rule falls ONLY when the user explicitly asks for a comment on a specific
snippet (e.g.: "add a comment explaining X"). In that case:

- Write the requested comment, and only the requested snippet.
- Do not use the opportunity to comment other parts of the file.
- Keep the comment short, factual and in English.

### Where Comments EXIST TODAY (legacy)

Legacy code may contain old comments. While working on a file:
- **FORBIDDEN** to add new comments.
- Pre-existing comments in the snippet you are editing MAY be removed if you
  are rewriting that snippet (cleanup allowed, not required).
- Never "sweep" all comments from files you are not touching — that is
  opportunistic refactoring, forbidden.

### Verification before finishing

Before marking a task as done (Go or web):

1. Review every changed file and remove EVERY new comment you wrote.
2. If a comment remains, ask: is it a docblock of an exported symbol? If not,
   remove it.
3. If a snippet became "too obvious" after removing the comment, rewrite the
   code with better names — do not re-add the comment.
4. `go vet` / `tsc` must pass without any comment-related warnings.

The absence of comments is NOT lack of documentation: the README, public API
docblocks and the tests document the system. A comment that is not a docblock
is a bug.

## Design System, Sidebar and Buttons: DO NOT TOUCH (unless Explicitly Requested)

The design system (color tokens, radius, typography, fonts) and all UI
components (sidebar, buttons, app-link, card-menu, org-switcher, modals,
inputs, tables) are part of the product's visual identity.

**FORBIDDEN**, unless explicitly requested by the user:
- Changing `web/src/styles.css` (theme/`@theme`, colors, `--radius-*`, fonts).
- Changing the structure, labels or styling of the sidebar
  (`web/src/components/shell.tsx`).
- Changing colors/radius/variants of buttons (`web/src/components/ui/button.tsx`),
  links (`app-link.tsx`) or any component in `web/src/components/ui/`.
- Adding/removing fonts in `package.json` (e.g.: swapping Inter for another)
  without asking.
- "Reusing" the theme of a reference mockup/HTML page to overwrite the app's
  global theme. The global theme MUST NOT change as a side effect of
  refactoring a screen.

When refactoring a specific screen:
- Use ONLY the existing design-system tokens.
- Do not introduce hardcoded colors/radius that diverge from the theme.
- Focus on the content/structure of the screen in question; do not alter the
  global shell.

Canonical tokens (do not change):
- `primary`: Material blue (`#b0c6ff` in dark mode).
- `--radius-DEFAULT`: `0.5rem`, `--radius-lg`: `0.625rem`, `--radius-xl`: `0.75rem`.
- App font: Inter (`--font-*`).

### Exceptions
IF AND ONLY IF the user explicitly asks (e.g.: "change the primary", "reset
the radius", "alter button X"), changes are allowed. Always confirm the scope
first.

## Database — NEVER Clean

- Permanent rule: NEVER `DROP/CREATE/TRUNCATE/DELETE` real production data.
- Never `podman rm -f $(podman ps -aq)` nor `--filter name=aether-` (catches
  infra).
- Always operate containers by specific name.
- Reading the database is allowed; writing only with explicit authorization.

## Realtime / Event-Driven Architecture — Polling Forbidden

The platform is **event-driven and realtime**. The frontend MUST NOT fetch
state by polling.

### How realtime works
- **Postgres is the source of truth** for states; Redis is the realtime bus:
  pub/sub (`notify:org:<org>`) for fanout + Streams for the event log
  (`ev:org:<org>`, seq per org) and the deploy queue (`q:deployments:*`,
  consumer groups).
- **Frontend**: a single WebSocket (`/api/v1/ws/realtime`) per session, via
  `RealtimeProvider`/`NotificationProvider`; initial bootstrap over REST and
  incremental updates through WS events. Protocol: subscribe by authorized
  scope, `seq` for replay (persisted in `localStorage`), **ephemeral** events
  (e.g.: `deploy.build.log`, `app.state`) with seq=0 that do not come back in
  replay.
- The server sends ping/heartbeat; the client must respond `{"op":"ping"}`
  every ~25s (hub read timeout is 45s). The `Timeout` middleware MUST NOT be
  applied to `/api/v1/ws/` paths.
- Backend: `AETHER_RUNTIME_BACKEND=redis` in production and dev
  (`dev.sh`/`install.sh`); `memory` only for tests.

### Anti-polling rule
- **FORBIDDEN** to add polling/`setInterval`/`refetchInterval` on the frontend
  for data already delivered by WS events (deploys, app states,
  notifications).
- Polling is allowed ONLY for low-frequency telemetry or as a fallback when
  the WS is disconnected, and must be reviewed with the user first.
  Already accepted: net-q 15s, presence 30s/10s, host-stats 2s, log-follow
  SSE, notifications fallback 30s (offline only).
- Doubt → consult the user before introducing polling.

## Containers / Infra

- API, frontend, postgres and redis run in podman containers (via
  `install.sh`).
- `podman` is the only host dependency.
- After code changes: rebuild via `./install.sh start` (builds the image and
  restarts).
- Podman machine: 4GB (increase if builds OOM).

## Tests

- Suite: `AETHER_TEST_DATABASE_PORT=5433 AETHER_API_TEST_DATABASE_PORT=5433 go test ./api/internal/... -count=1 -p 1 -timeout 25m`
- Test Postgres: container `aether-test-pg` on port 5433 (do not tear down).
- Test Redis: container `aether-redis-test` on port 6380.
- Run `go build -o /tmp/aether-api ./api/cmd/api` + `go vet ./api/internal/...` before finishing.
