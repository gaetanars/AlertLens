---
name: quality-gate
description: Run the full AlertLens quality suite (Go build, tests with race detector, golangci-lint, TypeScript type check, frontend unit tests) in the correct order and produce a clear pass/fail report. Use this skill before /review or /ship to catch issues early. Can be scoped to backend-only or frontend-only with an argument.
---

# Quality Gate

Run the complete AlertLens quality suite and report the result clearly. The goal is a single authoritative answer: **green** (ready to proceed) or **red** (here is what to fix).

---

## Scope detection

Check for an optional argument:
- `/quality-gate` → run everything
- `/quality-gate backend` → Go checks only
- `/quality-gate frontend` → frontend checks only

---

## Step 1: Sanity check

Before running anything, verify the working tree is in a sensible state:

```bash
git status --short
```

If there are untracked or modified files that look like they should be staged, report them. Do not abort — just note it.

---

## Step 2: Run the suite

Run each check in sequence. Capture output. **Do not stop on first failure** — run all checks and collect all failures before reporting.

### Backend checks

```bash
# 1. Build — catches compilation errors and missing imports
go build ./...

# 2. Tests with race detector — must pass 100%
go test ./... -race -count=1 -timeout 120s

# 3. Linter — zero warnings policy
golangci-lint run ./...
```

### Frontend checks

```bash
# 4. TypeScript type check — no type errors allowed
cd web && npx tsc --noEmit

# 5. Unit tests
cd web && npm test -- --run
```

---

## Step 3: Report

Use this format:

```
Quality gate — AlertLens
────────────────────────────────────────

  go build ./...            ✓  (or ✗ — N errors)
  go test ./... -race       ✓  (or ✗ — N failures, N s)
  golangci-lint run         ✓  (or ✗ — N issues)
  tsc --noEmit              ✓  (or ✗ — N errors)
  npm test                  ✓  (or ✗ — N failures)

Result: GREEN — all checks pass.
→ Ready for /review.

  — or —

Result: RED — N check(s) failed.
```

For each failure, extract the signal from the noise:

```
✗ go test — 2 failures:

  FAIL internal/auth: TestCSRFMiddleware/valid_token
    auth_test.go:84: expected status 200, got 403

  FAIL internal/configbuilder: TestBuilder/empty_routes
    builder_test.go:112: unexpected nil error

✗ golangci-lint — 3 issues:

  internal/api/handlers/alerts.go:42:
    [govet] printf: fmt.Sprintf format %v has arg err of wrong type
  internal/incident/store.go:67:
    [errcheck] Error return value of Store.Add not checked

✗ tsc — 1 error:

  web/src/lib/api/alerts.ts:23:
    Type 'string | undefined' is not assignable to type 'string'
```

---

## Step 4: Verdict and next action

**GREEN:**
```
Result: GREEN — all 5 checks pass.
→ Ready for /review [feature-name].
```

**RED:**
```
Result: RED — fix these before proceeding:

Priority order:
1. [Most blocking issue — e.g. compilation error]
2. [Test failures]
3. [Lint issues]
4. [Type errors]

Run /quality-gate again after fixing.
```

---

## Rules

- **Never skip a check** — partial green is not green.
- **Extract the meaningful lines** from compiler/test output — do not paste raw walls of text.
- If `golangci-lint` is not installed, report it as a setup issue and note the install command: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`.
- If a test fails due to a missing dev dependency (e.g. Alertmanager not running), say so explicitly — it's a setup issue, not a code issue.
- Race conditions detected by `-race` are always **Critical** — flag them prominently.
- This skill does not fix anything — it only reports. Fixes happen via `/implement` or direct edits.
