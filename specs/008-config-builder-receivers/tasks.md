# Tasks: Config Builder — Receivers & Time Intervals

**Status**: Ready
**Total**: 9 tasks · 4 phases

---

## Phase 1 — Backend

- [x] **T-1.1**: Detect unknown receiver types and add `RawYAML` escape hatch to `ReceiverDef`
  - **What**: Add `RawYAML string \`json:"raw_yaml,omitempty"\`` (no `yaml:` tag) to `ReceiverDef` in `model.go`. In `builder.go`, extend `parseReceivers()` to do a secondary raw-map pass over `b.raw["receivers"]`: for each receiver entry, check whether any map key other than `name`, `webhook_configs`, `slack_configs`, `email_configs`, `pagerduty_configs`, `opsgenie_configs` is present; if so, marshal that raw entry back to YAML and set `rec.RawYAML` before returning. The typed fields (`WebhookConfigs`, etc.) remain zero-valued for these receivers so the handler / frontend can unambiguously detect the fallback case via `raw_yaml != ""`.
  - **Files**: `internal/configbuilder/model.go`, `internal/configbuilder/builder.go`
  - **Test**: `go build ./...` succeeds; covered by T-1.4 round-trip test.
  - **Developer writes**: No

- [x] **T-1.2**: Add `SetReceiverRaw` to `ConfigBuilder` and route `UpsertReceiver` through it
  - **What**: Add `SetReceiverRaw(name, rawYAML string) error` to `ConfigBuilder`. The method parses `rawYAML` with `gopkg.in/yaml.v3` into a `map[string]interface{}`, asserts `name` in the map matches the argument (or injects it), then replaces the matching entry in `b.raw["receivers"]` by walking the raw slice; if not found, appends. Modify `UpsertReceiver`: if `rec.RawYAML != ""`, delegate to `SetReceiverRaw(rec.Name, rec.RawYAML)` and return; otherwise continue with the existing typed path. Add `ValidateReceiver` handling: when `rec.RawYAML != ""`, embed the raw YAML fragment directly instead of marshalling the typed struct (so validation is accurate for unknown types). Finally, in the `UpsertReceiver` handler in `handlers/builder.go`, pass the `ReceiverDef` as-is — no handler change is needed since the builder now routes internally.
  - **Files**: `internal/configbuilder/builder.go`
  - **Test**: `go build ./...` succeeds; covered by T-1.4 round-trip test.
  - **Developer writes**: No

- [x] **T-1.3**: Add `GET /api/builder/receivers/{name}/routes` endpoint
  - **What**: Add `ReceiverRoutes(w http.ResponseWriter, r *http.Request)` to `BuilderHandler` in `handlers/builder.go`. It calls `b.GetRoute()`, then recursively walks the `RouteSpec` tree collecting every node whose `Receiver` field equals the URL `{name}` parameter. Each match is recorded as `struct { Matchers []string \`json:"matchers"\`; Depth int \`json:"depth"\` }`. Response: `{"receiver": "<name>", "referenced_by": [...]}` (empty slice, not null, when nothing references it). Mount in `router.go` as `r.Get("/receivers/{name}/routes", bldrH.ReceiverRoutes)` inside the existing `/builder` route block with `requireConfigEditor` already in scope.
  - **Files**: `internal/api/handlers/builder.go`, `internal/api/router.go`
  - **Test**: `go build ./...` succeeds; covered by T-1.4 handler test.
  - **Developer writes**: No

- [x] **T-1.4**: Backend tests for new builder behaviour
  - **What**: Create `internal/api/handlers/builder_test.go` with table-driven parallel tests covering:
    1. `ReceiverRoutes` — (a) receiver not referenced by any route → `referenced_by: []`; (b) receiver is the root route's receiver → depth 0 in result; (c) receiver is referenced only in a nested child route → correct depth reported; (d) name not found in route tree at all.
    2. Unknown-receiver round-trip — build a raw YAML config containing a `victorops_configs:` receiver, call `ListReceivers`, assert the receiver comes back with a non-empty `raw_yaml` and empty typed-config slices; then call `UpsertReceiver` with that `ReceiverDef` (raw_yaml set), call `BuildRaw`, unmarshal the YAML and assert `victorops_configs` is still present with the original value.
    3. `SetReceiverRaw` error path — passing malformed YAML returns a non-nil error.
  - **Files**: `internal/api/handlers/builder_test.go`
  - **Test**: `go test ./internal/api/handlers/... -race -count=1` passes.
  - **Developer writes**: No

---

## Phase 2 — Frontend: types & API client

- [x] **T-2.1**: Replace `BuilderReceiverDef` stub and add all builder types to `types.ts`
  - **What**: In `web/src/lib/api/types.ts`, remove the `BuilderReceiverDef` stub interface. Add the following interfaces, with all field names and optionality exactly matching Go backend JSON tags:
    - `WebhookConfigDef` — `url: string; send_resolved?: boolean; max_alerts?: number`
    - `SlackConfigDef` — `channel: string; api_url?: string; username?: string; text?: string; title?: string; send_resolved?: boolean`
    - `EmailConfigDef` — `to: string; from?: string; smarthost?: string; auth_username?: string; auth_password?: string; send_resolved?: boolean`
    - `PagerdutyConfigDef` — `routing_key?: string; service_key?: string; description?: string; send_resolved?: boolean`
    - `OpsgenieConfigDef` — `api_key?: string; message?: string; priority?: string; send_resolved?: boolean`
    - `ReceiverDef` — `name: string; webhook_configs?: WebhookConfigDef[]; slack_configs?: SlackConfigDef[]; email_configs?: EmailConfigDef[]; pagerduty_configs?: PagerdutyConfigDef[]; opsgenie_configs?: OpsgenieConfigDef[]; raw_yaml?: string`
    - `TimeRangeDef` — `start_time: string; end_time: string`
    - `TimeIntervalDef` — `times?: TimeRangeDef[]; weekdays?: string[]; days_of_month?: string[]; months?: string[]; years?: string[]; location?: string`
    - `TimeIntervalEntry` — `name: string; time_intervals: TimeIntervalDef[]`
    - `BuilderReceiverRouteRef` — `matchers: string[]; depth: number`
    - `BuilderReceiverRoutesResponse` — `receiver: string; referenced_by: BuilderReceiverRouteRef[]`
  - **Files**: `web/src/lib/api/types.ts`
  - **Test**: `cd web && npx tsc --noEmit` passes.
  - **Developer writes**: No

- [x] **T-2.2**: Add full CRUD and validate functions to `builder.ts`
  - **What**: Modify `web/src/lib/api/builder.ts`. Replace the `listBuilderReceivers` function with correctly-typed equivalents. Add:
    - `listReceivers(instance?: string): Promise<{ receivers: ReceiverDef[] }>`
    - `getReceiver(name: string, instance?: string): Promise<ReceiverDef>`
    - `upsertReceiver(name: string, rec: ReceiverDef, instance?: string): Promise<{ receiver: ReceiverDef; raw_yaml: string; validation: ValidationResult }>`
    - `deleteReceiver(name: string, instance?: string): Promise<{ deleted: string; raw_yaml: string; validation: ValidationResult }>`
    - `validateReceiver(rec: ReceiverDef): Promise<ValidationResult>`
    - `getReceiverRoutes(name: string, instance?: string): Promise<BuilderReceiverRoutesResponse>`
    - `listTimeIntervals(instance?: string): Promise<{ time_intervals: TimeIntervalEntry[] }>`
    - `getTimeInterval(name: string, instance?: string): Promise<TimeIntervalEntry>`
    - `upsertTimeInterval(name: string, entry: TimeIntervalEntry, instance?: string): Promise<{ time_interval: TimeIntervalEntry; raw_yaml: string; validation: ValidationResult }>`
    - `deleteTimeInterval(name: string, instance?: string): Promise<{ deleted: string; raw_yaml: string; validation: ValidationResult }>`
    - `validateTimeInterval(entry: TimeIntervalEntry): Promise<ValidationResult>`
    Also update `ExportConfigRequest` to add `receivers?: ReceiverDef[]` and `time_intervals?: TimeIntervalEntry[]` fields.
  - **Files**: `web/src/lib/api/builder.ts`
  - **Test**: `cd web && npx tsc --noEmit` passes.
  - **Developer writes**: No

---

## Phase 3 — Frontend: receivers page

- [x] **T-3.1**: Create `ReceiverEditor.svelte` component
  - **What**: Create `web/src/lib/components/config/ReceiverEditor.svelte`. Props (using Svelte 5 rune syntax): `receiver: ReceiverDef`, `onUpdate: (r: ReceiverDef) => void`, `validationErrors?: string[]`, `readonly?: boolean`.
    - **Raw YAML mode** (`receiver.raw_yaml` is non-empty): render a `<textarea>` bound to the raw YAML string; on change call `onUpdate({ ...receiver, raw_yaml: value })`.
    - **Form mode**: render a "Add integration" dropdown (options: Webhook, Slack, PagerDuty, Email, OpsGenie); clicking an option appends an empty config object to the matching `*_configs` array and calls `onUpdate`. For each existing config in each array, render a collapsible card with:
      - Integration type badge
      - Type-specific fields (all text `<input>` elements; `send_resolved` as `<input type="checkbox">`)
      - Trash button that removes the config from its array and calls `onUpdate`
    - Field sets per type:
      - **Webhook**: `url` (required), `max_alerts` (number input), `send_resolved`
      - **Slack**: `channel` (required), `api_url`, `username`, `title`, `text`, `send_resolved`
      - **PagerDuty**: `routing_key` OR `service_key` (at least one), `description`, `send_resolved`
      - **Email**: `to` (required), `from`, `smarthost`, `auth_username`, `auth_password`, `send_resolved`
      - **OpsGenie**: `api_key`, `message`, `priority`, `send_resolved`
    - Below the form, if `validationErrors` is non-empty, render a red alert box listing each error.
    - All edit controls hidden when `readonly` is true.
  - **Files**: `web/src/lib/components/config/ReceiverEditor.svelte`
  - **Test**: `cd web && npx tsc --noEmit` passes; component renders without console errors when used with a mock `ReceiverDef`.
  - **Developer writes**: No

- [x] **T-3.2**: Migrate receivers page to builder API with delete guard and diff/save flow
  - **What**: Rewrite `web/src/routes/config/receivers/+page.svelte` to use the builder API end-to-end. Keep the existing three-column layout and instance selector.
    - **Data loading**: on mount and on instance change, call `listReceivers(instance)` to populate the list; replace the current raw-config parse.
    - **Selection**: clicking a receiver in the list sets `editing: ReceiverDef` to a deep clone of it; clicking "+ Add receiver" sets `editing` to `{ name: '', webhook_configs: [], slack_configs: [], email_configs: [], pagerduty_configs: [], opsgenie_configs: [] }`.
    - **Editor**: mount `<ReceiverEditor receiver={editing} onUpdate={r => editing = r} validationErrors={validationErrors} readonly={!$canEditConfig} />`.
    - **Inline validation**: `$effect` watches `editing`; after 500 ms debounce calls `validateReceiver(editing)` and stores errors in `validationErrors: string[]`.
    - **Save (upsert)**: "Save receiver" button calls `upsertReceiver(editing.name, editing, instance)`, then refreshes the list; shows toast/error on failure.
    - **Delete guard**: delete button calls `getReceiverRoutes(editing.name, instance)`; if `referenced_by.length > 0`, shows an inline confirmation section (not a browser confirm dialog) listing each referencing route's matchers and depth, with "Delete anyway" and "Cancel" buttons; otherwise calls `deleteReceiver` immediately.
    - **Diff/save flow**: "Preview diff" button calls `exportConfig({ instance, receivers: [editing] })` to get a full merged config YAML, then calls `diffConfig(instance, mergedYaml)` to get the diff, then switches to a diff step showing `<YamlDiffViewer>` and the same save-options UI (disk / GitHub / GitLab) as the routing page; "Confirm and save" calls `saveConfig(...)`.
    - **Read-only**: import `canEditConfig` from `$lib/stores/auth`; hide all edit, add, delete, and save controls when `!$canEditConfig`.
  - **Files**: `web/src/routes/config/receivers/+page.svelte`
  - **Test**: `cd web && npx tsc --noEmit` passes; `npm test` passes; manual smoke test: add a Slack receiver, validate, preview diff, verify YAML contains the new receiver.
  - **Developer writes**: No

---

## Phase 4 — Frontend: time intervals page

- [x] **T-4.1**: Migrate time intervals page to builder API with full field set, inline validation, and diff/save flow
  - **What**: Rewrite `web/src/routes/config/time-intervals/+page.svelte` to use the builder API. Keep the existing single-column layout and instance selector.
    - **Data loading**: on mount call `listTimeIntervals(instance)`; replace the current raw-config parse.
    - **State**: `intervals: TimeIntervalEntry[]` (reactive); `editingIdx: number | null` to track which interval is expanded for editing.
    - **Add interval**: "+ Add time interval" appends `{ name: '', time_intervals: [emptySpec()] }` where `emptySpec()` returns `{ times: [], weekdays: [], days_of_month: [], months: [], years: [], location: '' }`.
    - **Interval editor** (shown inline below each list item when selected): editable name field; for each `TimeIntervalDef` spec in `entry.time_intervals`:
      - **Time ranges**: repeatable pair of `<input type="time">` fields (`HH:MM`); `+` adds an empty `{ start_time: '', end_time: '' }`, `−` removes; maps to `times: TimeRangeDef[]`.
      - **Weekdays**: seven checkboxes (Monday – Sunday); checked state maps to inclusion in the `weekdays: string[]` array using lowercase full-day names (`"monday"`, …, `"sunday"`); preserves existing range syntax (e.g. `"monday:friday"`) by keeping pre-existing values that cannot be represented as single-day checkboxes in the array unchanged.
      - **Days of month**: single `<input type="text">` with placeholder `1:15, -1`; value stored as a comma-separated string parsed to/from `days_of_month: string[]` on read and write.
      - **Months**: single `<input type="text">` with placeholder `january:march, 12`; parsed to/from `months: string[]`.
      - **Years**: single `<input type="text">` with placeholder `2024:2026`; parsed to/from `years: string[]`.
      - **Timezone**: single `<input type="text">` with placeholder `Europe/Paris`; maps to `location: string`.
      - "+ Add spec" appends an empty spec; trash icon removes it.
    - **Delete interval**: calls `deleteTimeInterval(entry.name, instance)` with no confirmation guard; refreshes the list.
    - **Inline validation**: `$effect` per selected interval; 500 ms debounce → `validateTimeInterval(entry)` → `validationErrors: string[]` rendered below the spec editor.
    - **Diff/save flow**: "Preview diff" button calls `exportConfig({ instance, time_intervals: [entry] })`, then `diffConfig`, then switches to diff step with `<YamlDiffViewer>` + save options + `saveConfig`; same pattern as receivers page.
    - **Read-only**: hide all edit, add, delete, and save controls when `!$canEditConfig`.
  - **Files**: `web/src/routes/config/time-intervals/+page.svelte`
  - **Test**: `cd web && npx tsc --noEmit` passes; `npm test` passes; manual smoke test: add a Mon–Fri 09:00–17:00 interval, validate inline (no errors), preview diff, verify YAML contains the new entry.
  - **Developer writes**: No
