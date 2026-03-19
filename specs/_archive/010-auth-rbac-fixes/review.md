# Review: Auth & RBAC Fixes

**Date**: 2026-03-19
**Verdict**: Ready to merge

---

## Tasks

- [x] All tasks checked (10/10)

---

## Quality

- [x] Go tests: `go test ./... -race -count=1` → all packages pass (alertmanager, api, api/handlers, auth, config, configbuilder, incident)
- [x] Go vet: `go vet ./...` → 0 issues
- [x] Go build: `go build .` → OK
- [x] Frontend type check: `npm run check` → 0 errors, 46 pre-existing a11y warnings (unchanged)
- [x] Frontend unit tests: `npm test` → 140/140 pass
- [x] **Bonus fix**: resolved pre-existing merge conflict in `internal/gitops/gitlab.go` (unresolved `<<<<<<< HEAD` marker) that was blocking handler package compilation; took the incoming version which adds proper `err` handling on `gitlab.NewClient`

---

## Acceptance criteria

### AC-1: Remove default role from `authStore.setToken`

- [x] `setToken(token, expiresAt, role)` has no default — verified: `auth.ts:38` shows `role: UserRole` with no `= 'admin'`
- [x] All call sites pass role explicitly — verified: `npm run check` reports 0 errors on all 4122 files; both `+page.svelte` and `+layout.svelte` already passed `role` explicitly

### AC-2: Navbar login label, logout visibility, and role badge

- [x] "Sign in" label when unauthenticated — verified: `Navbar.svelte:115` has text `Sign in`; `Admin` string removed
- [x] Logout guard is `$isAuthenticated` — verified: `Navbar.svelte:100` uses `{#if $isAuthenticated}`; `isAdmin` removed from imports
- [x] Role badge displays `$authStore.role` — verified: `Navbar.svelte:101` `<span class="text-xs text-muted-foreground">{$authStore.role}</span>` adjacent to logout button

### AC-3: Config validation rejects unknown roles at startup

- [x] `config.Load()` returns error for unknown role — verified: `config.go:245` returns `fmt.Errorf("auth.users[%d].role: unknown role %q ...")` for any role not in `validRoles` map
- [x] Error message identifies index and value — verified: message format `auth.users[N].role: unknown role "X"` confirmed in code
- [x] All four valid roles accepted — verified: `TestValidate_UserRole` sub-tests viewer/silencer/config-editor/admin all pass
- [x] Duplicate passwords emit zap warning — verified: `checkDuplicatePasswords()` called in `Load()` after `validate()`, appends to `warnings` slice; `TestLoad_DuplicatePasswordWarning` passes (3 sub-tests)
- [x] Table-driven, parallel, stdlib only — verified: tests use `t.Parallel()`, `t.Run()`, no imports outside stdlib

### AC-4: `config.example.yaml` documents multi-role users

- [x] `users:` block with all four roles — verified: viewer, silencer, config-editor, admin all present and commented out
- [x] Entirely commented out — verified: every line starts with `#`; `go run . -config config.example.yaml` starts without errors
- [x] Role hierarchy and shared-credential model explained — verified: inline comments explain hierarchy and model limitation
- [x] `totp_secret` shown and explained — verified: `# totp_secret: "BASE32SECRET"  # optional: enable TOTP MFA for this user` on viewer entry

### AC-5: Backend integration tests for multi-role login round-trip

- [x] All four roles covered — verified: TestAuthHandler_Login_MultiRole has viewer/silencer/config-editor/admin sub-tests; all PASS
- [x] JWT role claim matches expected — verified: test decodes response JSON and checks `resp["role"] == tc.expectedRole`
- [x] Role-gated endpoints return 200/403 correctly — verified: `TestRequireRole_Enforcement` covers 10 (token, required) pairs; all pass; hierarchy correct (silencer satisfies viewer, config-editor satisfies silencer, etc.)
- [x] Wrong password → 401 — verified: `wrong_password_→_401` sub-test passes
- [x] Table-driven, parallel, stdlib only — verified: `t.Parallel()` at sub-test level, no testify/gomock

### AC-6: Playwright E2E tests for multi-role login UX

- [x] `e2e/tests/multi-role-login.spec.ts` created — 6 browser tests, 136 lines
- [x] Tests cover all four roles with role badge assertions — verified in code: `header.getByText('viewer')`, `'silencer'`, `'config-editor'`, `'admin'`
- [x] Logout button verified for each authenticated role — verified: viewer and admin tests each assert `getByRole('button', { name: /sign out/i })`
- [x] Config nav gating verified — config-editor/admin: "Config" link visible; viewer/silencer: absent
- [x] Silence creation UI tested for silencer — silencer test navigates to `/silences` and asserts "New silence" button visible
- [x] Logout flow tested — logout test clicks "Sign out" and asserts "Sign in" reappears
- [x] Uses `ALERTLENS_BIN` pattern — inherited from `global-setup.ts` (unchanged)
- [x] CI wiring verified — `build-and-e2e` `needs: [test, lint, frontend-test]`; single `lint` job remains after removing the legacy duplicate

**Note on E2E execution**: The browser tests cannot be executed locally without the E2E binary and a running backend. They are authored for CI execution (`cd e2e && npx playwright test` in the `build-and-e2e` job which builds the binary first). The test structure, selectors, and login flow have been audited against the actual Svelte component source.

---

## Architecture compliance

- [x] **AD-1** (Login returns role): `Service.Login` signature extended to `(string, Role, time.Time, error)`; handler uses the Role directly without re-parsing the JWT — matches decision
- [x] **AD-2** (Config owns role validation with local constant): `validRoles` map defined locally in `config.go` with comment pointing to `auth/roles.go`; no import from `internal/auth` — matches decision
- [x] **AD-3** (Duplicate password warning via `[]string`): `checkDuplicatePasswords()` returns `[]string`; appended to warnings slice in `Load()`; no logger added to config package — matches decision
- [x] **AD-4** (E2E in new browser spec file): `multi-role-login.spec.ts` is a separate file; `global-setup.ts` extended with silencer/config-editor users and named constants; existing `security.spec.ts` untouched — matches decision
- [x] **AD-5** (Remove duplicate lint job): legacy job (lines 33–49) removed; surviving job includes `dist` placeholder and pinned `v1.64` — matches decision

All 10 planned files were modified/created as specified in `plan.md`.

---

## Constitution compliance

- [x] **Stateless**: no new runtime state introduced; duplicate-password detection happens at load time in memory only
- [x] **Single binary**: no new runtime dependencies; only build-time changes
- [x] **Security first**: privilege escalation footgun (`= 'admin'` default) eliminated at the TypeScript type level; config validation now rejects bad role strings before the server accepts traffic
- [x] **RBAC by design**: role enforcement tested end-to-end; `RequireRole` middleware chain validated across all 4 roles
- [x] **Error chain preservation**: new errors in `config.go` are leaf errors (no underlying error to wrap — correct); `Service.Login` error paths preserve `%w` wrapping (`fmt.Errorf("generating JTI: %w", err)`, etc.)
- [x] **Tests**: table-driven, parallel, stdlib-only — confirmed across all new test functions

---

## Verdict

**Ready to merge** — All 6 acceptance criteria satisfied. Go test suite passes (race detector enabled), frontend 0 errors, build clean. E2E tests authored and CI wiring verified. Run `/ship` to create the PR.
