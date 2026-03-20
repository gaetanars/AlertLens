# Tasks: Config Builder — Routing Tree Editor

**Status**: Ready
**Total**: 7 tasks · 3 phases

## Phase 1 — Types & API client

- [x] **T-1.1**: Add `RouteSpec` and `BuilderReceiverDef` types to `types.ts`
  - Files: `web/src/lib/api/types.ts`
  - Add `RouteSpec` (mirrors Go `configbuilder.RouteSpec`): fields `receiver`, `group_by`, `matchers`, `continue`, `group_wait`, `group_interval`, `repeat_interval`, `mute_time_intervals`, `active_time_intervals`, `routes: RouteSpec[]` — all optional except `routes`
  - Add `BuilderReceiverDef`: `{ name: string }`
  - Test: `npx tsc --noEmit` in `web/` passes with no new errors
  - Developer writes: No

- [x] **T-1.2**: Create `web/src/lib/api/builder.ts` with four typed functions
  - Files: `web/src/lib/api/builder.ts` (create)
  - Functions:
    - `getRoute(instance?: string): Promise<{ route: RouteSpec }>`
    - `setRoute(instance: string, route: RouteSpec): Promise<{ route: RouteSpec; raw_yaml: string; validation: ValidationResult }>`
    - `exportConfig(req: { instance?: string; route?: RouteSpec }): Promise<{ raw_yaml: string; validation: ValidationResult }>`
    - `listBuilderReceivers(instance?: string): Promise<{ receivers: BuilderReceiverDef[] }>`
  - Pattern: follow `routing.ts` — use `api.get` / `api.post` from `./client`; encode `instance` as `?instance=` query param
  - Test: `npx tsc --noEmit` passes; functions are importable
  - Developer writes: No

## Phase 2 — RouteNodeEditor improvements

- [x] **T-2.1**: Replace free-text receiver input with a `<select>` dropdown
  - Files: `web/src/lib/components/config/RouteNodeEditor.svelte`
  - Add prop `availableReceivers: string[]` (default `[]`) to the component interface
  - When `availableReceivers.length > 0`: render `<select>` bound to `route.receiver`; include an empty `<option value="">— select receiver —</option>` first, then one `<option>` per name
  - When empty (loading or API error): keep the existing `<input>` as fallback (same placeholder)
  - The prop must be threaded recursively to all child `<RouteNodeEditor>` instances
  - Test: pass `availableReceivers={['team-a', 'team-b']}` in a Svelte component test or manual check; confirm `<select>` renders; without prop, `<input>` renders
  - Developer writes: No

- [x] **T-2.2**: Add up/down sibling reorder controls
  - Files: `web/src/lib/components/config/RouteNodeEditor.svelte`
  - Add props `index: number`, `total: number`, `onMove: (dir: 'up' | 'down') => void` (all with sensible defaults so root node is unaffected)
  - In the node header (non-root only): render ▲ button (disabled when `index === 0`) and ▼ button (disabled when `index === total - 1`), each calling `onMove('up')` / `onMove('down')`
  - In the parent's child list rendering: pass `index={i}`, `total={route.routes.length}`, and `onMove` that mutates the `routes` array (swap the element at `i` with `i-1` or `i+1`)
  - Thread `index`, `total`, `onMove` are NOT passed further down (only the immediate parent controls its own children)
  - Test: verify buttons appear for middle children, ▲ disabled for first child, ▼ disabled for last child
  - Developer writes: No

- [x] **T-2.3**: Add delete confirmation before removing a child route
  - Files: `web/src/lib/components/config/RouteNodeEditor.svelte`
  - Wrap the existing `removeChild(i)` call with `window.confirm('Remove this route and all its children?')` — only remove if confirmed
  - Test: confirm clicking the trash icon now shows a browser confirm dialog before removing
  - Developer writes: No

## Phase 3 — Routing page refactor & read-only mode

- [x] **T-3.1**: Refactor data loading to use `GET /api/builder/route` and `GET /api/builder/receivers`
  - Files: `web/src/routes/config/routing/+page.svelte`
  - Replace the current `fetchConfig()` call with `getRoute(instance)` from `builder.ts`
  - Add a parallel `listBuilderReceivers(instance)` call; store names in `availableReceivers: string[]`
  - Remove the manual YAML parse used to extract `availableTimeIntervals` from raw YAML; instead derive them from the route tree itself (collect all `mute_time_intervals` + `active_time_intervals` values as a deduplicated list), or keep a lightweight separate call to `fetchConfig` just for time intervals — **prefer** deriving from route tree to avoid an extra call
  - Pass `availableReceivers` down to `<RouteNodeEditor>`
  - Test: page loads and populates the receiver dropdown; reload on instance change works
  - Developer writes: No

- [x] **T-3.2**: Replace manual YAML sync with a `$derived` live preview and wire save flow to builder API
  - Files: `web/src/routes/config/routing/+page.svelte`
  - Replace the `rawYaml` `$state` (in form tab) with a `$derived` that calls `js-yaml.dump(formRouteToYaml(formRoute))` — so YAML preview updates on every form change without tab switching
  - The YAML tab keeps its own `$state` (for manual edits); `switchTab('yaml')` copies the derived YAML into it, `switchTab('form')` parses it into `formRoute` (existing logic)
  - For `previewDiff`: when in form tab, use the derived YAML directly (no `syncFormToYaml()` needed)
  - Wire save: after a successful `saveConfig()`, call `getRoute(instance)` to refresh `formRoute` from server (ensures form reflects persisted state)
  - AC-8: the form tab now uses structured data all the way through; raw YAML is only a view
  - Test: edit a field in the form tab → YAML preview updates instantly without switching tabs
  - Developer writes: No

- [x] **T-3.3**: Read-only mode for viewers + layout relaxation
  - Files: `web/src/routes/config/routing/+page.svelte`, `web/src/routes/config/+layout.svelte`
  - In `+layout.svelte`: change redirect condition from `!$canEditConfig` to `!$isAuthenticated`; import `isAuthenticated` from `$lib/stores/auth` (already exported)
  - In the routing page: import `canEditConfig` from `$lib/stores/auth`; wrap the entire editor (both tabs + action buttons) in `{#if $canEditConfig}...{:else}<RoutingTree>{/if}`
  - For the read-only branch: load route via `fetchRouting(instance)` (existing read-only API, no RBAC issue for viewers); show `<RoutingTree route={routeData.route} />` with an informational banner "You need the config-editor role to edit routing rules."
  - Test: log in as viewer → `/config/routing` shows the tree with the banner, no form; log in as config-editor → form is shown
  - Developer writes: No
