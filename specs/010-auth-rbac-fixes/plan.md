# Plan: Auth & RBAC Fixes

**Status**: Approved
**Spec**: specs/010-auth-rbac-fixes/spec.md

## Architecture decisions

### AD-1: Add `role` field to the Login API response

**Decision**: The `POST /api/auth/login` handler returns `role` alongside `token` and `expires_at` in the JSON body.

**Rationale**: The frontend `login()` TypeScript function already declares `role: UserRole` in its return type. The `+page.svelte` login page calls `authStore.setToken(res.token, res.expires_at, res.role)`. Because the Go handler does not currently include `role`, `res.role` resolves to `undefined` at runtime, causing the default `= 'admin'` in `setToken` to silently kick in — every login effectively becomes admin. Adding `role` to the response is the minimal fix with no API contract breakage (additive field).

**Alternatives considered**: Reading the role from `GET /api/auth/status` after login — rejected because it adds a second round-trip and doesn't fix the type-level footgun in `setToken`.

---

### AD-2: Config package owns role validation with a local constant list

**Decision**: `internal/config/config.go` validates `auth.users[N].role` using a local set `validRoles = map[string]bool{"viewer": true, "silencer": true, "config-editor": true, "admin": true}`. No import from `internal/auth`.

**Rationale**: Importing `internal/auth` from `internal/config` would create a dependency between two foundational packages. The four valid role strings are stable and unlikely to change (adding a role is a breaking API change in any case). A comment in `config.go` pointing to `internal/auth/roles.go` is sufficient to keep them in sync.

**Alternatives considered**: Exporting a `ValidRoles()` function from `internal/auth` — rejected as over-engineering for four stable strings.

---

### AD-3: Duplicate password warning uses the existing `[]string` warnings mechanism

**Decision**: Duplicate-password detection runs inside `config.Load()` after `validate()`, appending a warning string to the existing `warnings` slice. The caller (`main.go`) logs it via zap. `validate()` itself is not changed.

**Rationale**: `config.Load()` already returns a `[]string` of non-fatal warnings (used for ignored env vars). Reusing this mechanism avoids adding a logger parameter to the config package, keeping it pure and easily testable.

**Alternatives considered**: Detecting duplicates inside `auth.NewServiceFromConfig()` — rejected because hashed passwords can't be compared; detection must happen before hashing, at config load time.

---

### AD-4: E2E multi-role UX tests in a new browser spec file

**Decision**: Create `e2e/tests/multi-role-login.spec.ts` using Playwright's browser API (`page.goto`, `page.locator`). The existing `global-setup.ts` is extended to include silencer and config-editor users so they are available to all E2E tests.

**Rationale**: The existing `security.spec.ts` is API-only (`APIRequestContext`). Navbar rendering requires a real browser. Keeping them separate avoids breaking the rate-limiter login budget in `security.spec.ts` (currently 3 logins in `beforeAll`). `global-setup.ts` is the single source of truth for the test server config, so adding users there makes them available to both spec files without duplication.

**Alternatives considered**: Adding browser tests directly to `security.spec.ts` — rejected to avoid polluting the focused ADR-005 security coverage and to keep concerns separated.

---

### AD-5: Fix duplicate `lint` job in CI

**Decision**: Remove the first (legacy) `lint` job definition from `.github/workflows/ci.yml`, keeping only the second one (the correct `golangci-lint-action@v6` job that includes the `dist` placeholder step). Ensure `build-and-e2e` lists the correct job names in `needs:`.

**Rationale**: The YAML file currently defines two jobs both named `lint`. GitHub Actions silently uses only the last one, meaning the first is dead YAML. The `build-and-e2e` job's `needs: [test, lint, frontend-test]` already correctly gates E2E on all other jobs — CI wiring is otherwise fine.

**Alternatives considered**: Renaming one job — rejected; the surviving job already has a descriptive name.

---

## Impacted files

| File | Action | Description |
|------|--------|-------------|
| `internal/api/handlers/auth.go` | Modify | Add `role` field to the `Login` handler JSON response |
| `internal/config/config.go` | Modify | Add role string validation in `validate()`; add duplicate password check in `Load()` |
| `internal/config/config_test.go` | Modify | Add table-driven tests for role validation (valid + invalid strings) |
| `internal/api/handlers/auth_test.go` | Modify | Add table-driven multi-role login round-trip tests (all 4 roles, role claim, role-gate enforcement, wrong password) |
| `web/src/lib/stores/auth.ts` | Modify | Remove `= 'admin'` default from `setToken` |
| `web/src/lib/components/layout/Navbar.svelte` | Modify | Change "Admin" → "Sign in"; guard logout with `$isAuthenticated`; add role badge |
| `config.example.yaml` | Modify | Add commented-out `users:` block with all four roles |
| `e2e/global-setup.ts` | Modify | Add silencer and config-editor users to the test config |
| `e2e/tests/multi-role-login.spec.ts` | Create | Playwright browser tests for multi-role login UX (Navbar, role badge, logout) |
| `.github/workflows/ci.yml` | Modify | Remove duplicate `lint` job definition |

---

## Implementation phases

### Phase 1 — Backend correctness

**Goal**: Fix the root cause of the privilege escalation footgun and add config startup validation.

- `internal/api/handlers/auth.go`: include `"role": string(role)` in the `Login` JSON response (decode `role` from the issued JWT via `svc.Login` return value, or re-derive from the `Service`). Since `svc.Login` already returns `(token, exp, err)` but not the role directly, the cleanest approach is to call `svc.Validate(token)` immediately after `svc.Login` to get the role — or better, add `role` as a return value to `Service.Login`. This is a minor signature change scoped to the auth package. Actually, looking at the current `Login` service signature: `func (s *Service) Login(password, totpCode string) (string, time.Time, error)` — we extend it to `(string, Role, time.Time, error)`.
- `internal/config/config.go`: add role validation in `validate()`; add duplicate password check (compare all plaintext passwords, including `AdminPassword` vs `Users[i].Password`) in `Load()` after `validate()`.

**Files**: `auth.go` (handler), `auth.go` (service), `config.go`

---

### Phase 2 — Frontend fixes

**Goal**: Fix the Navbar UX and the TypeScript type footgun.

- `web/src/lib/stores/auth.ts`: remove `= 'admin'` default.
- `web/src/lib/components/layout/Navbar.svelte`: change `{#if $isAdmin}` → `{#if $isAuthenticated}` for logout block; change "Admin" label → "Sign in"; add role badge element showing `$authStore.role` when authenticated.

**Files**: `auth.ts` (store), `Navbar.svelte`

---

### Phase 3 — Documentation

**Goal**: Make multi-role config discoverable for operators.

- `config.example.yaml`: add commented `users:` block with all four roles.

**Files**: `config.example.yaml`

---

### Phase 4 — Backend tests

**Goal**: Verify the full login → JWT → role-gate round-trip for all roles in CI.

- `internal/config/config_test.go`: table-driven tests for valid roles (`viewer`, `silencer`, `config-editor`, `admin`) and invalid strings (`""`, `"read-only"`, `"superuser"`).
- `internal/api/handlers/auth_test.go`: table-driven tests for multi-role login — login each of the 4 roles, check `role` in response, check role-gated endpoints return correct 200/403; wrong password → 401. Uses `auth.NewServiceFromConfig` with inline `config.AuthConfig`.

**Files**: `config_test.go`, `auth_test.go`

---

### Phase 5 — E2E tests + CI fix

**Goal**: Add browser-based multi-role login UX coverage and clean up CI.

- `e2e/global-setup.ts`: add `e2e-silencer-pass` (role: silencer) and `e2e-config-editor-pass` (role: config-editor) to the embedded config YAML.
- `e2e/tests/multi-role-login.spec.ts`: browser tests for each role — log in, verify role badge text, verify logout button present, verify Config nav visible/hidden per role.
- `.github/workflows/ci.yml`: remove duplicate `lint` job.

**Files**: `global-setup.ts`, `multi-role-login.spec.ts`, `ci.yml`

---

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `Service.Login` signature change breaks callers | Low | Only one call site in `handlers/auth.go`; `auth_test.go` uses the service directly — update both during Phase 1 |
| E2E browser tests are flaky (race on page render) | Medium | Use `page.waitForSelector` / `expect(locator).toBeVisible()` with Playwright's built-in retry; test against role-specific `data-testid` attributes or stable text content |
| `global-setup.ts` adding new users breaks existing E2E rate-limit budget | Low | The `security.spec.ts` beforeAll logs in 3 times (within the 5 req/min burst). New users are only used by the new spec file. No rate-limit impact on existing tests |
| Duplicate `lint` job removal breaks `needs:` reference | Low | `build-and-e2e` references `lint` by name — survives because we keep the second (correct) `lint` job |
