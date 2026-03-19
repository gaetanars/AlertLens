# /tasks [feature-name] — Break into atomic tasks

Write `specs/NNN-feature-name/tasks.md` with the ordered list of tasks to implement.

## Auto-detection

If no name is provided: pick the first feature with an approved `plan.md` but no `tasks.md`.

## What to do

1. Read `specs/NNN-feature-name/plan.md` and `spec.md`
2. Break the implementation into atomic tasks:
   - Each task = a unit of work achievable in one Claude session
   - Logical order: dependencies first (types before logic, backend before frontend)
   - Precise on the what: files, functions, expected behaviors
   - Each task must be verifiable (test, observable behavior, clean build)
3. Indicate for each task whether **Developer writes** is Yes or No:
   - **No** (default): Claude implements fully
   - **Yes**: Claude prepares the scaffold and leaves a `TODO(developer)` — the user writes the key business logic
4. Write `specs/NNN-feature-name/tasks.md`

## Format of tasks.md

```markdown
# Tasks: [Feature name]

**Status**: Ready
**Total**: N tasks · P phases

## Phase 1 — [Name: e.g. "Data layer"]

- [ ] **T-1.1**: Precise description of what needs to be done
  - Files: `path/to/file.go`
  - Test: how to verify it's correct
  - Developer writes: No

- [ ] **T-1.2**: Description
  - Files: `path/to/other.go`
  - Test: ...
  - Developer writes: Yes — [clear instruction on what the user will write]

## Phase 2 — [Name: e.g. "API"]

- [ ] **T-2.1**: ...
```

## Rules

- Tasks in dependency order: never depend on an unchecked task
- If a task seems to take more than an hour, break it down
- "Developer writes: Yes" for parts with genuine learning value or business decisions — not for boilerplate
- When tasks are ready: "Ready! Run `/implement` to start T-1.1."
