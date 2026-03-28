# Spec: Config Builder — Save & History

**Status**: Approved
**Feature ID**: 009
**Depends on**: 007, 008
**GitHub issues**: #52

## Context

The Config Builder (features 007 and 008) lets users construct and validate an Alertmanager config through guided forms. However, there is no way to publish those changes — the assembled YAML lives only in the frontend until the user manually applies it. This feature adds the final step of the build → review → publish workflow: a diff preview, a save action (to disk or via GitOps), and an in-memory history of recent saves.

The backend already implements `POST /api/config/save` (disk, GitHub, GitLab) and `POST /api/config/diff`. The `YamlDiffViewer` component exists. This feature wires everything together with a UI and adds the missing history layer.

## User stories

- As a config-editor, I want to see a diff of my proposed changes before committing them, so that I do not accidentally push unintended changes to Alertmanager.
- As a config-editor, I want to save the config to disk or push it to a Git repository from the UI, so that I do not need CLI access to apply a change.
- As a config-editor, I want to see a history of saves made since the last restart (with timestamp, mode, and commit SHA), so that I can track what was changed and when.
- As a viewer, I want to see the save history (read-only) to understand recent changes applied to the configuration.

## Acceptance criteria

- [ ] AC-1: A "Save & Deploy" tab (or section) is present in the Config Builder layout and navigable from all existing builder tabs.
- [ ] AC-2: The Save panel fetches the diff between the current live AM config and the proposed YAML via `POST /api/config/diff` and renders it with the existing `YamlDiffViewer` component before the user can confirm a save.
- [ ] AC-3: When `has_changes` is false, the Save button is disabled and a "No changes" message is shown.
- [ ] AC-4: The user can select save mode: `disk`, `github`, or `gitlab`. Modes whose backend pusher is not configured are disabled in the UI with a tooltip explaining why.
- [ ] AC-5: For `disk` mode, a file path field is shown (pre-filled from the Alertmanager config's `config_file` path if available).
- [ ] AC-6: For `github` / `gitlab` modes, GitOps fields (repo, branch, file path, commit message, author name, author email) are shown, pre-filled from the server config when available.
- [ ] AC-7: A successful save triggers a confirmation with the mode and, for GitOps saves, a clickable link to the commit (`html_url`).
- [ ] AC-8: The backend exposes `GET /api/config/history?instance=<name>` returning an array of save records; the endpoint requires `config-editor` role.
- [ ] AC-9: Each save record written by `POST /api/config/save` is appended to the in-memory history store (scoped per Alertmanager instance) and contains: `saved_at` (RFC 3339), `mode`, `alertmanager`, `actor` (role from JWT), `commit_sha` (empty for disk), `html_url` (empty for disk).
- [ ] AC-10: The history list in the frontend displays each record with timestamp, mode badge, actor, and a diff expand button; the diff on expand calls `POST /api/config/diff` with the live config and the YAML saved at that point — the save record must therefore also store the `raw_yaml`.
- [ ] AC-11: History is in-memory only; it resets on process restart. No persistence is implemented in this feature.
- [ ] AC-12: The Save action is gated behind the `config-editor` role in the backend (`RequireRole("config-editor")`); the History read endpoint requires at minimum `config-editor` role.
- [ ] AC-13: The history store is protected by a mutex and safe for concurrent reads and writes.

## Out of scope

- Persistence of save history to disk or database (tracked in feature 024 / activity-log-persistence).
- Webhook trigger from the Save UI (the backend already supports `webhook_url` in the payload; this feature does not expose it in the frontend).
- Configuration of GitOps tokens from the UI (tokens are set in the static YAML config file).
- Live reload of Alertmanager after a disk save (Alertmanager hot-reload is outside AlertLens scope).
- Diffing against a previous history entry (the expand button always diffs against the current live config, not the state at save time).

## Open questions

- [ ] Q1: **Save button placement** — should "Save & Deploy" be a fourth tab in the Config Builder tab bar (alongside Routing, Time Intervals, Receivers), or a persistent "Save" button always visible in the layout header? A tab keeps the layout consistent but requires navigation; a header button is always accessible but adds visual weight. What is your preference?
- [ ] Q2: **Actor label** — the JWT carries only a role (e.g., `config-editor`), not a username. Should the history record's `actor` field show the role string, or should we introduce an optional `display_name` field in the JWT / login flow for future readability?
- [ ] Q3: **Diff stored in history** — storing the full `raw_yaml` per history entry means memory usage is proportional to config size × save count. Should we cap the in-memory history at a fixed size (e.g., last 50 saves), or leave it unbounded until restart?
- [ ] Q4: **GitOps pre-fill** — the backend config holds static GitOps defaults (repo, branch, file path). Should the frontend fetch these defaults via a new `GET /api/config/gitops-defaults` endpoint, or is it acceptable to leave the fields empty (requiring the user to fill them on each save)?
