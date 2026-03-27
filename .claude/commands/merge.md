# /merge [pr-number] — Squash merge a PR

Merge the PR and clean up the branch.

## What to do

1. **Identify the PR**: if no number is provided, run `gh pr list` to see open PRs and ask which one to merge.

2. **Check comments**:
   ```
   gh pr view PR# --comments
   ```
   If there are unaddressed comments: ask the user how to handle them before continuing.

3. **Squash message**: prepare the final commit message:
   ```
   feat: short, precise description (#PR#)
   ```
   **Wait for user approval.**

4. **Squash merge**:
   ```
   gh pr merge PR# --squash --subject "feat: description (#PR#)"
   ```

5. **Cleanup**:
   - Switch to main and pull: `git checkout main && git pull`
   - Delete the local branch: `git branch -d feat/NNN-feature-slug`

6. **Display state**: summary of what was just merged and the recommended next feature.

## Rules

- Never merge without squash message approval — mandatory human checkpoint
- Never force-push to main
- The spec archive and roadmap update are done at `/ship` time — no post-merge commits needed
- If CI checks fail: report to the user before continuing
