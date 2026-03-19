# /roadmap — Define the feature backlog

Decompose the product into ordered features and write `specs/roadmap.md`.

## What to do

1. Read `specs/constitution.md` to understand the project's principles, stack, and constraints
2. If a `PRD.md`, `BRIEF.md`, or `README.md` exists at the root, read it too
3. If `specs/roadmap.md` already exists, read it — you're enriching it, not overwriting it
4. Decompose the product into features:
   - Each feature should be independently deliverable (or nearly so)
   - Order by logical dependencies and user value
   - Group into coherent milestones
   - Identify cross-feature dependencies

**Checkpoint**: present the feature table for validation before writing the file. The user can reorder, merge, or split features.

5. Write `specs/roadmap.md` after validation

## Format of specs/roadmap.md

```markdown
# Roadmap — [Project name]

_Last updated: YYYY-MM-DD_

## Milestones

### Milestone 1 — [Name: e.g. "Foundations"]

| ID  | Feature | Description | Status | Depends on |
|-----|---------|-------------|--------|------------|
| 001 | feature-slug | Short, precise description | [ ] | — |
| 002 | another-feature | Short description | [ ] | 001 |

### Milestone 2 — [Name: e.g. "User experience"]

| ID  | Feature | Description | Status | Depends on |
|-----|---------|-------------|--------|------------|
| 003 | third-feature | Description | [ ] | 001 |

## Dependency graph

```
001 → 002 → 004
001 → 003
```

## Legend

- `[ ]` Not started
- `[~]` In progress
- `[x]` Done / Archived
```

## Rules

- 3-digit IDs, sequential (001, 002, ...)
- Slugs in kebab-case, self-explanatory
- Start small and add features rather than listing too many upfront
- One feature = one coherent functional scope, not too broad
