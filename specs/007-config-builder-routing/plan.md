# Plan: Config Builder — Routing Tree Editor

**Status**: Draft
**Spec**: specs/007-config-builder-routing/spec.md

## Architecture decisions

### AD-1: Backend requires zero changes

**Decision**: No Go code is touched. All required endpoints already exist and
are correctly wired under `requireConfigEditor` RBAC middleware:
- `GET  /api/builder/route` — load the root route as `RouteSpec`
- `PUT  /api/builder/route` — replace the root route
- `POST /api/builder/export` — assemble + validate full config YAML
- `GET  /api/builder/receivers` — list receiver names
- `POST /api/config/diff` / `POST /api/config/save` — diff and persist

**Rationale**: The backend was built ahead of the UI. The only work is frontend.

**Alternatives considered**: None — the API surface is complete.

---

### AD-2: New typed `builder.ts` API client

**Decision**: Create `web/src/lib/api/builder.ts` covering the four builder
endpoints used by the routing page. Add `RouteSpec` and `BuilderReceiverDef`
types to `types.ts`. The existing `config.ts` functions (`diffConfig`,
`saveConfig`) are reused as-is for the persist flow.

**Rationale**: The current page imports from `config.ts` and manipulates YAML
strings directly instead of the structured builder API. Introducing a typed
client gives compile-time safety and makes the intent explicit.

**Alternatives considered**: Extending `config.ts` — rejected because config.ts
is about raw YAML operations; the builder layer is a distinct concern.

---

### AD-3: Live YAML preview via local `js-yaml` serialisation

**Decision**: The YAML preview panel is derived reactively from the form state
using `js-yaml.dump()` inside a Svelte `$derived` block. No network call is
made during editing.

**Rationale**: Per spec decision Q2. `js-yaml` already ships as a dependency
(used on the current routing page). The server-side serialiser is only invoked
on "Preview diff" (which calls `POST /api/config/diff`), ensuring the final
YAML is always validated before save.

**Alternatives considered**: Calling `POST /api/builder/export` on every
change — rejected: too chatty for a keystroke-level reactive update.

---

### AD-4: Sibling reorder via up/down arrow buttons (no drag-and-drop)

**Decision**: Each non-root child route gets Up (▲) and Down (▼) buttons in
its header. The parent `RouteNodeEditor` receives an `onMove` callback
`(index: number, direction: 'up' | 'down') => void`. The root of the form
tree owns the mutation and propagates updated arrays down.

**Rationale**: Per spec (drag-and-drop is explicitly out of scope). Arrow
buttons are accessible, keyboard-friendly, and require no external library.

**Alternatives considered**: Drag-and-drop — out of scope per spec.

---

### AD-5: Delete confirmation via inline `window.confirm`

**Decision**: The "remove child route" action calls `window.confirm()` with a
short message before removing the node. No custom dialog component is needed.

**Rationale**: The existing codebase has no shared dialog/modal primitive.
Using `window.confirm` is the simplest compliant solution (verifiable: AC-3
says "a confirmation dialog is shown", not "a custom modal"). A shared dialog
component can be added in a later feature if needed.

**Alternatives considered**: Inline "Are you sure?" expand-in-place — more
work, same result.

---

### AD-6: Read-only mode for viewers via `RoutingTree.svelte` branch

**Decision**: The routing page (`/config/routing`) checks `$canEditConfig` at
render time. If false, it renders `RoutingTree.svelte` (read-only tree
visualiser) and hides all edit controls. The config layout
(`+layout.svelte`) is relaxed to redirect to `/login` only for unauthenticated
users (not for authenticated viewers), so viewers can reach `/config/routing`.

**Rationale**: Per spec decision Q3. `RoutingTree.svelte` already exists and
is the canonical read-only view. The layout change is minimal: one condition
change (`!$isAuthenticated` instead of `!$canEditConfig`).

**Alternatives considered**: Keeping the layout blocking all non-config-editors
and creating a separate `/routing-config` read-only route — rejected as
duplication; the existing route should serve both roles.

---

### AD-7: Route state owned by the page, not a store

**Decision**: The loaded `RouteSpec` and the `formRoute: RouteFormNode` are
local `$state` variables on the routing page. No shared Svelte store is
introduced.

**Rationale**: Routing config is not consumed by any other page or component.
A store would add indirection with no benefit for a page-scoped concern.

---

## Impacted files

| File | Action | Description |
|------|--------|-------------|
| `web/src/lib/api/builder.ts` | **Create** | Typed client: `getRoute`, `setRoute`, `exportConfig`, `listBuilderReceivers` |
| `web/src/lib/api/types.ts` | **Modify** | Add `RouteSpec` and `BuilderReceiverDef` interfaces |
| `web/src/lib/components/config/RouteNodeEditor.svelte` | **Modify** | Receiver `<select>`, up/down reorder, delete confirmation |
| `web/src/routes/config/routing/+page.svelte` | **Modify** | Full refactor: use builder API, live preview, read-only branch |
| `web/src/routes/config/+layout.svelte` | **Modify** | Relax redirect: block unauthenticated only, not viewers |

No Go files modified.

---

## Implementation phases

### Phase 1 — Types & API client
**Goal**: Establish the typed foundation that the UI will build on.
- Add `RouteSpec` (mirrors `configbuilder.RouteSpec` Go struct) and
  `BuilderReceiverDef` (`{ name: string }`) to `types.ts`
- Create `web/src/lib/api/builder.ts`:
  - `getRoute(instance?: string): Promise<{ route: RouteSpec }>`
  - `setRoute(instance: string, route: RouteSpec): Promise<{ route: RouteSpec; raw_yaml: string; validation: ValidationResult }>`
  - `exportConfig(req: ExportConfigRequest): Promise<{ raw_yaml: string; validation: ValidationResult }>`
  - `listBuilderReceivers(instance?: string): Promise<{ receivers: BuilderReceiverDef[] }>`

### Phase 2 — RouteNodeEditor improvements
**Goal**: Close the three UI gaps in the recursive form component.
- **Receiver `<select>`**: change `<input>` to `<select>` fed by a new
  `availableReceivers: string[]` prop (passed down from the page)
- **Up/Down reorder**: add `index`, `total`, and `onMove` props to each non-root
  node; render ▲/▼ buttons in the node header; parent `patchChild` / reorder logic
  handles the array mutation
- **Delete confirmation**: wrap `removeChild(i)` in `window.confirm()`

### Phase 3 — Routing page refactor & read-only mode
**Goal**: Wire the page to the builder API and satisfy all remaining ACs.
- Load via `getRoute()` on mount and instance change (replaces raw YAML fetch)
- Load receivers via `listBuilderReceivers()` and pass as `availableReceivers`
  to `RouteNodeEditor`
- Replace manual `syncYamlToForm` / `syncFormToYaml` with a `$derived` block
  that serialises `formRoute` to YAML via `js-yaml.dump()` at all times
- Keep the YAML editor tab (for power users) — it still drives `rawYaml` and
  sync-on-tab-switch stays for that direction
- Read-only branch: if `!$canEditConfig`, render `<RoutingTree>` instead of the
  form; hide the YAML tab and all action buttons
- Adjust `+layout.svelte` redirect: `!$isAuthenticated` → `/login`

---

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Layout relaxation exposes time-interval / receiver tabs to viewers prematurely | Medium | Those pages already require `config-editor` actions to be useful; they show empty states for viewers. Acceptable until 008 is specced. |
| `RouteFormNode` ↔ `RouteSpec` conversion has edge cases (e.g. string vs. object matchers from live config) | Medium | The conversion logic already exists in the page; extract and cover with unit tests in this phase. |
| `window.confirm` blocked in embedded / iframe deployments | Low | AlertLens is not designed for iframe embedding. Acceptable for MVP. |
| Receiver dropdown empty when receivers not yet loaded | Low | Show a loading state and fall back to free-text input if the API call fails. |
