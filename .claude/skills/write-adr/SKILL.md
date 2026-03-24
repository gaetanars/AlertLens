---
name: write-adr
description: Write an Architecture Decision Record (ADR) for the AlertLens project. Use this skill when a significant design decision has been made or is being debated — new package structure, protocol choice, persistence strategy, auth mechanism, API contract, or any decision whose rationale would be lost without documentation. Produces docs/adr/ADR-NNN_TITLE.md in the project's established format and updates the ADR index in CLAUDE.md.
---

# Write ADR

You are writing an Architecture Decision Record for AlertLens. ADRs are the project's institutional memory for significant decisions. A good ADR explains the *why* so clearly that a new engineer — or future Claude session — can understand the reasoning without needing to ask.

If existing ADRs are present in `docs/adr/`, read one to match the tone, depth, and format before writing. If none exist yet, use the format template below as the reference.

---

## Step 1: Gather context

If the user hasn't already described the decision, ask — one question at a time:

1. "What is the decision in one sentence?"
2. "What problem were you trying to solve? What constraints applied?"
3. "What alternatives did you consider and why did you reject them?"
4. "What are the known trade-offs or downsides of the chosen approach?"

Do not proceed to writing until you have answers to all four.

---

## Step 2: Determine the ADR number

List existing ADRs to find the next available number:

```bash
ls docs/adr/ADR-*.md 2>/dev/null | sort
```

If no ADRs exist yet, start at `ADR-001`. Otherwise the next ADR is `max(existing) + 1`. Use zero-padded three digits: `ADR-001`, `ADR-009`, `ADR-010`, etc.

If the `docs/adr/` directory does not exist, create it before writing the file.

---

## Step 3: Write the ADR

File path: `docs/adr/ADR-NNN_SLUG.md`

Where `SLUG` is the title in SCREAMING_SNAKE_CASE (e.g. `ADR-009_ACTIVITY_LOG_REQUALIFICATION`).

Use this exact format:

```markdown
# ADR-NNN — Title

**Status:** Accepted
**Date:** YYYY-MM-DD
**Related issues:** #NNN (or —)

---

## Context

[2-4 paragraphs. Describe the situation that forced a decision: what was the problem, what constraints existed, what was at stake. Be specific — name the packages, endpoints, or user flows involved. Write for a reader who wasn't in the room.]

---

## Decision

### 1. [Key aspect of the decision]

[Explain what was decided and why. Use sub-sections for complex decisions with multiple moving parts. Include code snippets, API shapes, or data models where they clarify the decision.]

```go
// Example if relevant
```

### 2. [Next aspect if needed]

[...]

---

## Consequences

**Positive:**
- [Concrete benefit — not vague. "Full audit trail from day one", not "better observability"]
- [...]

**Negative:**
- [Honest trade-off — every real decision has at least one]
- [...]

---

## Alternatives Considered

| Option | Rejected because |
|--------|-----------------|
| [Alternative 1] | [Specific reason] |
| [Alternative 2] | [Specific reason] |
```

---

## Step 4: Update CLAUDE.md

Add a row to the ADR index table in `CLAUDE.md`:

```markdown
| [ADR-NNN](docs/adr/ADR-NNN_SLUG.md) | Title | #issues or — |
```

---

## Step 5: Add cross-references (if applicable)

If the decision directly affects specific Go packages or Svelte components, add a comment in the relevant file:

```go
// ADR-NNN: brief reason this code is structured this way
```

Only add this where the code would otherwise seem arbitrary or counterintuitive.

---

## Output

After writing:

```
✓ ADR-NNN written: docs/adr/ADR-NNN_SLUG.md
✓ CLAUDE.md index updated

Key decisions recorded:
- [bullet summary of the decision]
- [trade-off acknowledged]

→ Reference this ADR in code comments where relevant with: // ADR-NNN: [one-line reason]
```

---

## Rules

- **Alternatives Considered is mandatory** — an ADR without rejected alternatives is incomplete.
- **Negative consequences are mandatory** — if you can't find any, you haven't thought hard enough.
- Status is always `Accepted` on creation. Use `Superseded by ADR-NNN` when a later decision overrides it.
- Write in past tense for the decision ("We chose X"), present tense for consequences ("This means Y").
- No marketing language — "simple", "elegant", "powerful" are banned. Be precise.
- If the decision hasn't been made yet and the user is still exploring options, write a **Draft** ADR and list the open questions at the bottom. Mark status `Draft`.
