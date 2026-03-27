# /ship — Branch, commit, push, open a PR

Prepare and send the work to GitHub.

## Prerequisite

Check that a `review.md` exists with verdict **"Ready to merge"** for the current feature. If not, stop and ask to run `/review` first.

## What to do

1. **Branch**: if on `main` or `master`, create a feature branch:
   ```
   git checkout -b feat/NNN-feature-slug
   ```
   If a branch already exists for this feature, use it.

2. **Archive the spec**: if the feature has a spec directory (`specs/NNN-feature-name/`), move it to `specs/_archive/NNN-feature-name/` and update `specs/roadmap.md` status to `[x]` now — before staging. This way the archive is part of the squash commit and `/merge` requires no extra commits on `main`.

3. **Staging**: identify files related to the feature (including the archived spec and updated roadmap). Never `git add .` — list and add files explicitly. Show the list to the user before staging.

4. **Commit**: propose a message in Conventional Commits format:
   ```
   feat(scope): short, precise description
   ```
   **Wait for explicit user approval before committing.**

5. **Push**: `git push -u origin feat/NNN-feature-slug`

6. **Collect issue references**: read `specs/NNN-feature-name/spec.md` and extract the `GitHub issues:` field. For each `#NNN`, add a `Closes #NNN` line to the PR body. If the feature has no linked issues, omit this section.

7. **Pull Request**: create the PR with `gh pr create`:

```
gh pr create \
  --title "feat(scope): short, precise description" \
  --body "$(cat <<'EOF'
## Summary

- [bullet: what was built]
- [bullet: key technical decision if non-obvious]

## Changes

<!-- list the main files added or modified, grouped by layer -->
**Backend**
- `internal/foo/bar.go` — [what changed]

**Frontend**
- `web/src/lib/api/foo.ts` — [what changed]

## Spec

`specs/NNN-feature-name/spec.md`

## Acceptance criteria

- [ ] AC-1: [copy from spec]
- [ ] AC-2: [copy from spec]

## Test plan

- [ ] `go test ./... -race` passes
- [ ] `cd web && npm test` passes
- [ ] [Feature-specific manual step — e.g. "Create a silence and verify it appears in Alertmanager"]

## Closes

Closes #NNN
Closes #NNN

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

8. **Labels**: apply relevant labels with `gh pr edit NNN --add-label "..."` based on what was changed:
   - `backend` if Go files were modified
   - `frontend` if Svelte/TS files were modified
   - `security` if auth or middleware was touched
   - `testing` if only tests were added

9. Return the PR URL.

## Rules

- Never skip commit approval — it's an intentional human checkpoint
- Always list staged files explicitly, never `git add .`
- **Always include `Closes #NNN`** for every issue linked in the spec — this auto-closes issues on merge
- Copy the acceptance criteria from the spec into the PR body verbatim — reviewers should not need to open the spec to understand what was built
- If a PR already exists for this branch: push to the existing branch without creating a new PR
- If the feature has no spec (hotfix, chore): omit the Spec and Acceptance criteria sections, keep Summary and Test plan
