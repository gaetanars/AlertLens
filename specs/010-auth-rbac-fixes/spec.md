# Spec: Auth & RBAC Fixes

**Status**: Approved
**Feature ID**: 010
**Depends on**: 006 (security-foundation)

## Context

The multi-role RBAC system was fully implemented in the backend (006), but several rough edges remain that affect correctness, operator experience, and test coverage:

1. **Default role footgun** — `authStore.setToken` defaults the `role` parameter to `'admin'`, meaning a future caller that omits the argument silently escalates privileges.
2. **Login UX mislabelling** — The Navbar login link reads "Admin" and the logout button is guarded by `$isAdmin`, so non-admin authenticated users (viewer, silencer, config-editor) cannot see the logout button.
3. **Config validation gap** — `config.Load()` does not reject unknown `role` strings in the `users:` block; a typo silently creates a dead credential.
4. **Missing operator documentation** — `config.example.yaml` has no `users:` section, making multi-role setup invisible to new operators.
5. **No backend integration tests for multi-role** — There are no tests that exercise the full login → JWT → role-gated endpoint round-trip for non-admin roles.
6. **No E2E tests for multi-role login UX** — Navbar copy, logout visibility, and role-based nav are untested end-to-end.

## User stories

- As an operator, I want `config.Load()` to fail at startup with a clear error if a user's role string is unrecognised, so misconfigured credentials are caught before the server serves traffic.
- As an operator, I want a `config.example.yaml` with a commented-out `users:` block showing all four roles, so I know how to add per-role credentials.
- As a viewer/silencer/config-editor user, I want the logout button to be visible after I sign in, so I can end my session from the UI.
- As any user, I want the navbar login link to read "Sign in" (not "Admin"), so I understand it is for all roles.
- As a developer, I want a TypeScript error if `setToken` is called without an explicit `role` argument, so the 'admin' default can never silently escalate privileges.
- As a developer, I want backend integration tests covering every role's login and role-gate enforcement, so regressions are caught by CI.
- As a developer, I want Playwright E2E tests for multi-role login UX, so navbar and logout regressions are caught automatically.

## Acceptance criteria

### AC-1: Remove default role from `authStore.setToken`

- [ ] `setToken(token, expiresAt, role)` has no default for `role`; TypeScript reports an error if `role` is omitted at any call site.
- [ ] All existing call sites pass `role` explicitly; `npm run build` and `npm test` pass without changes to call sites.

### AC-2: Navbar login label, logout visibility, and role badge

- [ ] When unauthenticated and auth is enabled, the Navbar shows "Sign in" (not "Admin").
- [ ] The logout button is visible for any authenticated user (`$isAuthenticated`), not just `$isAdmin`.
- [ ] The Navbar displays a role badge showing the current user's role (e.g. `viewer ·`) next to the logout button, derived from `$authStore.role`.

### AC-3: Config validation rejects unknown roles at startup

- [ ] `config.Load()` returns an error when `auth.users[N].role` is not one of: `viewer`, `silencer`, `config-editor`, `admin`.
- [ ] The error message identifies the index and the invalid value (e.g., `auth.users[1].role: unknown role "read-only"`).
- [ ] `config.Load()` continues to accept all four valid role strings without error.
- [ ] If two or more users share the same plaintext password, `config.Load()` emits a structured zap warning (does not fail startup).
- [ ] Table-driven tests cover all four valid roles and at least three invalid strings; tests are parallel and use stdlib `testing` only.

### AC-4: `config.example.yaml` documents multi-role users

- [ ] `config.example.yaml` contains a `users:` block under `auth:` with one commented-out entry per role (viewer, silencer, config-editor, admin).
- [ ] Each entry is commented out so the example does not change default behaviour.
- [ ] Comments explain the role hierarchy and the shared-credential model limitation.
- [ ] The optional `totp_secret` field is shown and explained for at least one entry.

### AC-5: Backend integration tests for multi-role login round-trip

- [ ] Tests cover all four roles: viewer, silencer, config-editor, admin.
- [ ] For each role: login returns HTTP 200 and a JWT with the correct `role` claim.
- [ ] Role-gated endpoints return 200 for the token's role and all lower roles; 403 for roles above the token.
- [ ] Wrong-password login returns 401.
- [ ] Tests are table-driven, parallel, and use stdlib `testing` only; `go test ./... -race -count=1` passes.

### AC-6: Playwright E2E tests for multi-role login UX

- [ ] Navbar shows "Sign in" (not "Admin") when unauthenticated.
- [ ] Logging in as viewer: role badge shows `viewer`, logout button visible, config nav item not visible.
- [ ] Logging in as silencer: role badge shows `silencer`, logout button visible, silence creation UI visible, config nav item not visible.
- [ ] Logging in as config-editor: role badge shows `config-editor`, config nav item visible.
- [ ] Logging in as admin: role badge shows `admin`, all nav items visible.
- [ ] Logout is accessible (button present and functional) for all four roles.
- [ ] Tests use the `ALERTLENS_BIN` pattern from existing E2E tests; `cd e2e && npx playwright test` passes.
- [ ] The CI workflow guarantees the binary is built before the E2E job runs (explicit `needs:` dependency verified or added).

## Out of scope

- Per-user identity / audit trail (shared-credential model is intentional).
- JWT secret rotation on password change (tracked separately as feature 025).
- SSO / OIDC integration (feature 012).
- Any new RBAC roles beyond the existing four.
- MFA (TOTP) changes — the MFA flow is out of scope for this fix batch.

## Open questions

_All questions resolved — spec ready for approval._

- [x] Q1: Role badge is **required** (not optional). Included in AC-2 and AC-6.
- [x] Q2: E2E CI wiring must be verified/fixed as part of this feature. Included as an AC-6 criterion.
- [x] Q3: Duplicate password detection emits a **zap warning** (no startup failure). Included in AC-3.
