# Review: Config Builder — Save & History

**Date**: 2026-03-28
**Verdict**: Ready to merge

---

## Tasks

- [x] All tasks checked (9/9)

---

## Quality

- [x] Tests pass: `go test ./... -race -count=1` → all 8 packages pass, race-detector clean
- [x] Lint clean: `golangci-lint run ./...` → 0 issues
- [x] TypeScript: `cd web && npx tsc --noEmit` → no errors
- [x] Frontend tests: `cd web && npm test` → 140/140 pass

---

## Acceptance criteria

- [x] **AC-1**: "Save & Deploy" tab present in Config Builder layout. Verified in `web/src/routes/config/+layout.svelte:19` — fourth tab with `href='/config/save'` and `Save` icon.

- [x] **AC-2**: Save panel fetches diff via `POST /api/config/diff` and renders with `YamlDiffViewer`. Verified in `web/src/routes/config/save/+page.svelte:72` — `loadDiff()` calls `diffConfig()`, result fed into `<YamlDiffViewer>` at line 212.

- [x] **AC-3**: Save button disabled and "No changes" message shown when `has_changes` is false. Verified: `canSave` derived value requires `!!diffResult?.has_changes` (line 49); "No changes to save." message rendered at line 294–296.

- [x] **AC-4**: Mode selector shows disk/github/gitlab; unconfigured modes disabled with tooltip. Verified: `isModeDisabled()` (line 130) checks `gitopsDefaults`; `modeTooltip()` (line 137) provides the tooltip text; `title={tip}` + `disabled={disabled}` applied on each mode label (lines 228, 234).

- [x] **AC-5**: For `disk` mode, file path field shown and pre-filled from `config_file_path` when set. Fixed: `configFilePath string` added to `alertmanager.Client`, populated in `NewClient`, exposed via `ConfigFilePath()` accessor. `GET /api/config/gitops-defaults?instance=<name>` now returns `disk_file_path`. Frontend `onMount` pre-fills `diskPath` from `d.disk_file_path` if the field is currently empty.

- [x] **AC-6**: GitHub/GitLab fields shown (repo, branch, file path, commit message, author name, author email). No pre-fill needed — plan AD-5 explicitly resolved that the static config stores only tokens, not repo/branch/file, so fields start empty by design. All six fields verified in `+page.svelte` lines 260–279.

- [x] **AC-7**: Successful save shows confirmation with mode and clickable commit link. Verified: success banner at lines 300–321 shows mode, truncated commit SHA, and `<a href={saveResult.html_url} target="_blank">View commit ↗</a>`.

- [x] **AC-8**: `GET /api/config/history?instance=<name>` returns save records; requires `config-editor` role. Verified: `router.go:178` registers `r.With(requireConfigEditor).Get("/config/history", cfgH.History)`; handler at `config.go:265` calls `h.store.List(instance)`.

- [x] **AC-9**: Each save record appended to in-memory history store with all required fields. Verified: `config.go:214–221` — `h.store.Append()` called with `SavedAt`, `Mode`, `Alertmanager`, `Actor` (from `auth.GetRole(r)`), `CommitSHA`, `HTMLURL`, `RawYAML`. Actor uses role from middleware context, not a second JWT parse.

- [x] **AC-10**: History rows have expand button calling `POST /api/config/diff` with the saved `raw_yaml`; records store `raw_yaml`. Verified: `SaveRecord.RawYAML` stored on append; `toggleExpandDiff()` at line 146 calls `diffConfig(record.alertmanager, record.raw_yaml)`; diff rendered inline with `YamlDiffViewer`.

- [x] **AC-11**: History is in-memory only, resets on restart. Verified: `confighistory` package has no persistence — `Store` is an in-memory `map`; `NewStore()` creates a fresh store on each `NewRouter()` call.

- [x] **AC-12**: Save action requires `config-editor`; History endpoint requires `config-editor`. Verified: both in `router.go:177–178` via `requireConfigEditor`; `GitopsDefaults` also behind `requireConfigEditor` at line 179. Frontend additionally guards the save form and button behind `$canEditConfig`.

- [x] **AC-13**: History store safe for concurrent access. Verified: `Store` uses `sync.RWMutex` — write lock in `Append()` (line 43), read lock in `List()` (line 57). Race-detector test `concurrent_append_and_list_are_race-free` passes.

---

## Architecture compliance

| AD | Planned | Status |
|----|---------|--------|
| AD-1: `internal/confighistory` package | Created as planned | ✓ |
| AD-2: Actor extracted in handler via `auth.GetRole(r)` | Implemented — no `*auth.Service` on handler | ✓ |
| AD-3: History requires `config-editor` | Both endpoints gated | ✓ |
| AD-4: Constructor injection `NewConfigHandler(..., store)` | Matches plan | ✓ |
| AD-5: `GET /api/config/gitops-defaults` with availability flags only | Implemented | ✓ |
| Plan: `configDraftStore` writable store | Created; all 3 builder pages write to it | ✓ |

All planned files were created or modified as specified. T-3.4 and T-3.5 were merged into a single file pass (the history section is part of the same page) — no divergence from intent.

---

## Product docs

- [x] `docs/features/config-builder.md` updated during T-2.3 with a new "Save & Deploy" section covering: diff preview, disk/GitHub/GitLab save modes, save history behaviour, and the 50-entry in-memory cap. Content is accurate.

---

## Constitution compliance

- [x] **Stateless**: history store is session-scoped in-memory only, resets on restart. No database, no file I/O.
- [x] **Single binary**: no new runtime dependencies introduced.
- [x] **Security first**: CSRF/RBAC enforced — both new endpoints are behind `requireConfigEditor`. Frontend disables save button when `!$canEditConfig`.
- [x] **RBAC by design**: role enforced in middleware, not in handlers. `auth.GetRole(r)` reads the already-validated role from context.
- [x] **Error chain preservation**: new backend code uses `fmt.Errorf` where errors are wrapped (existing patterns in the file).
- [x] **Tests**: table-driven, parallel, stdlib only — 7 sub-cases in `confighistory/store_test.go`.

---

## Verdict

**Ready to merge** — All 13 ACs satisfied. Run `/ship` to create the PR.
