---
name: alertlens-conventions
description: Review code written for the AlertLens project against its specific conventions: zap structured logging, %w error wrapping, table-driven parallel tests, chi handler patterns, RBAC middleware usage, and the stateless constraint. Use this skill after implementing a task or before /review, to catch AlertLens-specific issues that a generic review would miss.
---

# AlertLens Conventions Review

You are performing a focused code review against AlertLens-specific conventions. This is not a generic review — you are checking for patterns that are project-specific and cannot be inferred from static analysis alone.

Read `specs/constitution.md` and `CLAUDE.md` before starting if they haven't been read in this session.

---

## What to review

The user may pass a list of files, a feature name, or nothing. If nothing is specified, review all files modified since the last commit:

```bash
git diff --name-only HEAD
```

Read each modified `.go` and `.svelte`/`.ts` file in full before starting the review.

---

## Go conventions checklist

### Logging (zap)

- [ ] **No string interpolation in log messages.** Every dynamic value must be a zap field.
  ```go
  // WRONG
  logger.Error(fmt.Sprintf("failed to fetch %s: %v", name, err))
  // CORRECT
  logger.Error("failed to fetch alerts", zap.String("instance", name), zap.Error(err))
  ```
- [ ] Logger is injected via constructor — never `zap.L()` or a package-level global.

### Error handling

- [ ] **All wrapped errors use `%w`**, never bare `errors.New` for a wrapped error.
  ```go
  // WRONG
  return errors.New("failed to load config")
  // CORRECT
  return fmt.Errorf("config: load file: %w", err)
  ```
- [ ] Error messages follow the `package: operation: detail` prefix convention (e.g. `"alertmanager: fetch alerts: %w"`).
- [ ] Errors are not swallowed silently — every `err != nil` is either returned or logged.

### HTTP handlers

- [ ] Handler constructor injects dependencies (pool, store, logger) — no package-level state.
- [ ] `resolveClient(pool, w, instance)` is used when resolving an AM client — never a direct map lookup.
- [ ] Responses use `writeJSON`, `writeJSONStatus`, or `writeError` — never `json.NewEncoder(w).Encode(...)` directly.
- [ ] Every write endpoint (POST/PUT/DELETE/PATCH) declares its minimum required role via middleware, not inside the handler body.

### RBAC

- [ ] Route registration in the router assigns middleware (`RequireViewer`, `RequireSilencer`, `RequireConfigEditor`, `RequireAdmin`) — role checks are never inside handler logic.
- [ ] No hardcoded role strings — use the constants from `internal/auth/roles.go`.

### Tests

- [ ] Tests are **table-driven** with a `cases []struct{ name string; ... }` slice.
- [ ] Every `TestXxx` function calls `t.Parallel()` at the top.
- [ ] Every sub-test inside `t.Run` calls `t.Parallel()` at the top.
- [ ] **No third-party libraries** — stdlib `testing` only. No `testify`, no `gomock`, no `require`.
- [ ] Assertions use `t.Errorf` (non-fatal) or `t.Fatalf` (fatal) — not `t.Error` followed by manual returns.
- [ ] Test file lives alongside the package (`_test.go`). White-box helpers go in `export_test.go`.

### Stateless constraint

- [ ] No new package-level `var` that holds mutable runtime state.
- [ ] No `init()` functions that modify shared state.
- [ ] No direct filesystem writes at runtime (config pushes go through the GitOps layer).

---

## Frontend conventions checklist (TypeScript / Svelte)

- [ ] No `any` type without a `// eslint-disable` comment explaining why.
- [ ] API calls go through `$lib/api/` typed clients — no raw `fetch` in components.
- [ ] Stores use the established pattern in `$lib/stores/` — no ad-hoc writable stores in components.
- [ ] Error states are surfaced to the user — no silent catch blocks.

---

## Output format

For each file reviewed:

```
File: internal/foo/bar.go
  ✓ Logging: structured fields throughout
  ✗ Error wrapping: line 42 — bare errors.New("failed to parse") should use fmt.Errorf("foo: parse: %w", err)
  ✓ Handler pattern: uses writeJSON / writeError correctly
  ✓ Tests: table-driven, parallel
```

End with a summary:

```
Convention review complete
─────────────────────────
Files reviewed: N
Issues found: N
  Critical (must fix before /review): list
  Minor (recommended): list

→ Fix the N critical issues, then run /quality-gate to validate.
```

If no issues found:

```
Convention review complete — all clear.
→ Ready for /quality-gate then /review.
```

---

## Rules

- **Never suggest refactors beyond the conventions listed here.** This is not a general code review.
- Flag issues as **Critical** (violates a non-negotiable from `constitution.md`) or **Minor** (style preference).
- If a convention cannot be verified from the code alone (e.g. runtime behavior), say so explicitly.
- Do not modify files during this review — only report. Fixes happen in a subsequent `/implement` task or direct edit.
