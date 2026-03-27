# Plan: Config Builder — Receivers & Time Intervals

**Status**: Draft
**Spec**: specs/008-config-builder-receivers/spec.md

## Architecture decisions

### AD-1: Unknown receiver detection via a `raw_yaml` escape hatch on `ReceiverDef`

**Decision**: Add `RawYAML string \`json:"raw_yaml,omitempty"\`` to `ReceiverDef`
(JSON-only; no YAML tag so it is never serialised into Alertmanager config). When
`ListReceivers` / `GetReceiver` encounters a receiver that contains YAML keys beyond
the five known config arrays, it sets this field to the YAML serialisation of that
receiver block. The frontend inspects `raw_yaml != ""` at load time to decide between
the form renderer and the textarea renderer. On `PUT /api/builder/receivers/{name}` the
handler routes to `b.SetReceiverRaw(name, rawYAML)` when the incoming `ReceiverDef` has
`raw_yaml` set.

**Rationale**: The five-type model in `ReceiverDef` cannot represent arbitrary
Alertmanager integrations (VictorOps, Telegram, MSTeams, …). Silently dropping unknown
keys on round-trip would corrupt configs. An escape hatch at the whole-receiver level
(spec decision Q3) is the least-surprising behaviour and requires zero changes to the
YAML serialisation path.

**Alternatives considered**:
- *`map[string]interface{}` catch-all field on `ReceiverDef`*: preserves unknown keys
  structurally but leaks untyped data into the typed model and complicates validation.
- *Frontend fetches raw YAML separately and extracts the block*: avoids backend changes
  but requires fragile YAML string extraction in the frontend.
- *Per-integration-config fallback* (spec option rejected in Q3): more composable but
  harder to implement correctly when a receiver mixes known and unknown integration types.

---

### AD-2: New backend endpoint `GET /api/builder/receivers/{name}/routes`

**Decision**: Add a handler `ReceiverRoutes` to `BuilderHandler` mounted at
`GET /api/builder/receivers/{name}/routes`. It loads the live routing tree via
`b.GetRoute()`, walks the tree recursively, and returns every `RouteSpec` node whose
`receiver` field equals `{name}`. The response is:

```json
{
  "receiver": "slack-critical",
  "referenced_by": [
    { "matchers": ["severity=\"critical\""], "depth": 1 },
    { "matchers": ["env=\"prod\""], "depth": 2 }
  ]
}
```

The frontend calls this endpoint when the user clicks "Delete" on a receiver. If
`referenced_by` is non-empty a blocking confirmation dialog is shown listing the
matchers of the referencing routes.

**Rationale**: Keeps routing-tree traversal logic server-side (spec decision Q2),
consistent with the "backend owns config logic" principle. The endpoint is read-only and
stateless — safe to call on every delete intent.

**Alternatives considered**:
- *Client-side tree walk after `GET /api/builder/route`*: adds traversal logic to the
  frontend, duplicates knowledge that already lives in the Go layer.

---

### AD-3: Save flow reuses `exportConfig` → `diffConfig` → `saveConfig`

**Decision**: Both the receivers page and the time-intervals page assemble a full config
by calling `POST /api/builder/export` (which merges the edited resource into the live
config), then follow the exact same diff-and-save sequence as the routing editor:
`diffConfig` → `YamlDiffViewer` → `saveConfig`. No new save mechanism is introduced.

**Rationale**: The routing editor already implements this pattern end-to-end. Reusing it
avoids divergence and means both pages automatically benefit from future save-mode
additions (e.g. webhook save, added by feature 009).

**Alternatives considered**:
- *Page-local diff/save* not using `exportConfig`: would require each page to rebuild the
  entire config from scratch, which is fragile and duplicates the builder's assembly logic.

---

### AD-4: Debounced validation using Svelte's `$effect` rune

**Decision**: Each editor page uses a `$effect` rune that watches the current form state
and, after a 500 ms debounce (via `setTimeout` / `clearTimeout`), calls the appropriate
validate endpoint (`POST /api/builder/receivers/validate` or
`POST /api/builder/time-intervals/validate`). Validation results are stored in a
reactive `validationErrors` state variable and rendered below the form.

**Rationale**: Matches the SvelteKit rune pattern already used in the routing page.
500 ms avoids hitting the backend on every keystroke while still giving fast feedback.

---

### AD-5: `ReceiverEditor` extracted as a reusable Svelte component

**Decision**: The receiver editing UI (integration-type selector, per-type field sets,
raw YAML textarea fallback, validation error display) is implemented in a new
`ReceiverEditor.svelte` component following the `RouteNodeEditor` pattern: receives
`receiver: ReceiverDef` and `onUpdate: (r: ReceiverDef) => void` props, immutable
updates only.

**Rationale**: The receivers page already has a three-column layout (list / editor /
diff). Keeping the editor isolated makes the page code manageable and allows the
component to be reused in future contexts (e.g. inline receiver creation from a route
node). The time-interval editor is simpler (flat form) and stays inline in the page.

---

## Impacted files

| File | Action | Description |
|------|--------|-------------|
| `internal/configbuilder/model.go` | Modify | Add `RawYAML string` (JSON-only) to `ReceiverDef` |
| `internal/configbuilder/builder.go` | Modify | Detect unknown receiver types in `ListReceivers`; add `SetReceiverRaw`; route `UpsertReceiver` through raw path when `RawYAML` set |
| `internal/api/handlers/builder.go` | Modify | Add `ReceiverRoutes` handler; update `ListReceivers` / `GetReceiver` to annotate unknown receivers; update `UpsertReceiver` to handle raw YAML path |
| `internal/api/router.go` | Modify | Mount `GET /builder/receivers/{name}/routes` |
| `internal/api/handlers/builder_test.go` | Create | Table-driven tests for `ReceiverRoutes`, unknown-receiver round-trip, time-interval CRUD (new happy-path + error cases not yet covered) |
| `web/src/lib/api/types.ts` | Modify | Replace `BuilderReceiverDef` stub; add `ReceiverDef`, `SlackConfigDef`, `EmailConfigDef`, `PagerdutyConfigDef`, `OpsgenieConfigDef`, `WebhookConfigDef`, `TimeIntervalEntry`, `TimeIntervalDef`, `TimeRangeDef` |
| `web/src/lib/api/builder.ts` | Modify | Add full CRUD + validate functions for receivers and time intervals; add `getReceiverRoutes` |
| `web/src/lib/components/config/ReceiverEditor.svelte` | Create | Extracted receiver form component: type selector, per-type fields, raw YAML fallback, validation error display |
| `web/src/routes/config/receivers/+page.svelte` | Modify | Migrate to builder API; integrate `ReceiverEditor`; wire delete guard; add diff/save flow via `exportConfig` |
| `web/src/routes/config/time-intervals/+page.svelte` | Modify | Migrate to builder API; add all six field groups; wire inline validation; add diff/save flow via `exportConfig` |

---

## Implementation phases

### Phase 1 — Backend: unknown receiver detection + route-reference endpoint

**Goal**: The builder correctly round-trips receivers with unknown integration types, and
exposes which routes reference a given receiver name.

- Modify `internal/configbuilder/model.go`: add `RawYAML string \`json:"raw_yaml,omitempty"\`` to `ReceiverDef`.
- Modify `internal/configbuilder/builder.go`:
  - In `ListReceivers`, after unmarshalling, do a secondary raw `map[string]any` parse of
    each receiver block; if keys beyond the five known types are present, set `rec.RawYAML`.
  - Add `SetReceiverRaw(name, rawYAML string) error`: locate the receiver block in the raw
    YAML by name and replace it with the provided YAML fragment.
  - In `UpsertReceiver`, if `rec.RawYAML != ""`, delegate to `SetReceiverRaw`.
- Modify `internal/api/handlers/builder.go`:
  - Update `ListReceivers` / `GetReceiver` — no change needed at handler level (builder
    now populates `raw_yaml` transparently).
  - Update `UpsertReceiver` to pass the `ReceiverDef` as-is (builder handles routing).
  - Add `ReceiverRoutes(w, r)`: load route tree, walk recursively, collect matching nodes.
- Modify `internal/api/router.go`: mount `GET /builder/receivers/{name}/routes`.
- Create `internal/api/handlers/builder_test.go`: cover `ReceiverRoutes` (found / not found / nested match), unknown-receiver round-trip (list → raw YAML preserved → upsert → re-list).

**Files**: `model.go`, `builder.go`, `handlers/builder.go`, `router.go`, `handlers/builder_test.go`

---

### Phase 2 — Frontend: types and API client

**Goal**: TypeScript has full typed coverage of all builder resources; the API client
exports every function required by the pages.

- Modify `web/src/lib/api/types.ts`:
  - Remove `BuilderReceiverDef` stub.
  - Add: `ReceiverDef`, `WebhookConfigDef`, `SlackConfigDef`, `EmailConfigDef`,
    `PagerdutyConfigDef`, `OpsgenieConfigDef`, `TimeIntervalEntry`, `TimeIntervalDef`,
    `TimeRangeDef`.
  - All fields exactly mirror the Go backend JSON tags.
- Modify `web/src/lib/api/builder.ts`:
  - Replace `listBuilderReceivers` stub with full-signature `listReceivers`.
  - Add: `getReceiver`, `upsertReceiver`, `deleteReceiver`, `validateReceiver`,
    `getReceiverRoutes`.
  - Add: `listTimeIntervals`, `getTimeInterval`, `upsertTimeInterval`,
    `deleteTimeInterval`, `validateTimeInterval`.
  - Export interfaces `BuilderReceiverRouteRef` and `BuilderReceiverRoutesResponse` for
    the delete guard.

**Files**: `types.ts`, `builder.ts`

---

### Phase 3 — Frontend: receivers page

**Goal**: The receivers page uses the builder API end-to-end, with full form support,
inline validation, delete guard, and diff/save.

- Create `web/src/lib/components/config/ReceiverEditor.svelte`:
  - Props: `receiver: ReceiverDef`, `onUpdate: (r: ReceiverDef) => void`,
    `validationErrors: string[]`, `readonly?: boolean`.
  - When `receiver.raw_yaml` is set: render a single `<textarea>` with the raw YAML.
  - Otherwise: render integration-type selector (Slack / PagerDuty / Email / OpsGenie /
    Webhook) with `+` button; each added integration renders its type-specific fields.
  - Each integration config can be removed individually (trash icon).
  - `send_resolved` checkbox present on all types.
  - Validation errors displayed as a summary list below the form.
- Modify `web/src/routes/config/receivers/+page.svelte`:
  - On mount: `listReceivers(instance)` → populate list.
  - Selecting a receiver: set `editing = ReceiverDef` (clone).
  - "+ Add receiver": push a fresh `{ name: '', raw_yaml: '' }` draft.
  - Save button: `upsertReceiver(instance, editing)` → refresh list.
  - Delete button:
    1. Call `getReceiverRoutes(instance, name)`.
    2. If `referenced_by.length > 0`: show modal listing routes + "Delete anyway" / "Cancel".
    3. Otherwise: `deleteReceiver(instance, name)` immediately.
  - Validation: `$effect` debounce 500 ms → `validateReceiver(editing)` → update
    `validationErrors`.
  - "Preview diff" button: `exportConfig({ instance, receivers: [editing] })` → `diffConfig`
    → switch to diff step → `YamlDiffViewer` + save options + `saveConfig`.
  - Read-only mode for viewers: list rendered without edit/add/delete controls.

**Files**: `ReceiverEditor.svelte`, `receivers/+page.svelte`

---

### Phase 4 — Frontend: time intervals page

**Goal**: The time-intervals page uses the builder API end-to-end, with full six-field
form, inline validation, and diff/save.

- Modify `web/src/routes/config/time-intervals/+page.svelte`:
  - On mount: `listTimeIntervals(instance)` → populate list.
  - Each `TimeIntervalEntry` displays its name (editable) and a list of `TimeIntervalDef`
    specs. Each spec has:
    - **Time ranges**: repeatable `HH:MM – HH:MM` row with `+` / `−` buttons.
    - **Weekdays**: checkbox row (Mon–Sun), unchanged from current UI.
    - **Days of month**: plain text input (`1:15`, `-1`, comma-separated ranges).
    - **Months**: plain text input (`january:march`, `6`, comma-separated ranges).
    - **Years**: plain text input (`2024:2026`, comma-separated ranges).
    - **Timezone**: plain text input (IANA), unchanged.
    - "+ Add spec" button appends an empty `TimeIntervalDef` to the entry's slice.
  - "+ Add interval" button appends a fresh `TimeIntervalEntry`.
  - Delete interval button: `deleteTimeInterval(instance, name)` (no route guard needed
    for time intervals; dangling references are a validation concern, not a blocker).
  - Validation: `$effect` debounce 500 ms → `validateTimeInterval(entry)` → inline errors.
  - "Preview diff" button: `exportConfig({ instance, time_intervals: [entry] })` →
    `diffConfig` → diff step → `saveConfig`.
  - Read-only mode for viewers.

**Files**: `time-intervals/+page.svelte`

---

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Raw YAML round-trip corrupts unknown receiver config (whitespace, ordering) | Medium | Use `gopkg.in/yaml.v3` marshal/unmarshal for the round-trip; add a specific test that asserts the original YAML keys survive a list → upsert cycle |
| `SetReceiverRaw` YAML surgery introduces syntax errors | Medium | Validate the patched config with `configbuilder.Validate` before returning; return 422 if invalid |
| `exportConfig` with only one resource (receiver or time interval) clobbers unrelated config changes made in another browser tab | Low | This is a pre-existing limitation of the stateless builder architecture; documented in the constitution (no persistent state). Out of scope for this feature. |
| Delete guard endpoint adds latency on every delete click | Low | The endpoint is O(depth of routing tree) which is bounded in practice. No caching needed. |
| TypeScript strict mode rejects `raw_yaml` field on `ReceiverDef` when absent | Low | Field is `raw_yaml?: string` (optional); callers use `?? ''` guard when rendering |
