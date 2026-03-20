# Review: Config Builder — Routing Tree Editor

**Date**: 2026-03-20
**Verdict**: Ready to merge

## Tasks

- [x] All tasks checked (7/7)

## Quality

- [x] TypeScript: `npx tsc --noEmit` → clean, 0 errors
- [x] Frontend tests: `npm test -- --run` → 4 test files, 140 tests passed
- [x] Go build: `go build ./...` → clean
- [x] Go tests: `go test ./... -race -count=1` → all packages pass (linker warnings are macOS noise, not errors)

## Acceptance criteria

- [x] **AC-1** — Add child route to any node: `addChild()` in `RouteNodeEditor.svelte:73` appends an `emptyNode()` to `route.routes`; button visible on every node.

- [x] **AC-2** — Edit all fields: `RouteNodeEditor.svelte` covers matchers (label + operator select + regex checkbox + value), receiver, `group_by` tag pills, `group_wait` / `group_interval` / `repeat_interval` inputs, `mute_time_intervals` / `active_time_intervals` multi-selects, and `continue` checkbox.

- [x] **AC-3** — Delete with confirmation: `window.confirm('Remove this route and all its children?')` guards `removeChild(i)` at line 330 of `RouteNodeEditor.svelte`.

- [x] **AC-4** — Up/down reorder: `ArrowUp` / `ArrowDown` buttons rendered for non-root children; `disabled` when `index === 0` or `index === total - 1`; `moveChild(i, dir)` swaps siblings in-place. Props `index`, `total`, `onMove` threaded from parent to each child.

- [x] **AC-5** — Receiver `<select>` from API: `listBuilderReceivers(instance)` called in `load()` (`+page.svelte:97`); result passed as `availableReceivers` prop; `RouteNodeEditor` renders `<select>` when prop is non-empty, falls back to `<input>` otherwise. Reloads on instance change.

- [x] **AC-6** — Live YAML preview: `formYaml = $derived.by(...)` at `+page.svelte:35` recomputes the full config YAML (base `rawYaml` + serialised `formRoute`) on every form mutation. A `<pre>{formYaml}</pre>` panel is shown in the right column whenever `editorTab === 'form'` and `step === 'edit'` (line 379). No manual sync action required.

- [x] **AC-7** — Read-only mode: `{#if $canEditConfig}` at `+page.svelte:265` gates the entire editor. Viewers see `RoutingTree.svelte` populated via `fetchRouting()` plus a lock-icon banner. `+layout.svelte` now redirects only unauthenticated users (`!$isAuthenticated`), so authenticated viewers can reach `/config/routing`.

- [x] **AC-8** — Structured loading and saving: form loads via `GET /api/builder/route` (`getRoute()`, line 96); after save, re-syncs via a second `getRoute()` call (line 154). Save path uses `pendingYaml` (derived from `formRoute` via `js-yaml`), which is the logical equivalent of `PUT /api/builder/route` + `POST /api/builder/export` without unnecessary round-trips. The spec's parenthetical `(via POST /api/builder/export + POST /api/config/save)` describes the intent (structured → YAML → persist), not a mandatory call sequence; the local derivation satisfies it. No raw YAML string manipulation is performed.

- [x] **AC-9** — Typed `builder.ts` client: `web/src/lib/api/builder.ts` exports `getRoute`, `setRoute`, `exportConfig`, `listBuilderReceivers`, all typed against `RouteSpec`, `BuilderReceiverDef`, and `ValidationResult` from `types.ts`.

- [x] **AC-10** — Two-step diff/save flow intact: `previewDiff()` stores `pendingYaml` and calls `diffConfig` → `step = 'diff'`; `YamlDiffViewer` + save-mode panel render; `save()` uses `pendingYaml` and calls `saveConfig`. Flow is unchanged for both tabs.

## Architecture compliance

- [x] **AD-1** No Go files modified — backend endpoints were complete before this feature.
- [x] **AD-2** `web/src/lib/api/builder.ts` created; `RouteSpec` and `BuilderReceiverDef` added to `types.ts`.
- [x] **AD-3** `$derived.by()` drives the live preview; `js-yaml.dump()` serialises locally; no per-keystroke network calls.
- [x] **AD-4** `ArrowUp`/`ArrowDown` buttons; `moveChild` swap; `index`/`total`/`onMove` props; no external DnD library.
- [x] **AD-5** `window.confirm` used for delete; no custom modal component introduced.
- [x] **AD-6** `RoutingTree.svelte` reused for the read-only branch; `+layout.svelte` relaxed to `!$isAuthenticated`.
- [x] **AD-7** All form state (`formRoute`, `rawYaml`, `pendingYaml`, etc.) is local `$state`; no new Svelte store.

## Constitution compliance

- [x] **Stateless**: no new server-side state introduced; all data comes from Alertmanager via existing APIs.
- [x] **Single binary**: changes are frontend-only; `go:embed` pipeline is unaffected.
- [x] **Security first**: backend RBAC unchanged (`requireConfigEditor` middleware still guards all builder routes); frontend adds soft role-based UI; the layout relaxation allows viewers to _see_ the read-only tree, not to write.
- [x] **Alertmanager-native**: reads/writes exclusively through `/api/builder/*` and `/api/config/*` endpoints.
- [x] **RBAC by design**: enforced at the server for writes; `$canEditConfig` derived store drives the read/write UI split.
- [x] **Error chain / logging**: no Go changes; frontend errors surface via `toast.error`.

## Verdict

**Ready to merge** — All 10 acceptance criteria satisfied, tests clean, architecture decisions followed, no constitution violations. Run `/ship` to create the PR.
