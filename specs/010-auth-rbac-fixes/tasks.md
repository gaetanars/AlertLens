# Tasks: Auth & RBAC Fixes

**Status**: Ready
**Total**: 10 tasks · 5 phases

---

## Phase 1 — Backend correctness

- [x] **T-1.1**: Extend `Service.Login` to return the authenticated role
  - Files: `internal/auth/auth.go`, `internal/auth/auth_test.go`
  - Change signature from `Login(password, totpCode string) (string, time.Time, error)` to `(string, Role, time.Time, error)`. Update the single call site in `internal/api/handlers/auth.go` to receive and discard the role for now (will be used in T-1.2). Update existing unit tests for `Login` to handle the new return value.
  - Test: `go test ./internal/auth/... -race -count=1` passes; existing `TestAuthHandler_Login_*` tests compile and pass.
  - Developer writes: No

- [x] **T-1.2**: Add `role` field to `POST /api/auth/login` response
  - Files: `internal/api/handlers/auth.go`
  - In the `Login` handler, use the `Role` returned by `svc.Login` (from T-1.1) and include `"role": string(role)` in the `writeJSON` map alongside `"token"` and `"expires_at"`.
  - Test: `go test ./internal/api/handlers/... -race -count=1` passes. Manual check: `curl -X POST .../api/auth/login -d '{"password":"..."}' | jq .role` returns a non-empty role string.
  - Developer writes: No

- [x] **T-1.3**: Validate `auth.users[N].role` strings at config load time + warn on duplicate passwords
  - Files: `internal/config/config.go`
  - In `validate()`: add a local `validRoles` map; return an error for any `Users[i].Role` not in that set (message: `auth.users[N].role: unknown role "X"`). In `Load()` after `validate()`: collect all plaintext passwords (including `AdminPassword`); if any two are equal, append a warning to the returned `warnings` slice.
  - Test: `go test ./internal/config/... -race -count=1` passes; existing tests still pass.
  - Developer writes: No

---

## Phase 2 — Frontend fixes

- [x] **T-2.1**: Remove default `'admin'` role from `authStore.setToken`
  - Files: `web/src/lib/stores/auth.ts`
  - Remove `= 'admin'` from the `role` parameter of `setToken`. TypeScript will now report a compile error at any call site that omits `role`.
  - Test: `cd web && npm run check` reports no errors (confirms all call sites already pass `role` explicitly). `npm test` passes.
  - Developer writes: No

- [x] **T-2.2**: Fix Navbar — "Sign in" label, logout for all roles, role badge
  - Files: `web/src/lib/components/layout/Navbar.svelte`
  - Three changes:
    1. Change the login link label from `Admin` to `Sign in`.
    2. Change the logout block guard from `{#if $isAdmin}` to `{#if $isAuthenticated}` (import `isAuthenticated` from the auth store).
    3. Inside the logout block, add a role badge element showing `$authStore.role` (e.g. a `<span>` with class `text-xs text-muted-foreground` displaying the role string).
  - Test: `cd web && npm run check` passes. Visual verification: log in as a viewer — logout button and role badge (`viewer`) are visible; "Sign in" appears when unauthenticated.
  - Developer writes: No

---

## Phase 3 — Documentation

- [x] **T-3.1**: Document multi-role `users:` block in `config.example.yaml`
  - Files: `config.example.yaml`
  - Under `auth:`, add a commented-out `users:` block with one entry per role (viewer, silencer, config-editor, admin). Include inline comments explaining the role hierarchy and the shared-credential model. Show `totp_secret` as an optional commented field for the viewer entry.
  - Test: `go run . -config config.example.yaml` must still start without errors (the block is fully commented out).
  - Developer writes: No

---

## Phase 4 — Backend tests

- [x] **T-4.1**: Table-driven tests for config role validation
  - Files: `internal/config/config_test.go`
  - Add a `TestValidate_UserRole` test (table-driven, parallel) covering: all four valid roles → no error; empty string → error; `"read-only"` → error; `"superuser"` → error. Add a `TestLoad_DuplicatePasswordWarning` test that builds a minimal config with two identical passwords and asserts the returned warnings slice contains a duplicate-password warning string.
  - Test: `go test ./internal/config/... -race -count=1 -v` — all new sub-tests pass.
  - Developer writes: No

- [x] **T-4.2**: Table-driven multi-role login round-trip tests in the handlers package
  - Files: `internal/api/handlers/auth_test.go`
  - Add `TestAuthHandler_Login_MultiRole` (table-driven, parallel) that:
    - Builds an `auth.Service` via `auth.NewServiceFromConfig` with one user per role.
    - For each role: POST to `Login` handler → assert 200, assert `role` field in JSON response matches expected role string.
    - Wrong password → assert 401.
  - Add `TestRequireRole_Enforcement` (table-driven, parallel) using a minimal chi router wired with `RequireRole` middleware for each level. For each (token role, required role) pair assert 200 or 403.
  - Test: `go test ./internal/api/handlers/... -race -count=1 -v` passes.
  - Developer writes: No

---

## Phase 5 — E2E tests + CI fix

- [x] **T-5.1**: Add silencer and config-editor users to the E2E global setup
  - Files: `e2e/global-setup.ts`
  - Add two entries to the `CONFIG_YAML` users block:
    - `password: "e2e-silencer-pass"`, `role: "silencer"`
    - `password: "e2e-config-editor-pass"`, `role: "config-editor"`
  - Export the passwords as named constants (`E2E_SILENCER_PASS`, `E2E_CONFIG_EDITOR_PASS`) so the new spec can import them.
  - Test: Existing `security.spec.ts` still passes (`cd e2e && npx playwright test tests/security.spec.ts`). The rate-limiter budget is unaffected (new users are not logged in by `security.spec.ts`).
  - Developer writes: No

- [x] **T-5.2**: Playwright browser tests for multi-role login UX
  - Files: `e2e/tests/multi-role-login.spec.ts`
  - Create a new spec with `workers: 1` (inherits from config). Use `page.goto('/login')` for browser tests. Cover:
    1. **Unauthenticated navbar** — navigate to `/alerts`, expect "Sign in" link visible, "Admin" text absent.
    2. **Viewer login** — fill password, submit, expect: role badge contains `viewer`, logout button visible, Config nav item absent.
    3. **Silencer login** — expect: role badge `silencer`, logout button visible, Config nav absent.
    4. **Config-editor login** — expect: role badge `config-editor`, Config nav item visible.
    5. **Admin login** (using `e2e-admin-pass`) — expect: role badge `admin`, Config nav visible.
    6. **Logout flow** — for viewer: click logout button, expect redirect to unauthenticated state ("Sign in" visible).
  - Each test logs in fresh (no shared token state between browser tests). Use `page.locator` with stable text/role selectors; avoid CSS class selectors.
  - Test: `cd e2e && npx playwright test tests/multi-role-login.spec.ts` passes.
  - Developer writes: No

- [x] **T-5.3**: Remove duplicate `lint` job from CI
  - Files: `.github/workflows/ci.yml`
  - Delete the first `lint` job block (lines 33–48, named "Go linters", using the old `golangci-lint-action` pattern without the `dist` placeholder). Keep the second `lint` job (lines 75–93, named "Go lint", includes `mkdir -p dist`). Verify `build-and-e2e`'s `needs: [test, lint, frontend-test]` still resolves correctly.
  - Test: `cat .github/workflows/ci.yml | grep "^  lint:" | wc -l` outputs `1`. Optionally validate with `yamllint` or `actionlint` if available locally.
  - Developer writes: No
