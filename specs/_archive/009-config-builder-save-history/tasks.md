# Tasks: Config Builder — Save & History

**Status**: Ready
**Total**: 9 tasks · 3 phases

---

## Phase 1 — History store (backend)

- [x] **T-1.1**: Create `internal/confighistory` package with `SaveRecord` struct and mutex-protected `Store`
  - Files: `internal/confighistory/store.go`
  - `SaveRecord` fields: `SavedAt time.Time`, `Mode string`, `Alertmanager string`, `Actor string`, `CommitSHA string`, `HTMLURL string`, `RawYAML string`
  - `Store` holds `mu sync.RWMutex` and `entries map[string][]SaveRecord` (key = alertmanager instance name)
  - `Append(instance string, r SaveRecord)` — acquires write lock, appends, evicts oldest when `len > 50`
  - `List(instance string) []SaveRecord` — acquires read lock, returns a copy of the slice (newest-first order: reverse before returning)
  - Package-level doc comment: "Package confighistory implements an in-memory, per-instance save history for the Config Builder (feature 009). History resets on process restart."
  - Test: `go test ./internal/confighistory/... -race -count=1`
  - Developer writes: No

- [x] **T-1.2**: Write table-driven tests for `internal/confighistory`
  - Files: `internal/confighistory/store_test.go`
  - Cases: append single record → List returns it; append 51 records → List returns 50 (oldest evicted); List on unknown instance → empty slice; concurrent Append + List passes race detector; newest-first ordering
  - Test: `go test ./internal/confighistory/... -race -count=1`
  - Developer writes: No

---

## Phase 2 — API layer

- [x] **T-2.1**: Extend `ConfigHandler` with history store + add `History` handler
  - Files: `internal/api/handlers/config.go`
  - Add `store *confighistory.Store` field to `ConfigHandler`; update `NewConfigHandler(pool, gh, gl, store)` signature
  - In `Save`: after the successful `writeJSON` at the end, extract the JWT role via `auth.ExtractBearerToken` + `h.svc.Validate` (requires adding `svc *auth.Service` field too, injected via constructor); build a `confighistory.SaveRecord` and call `h.store.Append(req.Alertmanager, record)`
  - Add `History` handler: `GET /api/config/history?instance=<name>` → `writeJSON(w, map[string]any{"history": h.store.List(instance), "alertmanager": instance})`; returns empty array (not null) when no records exist
  - Note: actor extraction follows the same pattern as `auth.ExtractBearerToken` used in middleware — no extra HTTP round-trip
  - Test: `go build ./...` passes; handler covered by T-2.3
  - Developer writes: No

- [x] **T-2.2**: Add `GitopsDefaults` handler to `ConfigHandler`
  - Files: `internal/api/handlers/config.go`
  - Add `GitopsDefaults` handler: `GET /api/config/gitops-defaults` → `writeJSON(w, map[string]any{"github_configured": h.ghPusher != nil, "gitlab_configured": h.glPusher != nil})`
  - No auth.Service needed here — the middleware already enforces config-editor role before the handler runs
  - Test: `go build ./...` passes; handler covered by T-2.3
  - Developer writes: No

- [x] **T-2.3**: Wire new routes and history store in the router
  - Files: `internal/api/router.go`
  - Instantiate `confighistory.NewStore()` inside `NewRouter`
  - Pass the store (and `authSvc`) to the updated `NewConfigHandler`
  - Register under `requireConfigEditor`:
    - `r.With(requireConfigEditor).Get("/config/history", cfgH.History)`
    - `r.With(requireConfigEditor).Get("/config/gitops-defaults", cfgH.GitopsDefaults)`
  - Ensure import of `internal/confighistory` is added
  - Test: `go build ./...` + `go test ./internal/api/... -race -count=1`
  - Developer writes: No

---

## Phase 3 — Frontend

- [x] **T-3.1**: Add types and API client functions
  - Files: `web/src/lib/api/types.ts`, `web/src/lib/api/config.ts`
  - In `types.ts` add:
    ```ts
    export interface SaveRecord {
      saved_at: string;        // RFC 3339
      mode: 'disk' | 'github' | 'gitlab';
      alertmanager: string;
      actor: string;
      commit_sha: string;
      html_url: string;
      raw_yaml: string;
    }
    export interface GitopsDefaults {
      github_configured: boolean;
      gitlab_configured: boolean;
    }
    ```
  - In `config.ts` add:
    ```ts
    export function fetchHistory(instance: string): Promise<{ history: SaveRecord[]; alertmanager: string }> { ... }
    export function fetchGitopsDefaults(): Promise<GitopsDefaults> { ... }
    ```
  - Test: `cd web && npx tsc --noEmit`
  - Developer writes: No

- [x] **T-3.2**: Add shared `configDraftStore` Svelte writable store
  - Files: `web/src/lib/stores/configDraft.ts`
  - Export a writable store: `export const configDraftStore = writable<{ instance: string; rawYaml: string } | null>(null)`
  - The existing builder pages (routing, receivers, time-intervals) should `import { configDraftStore } from '$lib/stores/configDraft'` and call `configDraftStore.set({ instance, rawYaml })` after each successful mutation (the `raw_yaml` field already returned by every builder endpoint)
  - Modify three existing pages to call `configDraftStore.set(...)` after any successful upsert/delete/set-route response
  - Files also modified: `web/src/routes/config/routing/+page.svelte`, `web/src/routes/config/receivers/+page.svelte`, `web/src/routes/config/time-intervals/+page.svelte`
  - Test: `cd web && npx tsc --noEmit` + manual: edit a receiver, navigate to Save & Deploy, verify YAML appears in diff
  - Developer writes: No

- [x] **T-3.3**: Add "Save & Deploy" tab to Config Builder layout
  - Files: `web/src/routes/config/+layout.svelte`
  - Import `Save` icon from `lucide-svelte`
  - Add `{ href: '/config/save', label: 'Save & Deploy', icon: Save }` as the 4th entry in `tabs`
  - Test: `cd web && npx tsc --noEmit` + visual: tab appears in the tab bar
  - Developer writes: No

- [x] **T-3.4**: Create the Save & Deploy page — diff preview and save form
  - Files: `web/src/routes/config/save/+page.svelte`
  - On mount:
    1. Call `fetchGitopsDefaults()` to know which modes to enable
    2. If `$configDraftStore` is non-null, call `diffConfig(instance, rawYaml)` and render `YamlDiffViewer` with the result
    3. If `$configDraftStore` is null, show an info message: "No pending changes. Edit the routing, receivers, or time intervals first."
  - Mode selector: radio/button group for `disk` / `github` / `gitlab`; disabled with `title` tooltip when the corresponding pusher is not configured
  - Conditional form fields:
    - `disk`: single text input for file path (label "Config file path")
    - `github` / `gitlab`: inputs for repo, branch, file path, commit message, author name, author email
  - Save button: disabled when `has_changes === false` or required fields are empty; on click calls `saveConfig(...)` from `$lib/api/config`
  - On success: show a success banner with mode and, for GitOps modes, a link (`<a href={html_url} target="_blank">`) to the commit
  - On error: show an error banner with the message
  - Test: `cd web && npx tsc --noEmit` + manual flow (see T-3.5)
  - Developer writes: No

- [x] **T-3.5**: Add history list to the Save & Deploy page
  - Files: `web/src/routes/config/save/+page.svelte`
  - Below the save form, add a "Save History" section
  - On mount (alongside the diff): call `fetchHistory(instance)` and store results reactively
  - Refresh history after each successful save
  - Render each `SaveRecord` as a row: formatted `saved_at` timestamp, `mode` badge (colour-coded: disk=gray, github=blue, gitlab=orange), `actor` label
  - Each row has an "Expand diff" toggle button; on expand, call `diffConfig(instance, record.raw_yaml)` (lazy — only on first expand) and render a `YamlDiffViewer` inline below the row
  - When history is empty: show "No saves recorded since last restart."
  - Test: `cd web && npm test` (unit) + `cd web && npx tsc --noEmit` + manual: perform a save and verify the new record appears at the top of the list
  - Developer writes: No

---

## Verification checklist (run after all tasks)

```bash
go build ./...
go test ./... -race -count=1
cd web && npx tsc --noEmit
cd web && npm test
```

All AC-1 through AC-13 from the spec must be manually verified before `/review`.
