# Plan: Config Builder — Save & History

**Status**: Approved
**Spec**: specs/009-config-builder-save-history/spec.md

## Open questions resolved

| Q# | Decision |
|----|----------|
| Q1 — Save placement | **4th tab** ("Save & Deploy") in the Config Builder tab bar, consistent with Routing / Time Intervals / Receivers. |
| Q2 — Actor label | **Role string** from the JWT (`config-editor`, `admin`). No `display_name` field introduced — this is a session-scoped log, not an audit trail. |
| Q3 — History size | **Capped at 50 entries per Alertmanager instance**. Oldest entry evicted on overflow. The `raw_yaml` stored per record is typically < 10 KB; 50 × 10 KB = ~500 KB worst-case per instance, acceptable for in-memory session data. |
| Q4 — GitOps pre-fill | **New `GET /api/config/gitops-defaults`** endpoint returning `{github_configured, gitlab_configured}`. The static config only stores tokens, not repo/branch/file, so only pusher availability is exposed. Fields remain empty and the user fills them at save time. |

---

## Architecture decisions

### AD-1: New `internal/confighistory` package

**Decision**: Introduce `internal/confighistory` as an independent package with a `Store` type — mirroring the `internal/incident` package pattern.

**Rationale**: Keeps the history concern self-contained and testable in isolation. The handler layer depends on the store through a concrete type (not an interface), consistent with how `incident.Store` is used. No interface is added since there is no alternate implementation in scope.

**Alternatives considered**: Embedding the history store directly in `ConfigHandler` — rejected because it would make the handler impossible to test without instantiating a full handler, and it blurs the package's single responsibility.

---

### AD-2: Actor extracted in the handler, not in the store

**Decision**: The `Save` handler extracts the role from the JWT (`auth.ExtractBearerToken` + `svc.Validate`) and passes it as `Actor` when appending to the history store.

**Rationale**: The store should be a dumb data structure with no auth dependency. The handler already has access to the auth service via middleware-populated context (or direct token extraction) and is the correct place to resolve identity.

**Alternatives considered**: Passing the `*auth.Service` into the store — rejected, circular dependency risk and wrong separation of concerns.

---

### AD-3: History read endpoint requires `config-editor`, not `viewer`

**Decision**: `GET /api/config/history` requires `config-editor` role, consistent with AC-8 and AC-12.

**Rationale**: The history records contain `raw_yaml` snapshots of the full Alertmanager config. Exposing them to viewers would leak the full config to roles that can only see the routing tree (which is already read-only). The spec is explicit on this point.

**Alternatives considered**: Returning a history list without `raw_yaml` for viewers — rejected because the spec scopes history to `config-editor` and the diff-on-expand feature requires the stored YAML.

---

### AD-4: `ConfigHandler` receives the history store via constructor injection

**Decision**: Extend `NewConfigHandler` to accept a `*confighistory.Store`. The router creates one store per process and passes it in.

**Rationale**: Matches how `NewIncidentsHandler` receives an `*incident.Store`. No global state.

**Alternatives considered**: A package-level store singleton — rejected, untestable and violates the stateless-per-process philosophy.

---

### AD-5: GitOps defaults endpoint returns availability flags only

**Decision**: `GET /api/config/gitops-defaults` returns `{github_configured: bool, gitlab_configured: bool}`. No repo/branch/file defaults are exposed because the static config does not store them.

**Rationale**: The frontend needs to know which save modes to enable (AC-4). Token presence is the only signal available from the static config. This is a thin, stable contract.

**Alternatives considered**: Omitting the endpoint and relying on the save error response to signal a misconfigured mode — rejected because it gives poor UX (the user would fill the form before learning the mode is unavailable).

---

## Impacted files

| File | Action | Description |
|------|--------|-------------|
| `internal/confighistory/store.go` | Create | `SaveRecord` struct + `Store` (mutex-protected, 50-entry cap per instance) |
| `internal/confighistory/store_test.go` | Create | Table-driven tests: append, cap eviction, list ordering, concurrent safety |
| `internal/api/handlers/config.go` | Modify | Inject `*confighistory.Store`; `Save` appends record; add `History` and `GitopsDefaults` handlers |
| `internal/api/router.go` | Modify | Instantiate `confighistory.Store`; pass to `NewConfigHandler`; register two new routes |
| `web/src/routes/config/+layout.svelte` | Modify | Add "Save & Deploy" tab (Save icon, `/config/save` href) |
| `web/src/routes/config/save/+page.svelte` | Create | Diff preview + save mode selector + save form + history list with expand-to-diff |
| `web/src/lib/api/config.ts` | Modify | Add `fetchHistory()`, `fetchGitopsDefaults()` |
| `web/src/lib/api/types.ts` | Modify | Add `SaveRecord` and `GitopsDefaults` types |

---

## Implementation phases

### Phase 1 — Backend: history store

- **Goal**: Implement `internal/confighistory` with a mutex-protected, per-instance capped store.
- **Files**: `internal/confighistory/store.go`, `internal/confighistory/store_test.go`
- **Deliverable**: `Store.Append(instanceName, SaveRecord)` and `Store.List(instanceName) []SaveRecord` pass all table-driven tests including the 50-entry eviction and race-detector clean run.

### Phase 2 — Backend: API wiring

- **Goal**: Extend `ConfigHandler` with the history store, wire new routes.
- **Files**: `internal/api/handlers/config.go`, `internal/api/router.go`
- **Changes**:
  - `NewConfigHandler` gains `*confighistory.Store` parameter.
  - `Save` handler: after a successful save, build a `confighistory.SaveRecord` (extract actor from JWT, capture `raw_yaml`, `mode`, `commit_sha`, `html_url`, `saved_at`) and call `store.Append`.
  - Add `History` handler: `GET /api/config/history?instance=<name>` → `store.List(instance)` as JSON array.
  - Add `GitopsDefaults` handler: `GET /api/config/gitops-defaults` → `{github_configured: bool, gitlab_configured: bool}` based on pusher nil-checks.
  - Router: register both new routes under `requireConfigEditor`.

### Phase 3 — Frontend: Save & Deploy page

- **Goal**: Add the Save & Deploy tab and its full page: diff viewer, mode selector, form, and history list.
- **Files**: `web/src/routes/config/+layout.svelte`, `web/src/routes/config/save/+page.svelte`, `web/src/lib/api/config.ts`, `web/src/lib/api/types.ts`
- **Page behaviour**:
  1. On mount: call `GET /api/config/gitops-defaults` to know which modes to enable; call `POST /api/config/diff` with the proposed YAML (passed via a shared store or URL state) and render `YamlDiffViewer`.
  2. Mode selector: three options (disk / github / gitlab); disabled with tooltip if pusher not configured.
  3. Conditional form fields: disk → file path; github/gitlab → repo, branch, file path, commit message, author name, author email.
  4. Save button: disabled when `has_changes` is false or form invalid; calls `POST /api/config/save`; on success shows confirmation with optional commit link.
  5. History section below the form: calls `GET /api/config/history?instance=<name>`; renders list with timestamp, mode badge, actor; each row has an expand button that calls `POST /api/config/diff` with the record's `raw_yaml` and renders a `YamlDiffViewer` inline.

---

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| The proposed YAML to diff is not easily available on the Save page (it's built up across three builder tabs) | Medium | The frontend config-builder pages already hold `raw_yaml` in local state; we introduce a lightweight Svelte writable store (`configDraftStore`) shared across the `/config/*` route group so the Save page can read the last assembled YAML. |
| Race condition in `Store.Append` under concurrent saves | Low | `sync.Mutex` (write lock) on every append; `sync.RWMutex` for reads — same pattern as `incident.Store`. |
| The 50-entry cap causes a save record to be evicted before a user reviews it | Low | Cap is per-instance; typical deployments make far fewer than 50 config saves per restart cycle. |
| `auth.ExtractBearerToken` + `svc.Validate` in the Save handler adds a second token parse per request | Low | Negligible CPU cost; the middleware already validated the token. Alternative would be storing the role in context — acceptable future refactor but not needed here. |
