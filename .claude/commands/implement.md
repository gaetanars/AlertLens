# /implement [feature-name] — Implement one task

Implement the next unchecked task and mark it done.

## Auto-detection

If no name is provided:
- Find the first feature with unchecked `[ ]` tasks in `tasks.md`
- Identify the next unchecked task
- Ask for confirmation: "Continuing **feature-name** — next: T-X.Y ([N/total]). Shall we go?"

## What to do

1. Read `specs/NNN-feature-name/tasks.md`, `spec.md`, and `plan.md`
2. Load the relevant context: files to modify, interfaces, APIs used
3. Based on the task's **Developer writes** flag:

   **Developer writes: No** — Claude implements fully:
   - Write the complete code
   - Run tests / build
   - Briefly explain the important technical choices (so the user understands)

   **Developer writes: Yes** — Claude prepares, user writes:
   - Create the file with the structure, imports, and types
   - Explain the pattern to use and why
   - Mark the exact spot: `// TODO(developer): [clear instruction on what to write]`
   - Wait for the user to implement before continuing

4. Verify the implementation works (tests, build)
5. Check the task in `tasks.md`: `- [x] **T-X.Y**: ...`
6. Announce what's next

## After each task

```
✓ T-X.Y done.
Next: T-X.Z — [description]. Run /implement to continue.
```

If all tasks are checked:
```
✓ All tasks completed!
Run /review to audit the implementation before shipping.
```

## Rules

- **One task per invocation.** Don't chain multiple tasks without confirmation.
- If a task reveals an unforeseen design problem: stop, explain the issue, discuss with the user before continuing.
- Never modify `spec.md` or `plan.md` during implementation — flag any needed changes instead.
- Respect existing patterns and conventions in the codebase.
