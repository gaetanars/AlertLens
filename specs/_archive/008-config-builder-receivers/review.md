# Review: Config Builder — Receivers & Time Intervals

**Date**: 2026-03-27
**Verdict**: Ready to merge

## Tasks

- [x] All tasks checked (9/9)

## Quality

- [x] Go build: `go build ./...` → clean, no errors
- [x] Go tests: `go test ./... -race -count=1` → all packages pass (`internal/api/handlers` 17 s, `internal/configbuilder` 2 s, all others pass)
- [x] TypeScript: `cd web && npx tsc --noEmit` → clean, no errors
- [x] Frontend tests: `npm test` → 140 passed (4 test files)

**Minor fix applied during review**: `upsertTimeInterval` was imported but never called in `time-intervals/+page.svelte` (the save flow uses `exportConfig → diffConfig → saveConfig`, not a direct builder upsert). Removed the unused import before finalising.

## Acceptance criteria

- [x] **AC-1**: Receivers page uses `listReceivers` (→ `GET /api/builder/receivers`), `upsertReceiver` (→ `PUT /api/builder/receivers/{name}`), and `deleteReceiver` (→ `DELETE /api/builder/receivers/{name}`). Verified by grep on `receivers/+page.svelte` imports and call sites.

- [x] **AC-2**: `ReceiverEditor.svelte` handles all five integration types. Each renders only type-specific fields: Webhook (url, max_alerts, send_resolved), Slack (channel, api_url, username, title, text, send_resolved), PagerDuty (routing_key OR service_key, description, send_resolved), Email (to, from, smarthost, auth_username, auth_password, send_resolved), OpsGenie (api_key, message, priority, send_resolved). Verified by reading `ReceiverEditor.svelte` field sets.

- [x] **AC-3**: `ReceiverEditor.svelte` checks `receiver.raw_yaml` first (line 164). When truthy it renders a `<textarea>` bound to the raw YAML string, bypassing all typed form sections. `onUpdate` preserves `raw_yaml` on change. Verified by reading the component.

- [x] **AC-4**: `receivers/+page.svelte` runs `validateReceiver(snap)` inside a `$effect` with a 500 ms `setTimeout` debounce (lines 48–61). Errors stored in `validationErrors` and passed as a prop to `ReceiverEditor`, which renders them as a red alert box. Verified by reading the `$effect` block and `ReceiverEditor`'s validation error section.

- [x] **AC-5**: Delete intent calls `getReceiverRoutes(name, instance)`. If `referenced_by.length > 0`, a `deleteGuard` state object is set and an inline confirmation block renders (lines 265–291), listing each route's matchers and depth with "Delete anyway" / "Cancel" buttons. Immediate delete only proceeds when `referenced_by` is empty. Verified by reading `initiateDelete` and the template guard block.

- [x] **AC-6**: End-to-end Slack save flow is wired: `addReceiver()` creates a blank `ReceiverDef`, `ReceiverEditor` renders the "Add integration" selector → Slack card with all fields. "Save receiver" calls `upsertReceiver`. "Preview diff" calls `exportConfig({ receivers: [editing] })` → `diffConfig` → `YamlDiffViewer` → "Confirm and save" calls `saveConfig`. Cannot smoke-test against a live instance in review, but all code paths are present and correctly typed.

- [x] **AC-7**: Time intervals page uses `listTimeIntervals` (→ `GET /api/builder/time-intervals`) and `deleteTimeInterval` (→ `DELETE /api/builder/time-intervals/{name}`). Diff/save flow calls `exportConfig({ time_intervals: [entry] })` → `diffConfig` → `saveConfig` (→ any save target). Verified by grep on `time-intervals/+page.svelte` imports and call sites.

- [x] **AC-8**: All six field groups are present in the interval editor: time ranges (repeatable `<input type="time">` pairs with +/−), weekdays (7 checkboxes Mon–Sun), days-of-month (text input, comma-separated ranges), months (text input), years (text input), timezone/location (text input). Each is individually optional — empty arrays and empty strings are valid. Verified by reading the spec editor block in `time-intervals/+page.svelte` (lines 283–397).

- [x] **AC-9**: `time-intervals/+page.svelte` runs `validateTimeInterval(entry)` inside a `$effect` with a 500 ms `setTimeout` debounce (lines 45–61). Errors rendered below the spec editor when `editingIdx === i`. Verified by reading the `$effect` block and the validation error template.

- [x] **AC-10**: End-to-end business-hours save: `addInterval()` creates a blank `TimeIntervalEntry`. Editor expands to show time range inputs (09:00–17:00), weekday checkboxes (Mon–Fri), and "Preview diff" → `exportConfig` → `diffConfig` → `saveConfig` flow. All code paths present and typed. Cannot smoke-test against a live instance in review.

- [x] **AC-11**: Both pages import `canEditConfig` from `$lib/stores/auth`. All edit/add/delete/save controls are conditionally rendered with `{#if $canEditConfig}`. `ReceiverEditor` hides form inputs when `readonly={!$canEditConfig}`. The `/config/*` layout (`web/src/routes/config/+layout.svelte`) redirects unauthenticated users to `/login`. Verified by reading both pages and the layout file.

- [x] **AC-12**: `builder.ts` exports all 10 required functions: `listReceivers`, `getReceiver`, `upsertReceiver`, `deleteReceiver`, `validateReceiver`, `listTimeIntervals`, `getTimeInterval`, `upsertTimeInterval`, `deleteTimeInterval`, `validateTimeInterval`. Verified by grep on exported functions.

- [x] **AC-13**: All required interfaces defined in `types.ts`: `ReceiverDef` (line 328), `WebhookConfigDef` (290), `SlackConfigDef` (296), `EmailConfigDef` (305), `PagerdutyConfigDef` (314), `OpsgenieConfigDef` (321), `TimeIntervalEntry` (352), `TimeIntervalDef` (343), `TimeRangeDef` (338). Field names and optionality match Go backend JSON tags. Verified by reading `types.ts`.

## Constitution compliance

- [x] **Stateless**: No new persistent state introduced. Both pages use the in-memory builder API; all mutations are delegated to Alertmanager via the existing save flow. Compliant.

- [x] **Single binary**: Only frontend changes and a Go handler addition. No new runtime dependencies. The `go:embed` path is unaffected. Compliant.

- [x] **Security first / RBAC by design**: The `/config/*` layout enforces authentication redirect. Both pages gate all write controls on `$canEditConfig`. The `ReceiverRoutes` backend endpoint is mounted inside the existing `requireConfigEditor` middleware scope (verified in `router.go` from the T-1.3 task). Compliant.

- [x] **Alertmanager-native**: All mutations flow through the configbuilder, which writes to the Alertmanager config. No shadow data formats. Compliant.

- [x] **Error chain preservation**: New Go handler (`ReceiverRoutes`) and builder methods use `fmt.Errorf("...: %w", err)` wrapping as implemented in T-1.3. Frontend errors go through `toast.error`. Compliant.

- [x] **TypeScript strict mode / no `any`**: `grep -n "as any\|: any"` returns zero matches across all three new/modified frontend files. Compliant.

- [x] **Tests**: Backend covered by `internal/api/handlers/builder_test.go` (added in T-1.4). Frontend tests pass (140/140). Compliant.

## Verdict

**Ready to merge** — All 13 acceptance criteria satisfied, all quality gates pass. Run `/ship` to create the PR.
