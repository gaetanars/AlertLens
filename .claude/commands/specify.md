# /specify [feature-name] — Write the spec

Write `specs/NNN-feature-name/spec.md` for a feature.

## Auto-detection

If no feature name is provided:
- Read `specs/roadmap.md`
- Pick the first feature with status `[ ]` whose dependencies are all `[x]`
- Ask for confirmation: "Next feature from roadmap: **feature-name**. Shall we specify it?"

## What to do

1. Read `specs/constitution.md` and `specs/roadmap.md`
2. Look up the feature's `Issues` column in the roadmap — note every `#NNN` linked to this feature
3. For each linked issue, fetch its title and body with `gh issue view NNN` to extract context, acceptance criteria hints, and scope notes
4. Analyze the relevant parts of the existing codebase for this feature
5. Write `specs/NNN-feature-name/spec.md`
6. List open questions at the end of the file

## Format of spec.md

```markdown
# Spec: [Human-readable feature name]

**Status**: Draft
**Feature ID**: NNN
**Depends on**: NNN (or —)
**GitHub issues**: #NNN, #NNN (or —)

## Context

Why this feature exists, what problem it solves for which user.

## User stories

- As a [role], I want to [action] so that [benefit]
- ...

## Acceptance criteria

- [ ] AC-1: precise and verifiable description (not vague)
- [ ] AC-2: ...

## Out of scope

What explicitly does not belong to this spec.

## Open questions

- [ ] Q1: question to resolve before implementing — [context]
- [ ] Q2: ...
```

## Human checkpoint

After writing the file:
- Present the open questions and wait for answers
- The user must change the status from `Draft` to `Approved` before moving to `/plan`

A wrong spec produces wrong code. Time spent here is never wasted.

## Rules

- Always fetch and read the linked GitHub issues — they contain decisions already made and scope agreed upon
- Acceptance criteria must be concretely verifiable (observable behavior, not "works well")
- The spec describes the **what**, not the **how** — no implementation decisions here
- One spec = one coherent feature. If the spec grows too large, propose splitting it
- If an issue contains an explicit acceptance criterion or out-of-scope note, carry it into the spec verbatim
