# /plan [feature-name] — Design the architecture

Write `specs/NNN-feature-name/plan.md` with architecture decisions and implementation phases.

## Auto-detection

If no name is provided: pick the first feature with an Approved `spec.md` but no `plan.md`.

## What to do

1. Read `specs/NNN-feature-name/spec.md` — verify status is "Approved". If Draft, stop and ask the user to approve the spec first.
2. Read `specs/constitution.md` for technical constraints and principles
3. Analyze the relevant existing code: directory structure, patterns used, existing APIs, shared types/interfaces
4. Design the architecture by answering:
   - Which files to create or modify?
   - What important technical decisions need to be made?
   - How to break the implementation into logical phases?
   - What are the risks?
5. Write `specs/NNN-feature-name/plan.md`

## Format of plan.md

```markdown
# Plan: [Feature name]

**Status**: Draft
**Spec**: specs/NNN-feature-name/spec.md

## Architecture decisions

### AD-1: [Decision title]
**Decision**: What we chose to do.
**Rationale**: Why this is the right choice in this context.
**Alternatives considered**: What was rejected and why.

### AD-2: ...

## Impacted files

| File | Action | Description |
|------|--------|-------------|
| `path/to/file.go` | Create | Role description |
| `path/to/other.ts` | Modify | What changes |

## Implementation phases

### Phase 1 — [Name: e.g. "Data layer"]
- Goal: ...
- Files involved: ...

### Phase 2 — [Name: e.g. "API"]
- Goal: ...

### Phase 3 — [Name: e.g. "UI"]
- Goal: ...

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Risk description | High/Medium/Low | How we handle it |
```

## Human checkpoint

Present the architecture decisions for validation. The user challenges the technical choices, validates the phase breakdown. Mark status "Approved" before moving to `/tasks`.
