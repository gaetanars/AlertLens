# /status — Workspace state

Give a complete overview of the project state and recommend the next action.

## What to do

1. Read `specs/roadmap.md` to get the full feature list
2. For each listed feature, check which files exist in `specs/NNN-feature-slug/`:
   - `spec.md` present? Status (Draft / Approved)?
   - `plan.md` present?
   - `tasks.md` present? How many tasks checked vs total?
   - `review.md` present? What is the verdict?
3. Build a status table and determine the recommended next action

## Output format

```
Project state: [project name]
────────────────────────────────────────────
FEATURES

  001 · feature-name          [spec ✓] [plan ✓] [tasks 3/7] [review -]
  002 · another-feature       [spec ✓] [plan -] [tasks -]   [review -]
  003 · third-feature         [ ] not started
  004 · completed-feature     [x] archived

NEXT ACTION
→ /implement feature-name
  (Task T-2.1: description of the next unchecked task)
```

## Recommendation logic

| Detected state | Recommendation |
|---|---|
| No feature started | `/specify` |
| Feature with Draft spec | Open `specs/NNN/spec.md`, resolve open questions, change status → Approved |
| Feature with Approved spec, no plan | `/plan feature-name` |
| Feature with plan, no tasks | `/tasks feature-name` |
| Feature with unchecked tasks | `/implement feature-name` |
| Feature with all tasks checked, no review | `/review feature-name` |
| Feature with review "Ready to merge" | `/ship` |

## Rules

- Features in `specs/_archive/` are shown as `[x] archived`
- If multiple features are in progress simultaneously, list them all and recommend finishing the most advanced one first
- Never invent a state — if a file doesn't exist, its state is `-`
