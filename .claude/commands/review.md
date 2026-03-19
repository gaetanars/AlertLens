# /review [feature-name] — Audit the implementation

Write `specs/NNN-feature-name/review.md` with a clear verdict.

## Auto-detection

If no name is provided: pick the first feature whose tasks are all checked `[x]` but has no `review.md`.

## What to do

1. Read `specs/NNN-feature-name/spec.md`, `plan.md`, and `tasks.md`
2. Run checks in order:

   **Tasks**: all checked? (if not, stop and report)

   **Quality**: run the appropriate commands for the stack:
   - Tests: `go test ./...` / `npm test` / `pytest` / etc.
   - Linter / type checker: `go vet` / `tsc --noEmit` / `eslint` / etc.

   **Acceptance criteria**: verify each AC from the spec, one by one. For each, explain how you verified it.

   **Architecture**: does the implementation follow the architecture decisions from `plan.md`? Do the created files match what was planned?

   **Constitution**: are the project's non-negotiable principles respected?

3. Write `specs/NNN-feature-name/review.md`

## Format of review.md

```markdown
# Review: [Feature name]

**Date**: YYYY-MM-DD
**Verdict**: Ready to merge / Needs fixes

## Tasks

- [x] All tasks checked (N/N)

## Quality

- [x] Tests pass: `[command run]` → [result]
- [x] Clean build: `[command]` → [result]
- [ ] Issue: [description of the problem found]

## Acceptance criteria

- [x] AC-1: verified — [how you verified it]
- [ ] AC-2: not satisfied — [explanation of what's missing]

## Constitution compliance

- [x] [Principle]: compliant
- [ ] [Principle]: issue — [explanation]

## Issues to fix

_(if Needs fixes)_

1. **Issue**: precise description
   **Suggestion**: how to fix it

## Verdict

**Ready to merge** — All criteria satisfied. Run `/ship` to create the PR.

_or_

**Needs fixes** — N issues to address. Fix them then re-run `/review`.
```

## Rules

- Verifying = proving, not assuming. If you can't verify an AC, say so explicitly.
- Don't round off the edges: one unverified AC = "Needs fixes" verdict
- After "Ready to merge": point to the next step (`/ship`)
