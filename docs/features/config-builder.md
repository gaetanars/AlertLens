# Configuration Builder

The Configuration Builder is AlertLens's differentiating feature: a guided interface to create and modify Alertmanager configurations without editing YAML directly.

!!! warning "config-editor role required"
    The Configuration Builder is only accessible to users with the `config-editor` (or `admin`) role. Viewers can browse the routing tree, receiver list, and time interval list in read-only mode.

---

## Overview

The builder covers the three main sections of `alertmanager.yml`:

1. **Routing Tree** — the hierarchy of routes that determines which receiver handles an alert
2. **Receivers** — integrations (Slack, PagerDuty, email, webhook, etc.)
3. **Time Intervals** — named temporal windows referenced by routes to mute or restrict notifications

All changes are validated with the **official Prometheus Alertmanager library** before any write or push operation.

---

## Routing Tree Editor

The visual editor lets you build and modify the route hierarchy through forms — no YAML knowledge required.

### Supported route fields

- `matchers` / `match` / `match_re`
- `receiver`
- `continue`
- `group_by`, `group_wait`, `group_interval`, `repeat_interval`
- `mute_time_intervals` — suppress notifications during the selected intervals (child routes only)
- `active_time_intervals` — send notifications only during the selected intervals (child routes only)
- Child routes (nested)

!!! note "Root route restriction"
    `mute_time_intervals` and `active_time_intervals` are not available on the root route — this is an Alertmanager constraint.

A **live YAML preview** updates in real time as you make changes, so you always see exactly what will be written.

---

## Time Intervals Manager

Define named `time_intervals` (the root-level key in `alertmanager.yml`) using a calendar-style interface:

- Select days of the week (`weekdays`), time ranges (`times`), days of the month (`days_of_month`), months, years, and timezone (`location`)
- Support for complex recurring patterns (e.g., weekends, business hours, maintenance windows)
- Named intervals are then referenced in child routes as `mute_time_intervals` (suppress notifications) or `active_time_intervals` (restrict notifications to that window)

---

## Receivers

Guided forms for the five most common Alertmanager integration types:

| Type | Key fields |
|---|---|
| **Slack** | Channel, API URL, username, title, message text |
| **PagerDuty** | Routing key (v2) or service key (v1), description |
| **Email** | To, from, smarthost, SMTP credentials |
| **Webhook** | URL, max alerts per batch |
| **OpsGenie** | API key, message, priority |

All types expose a `send_resolved` toggle to control whether recovery notifications are sent.

**Inline validation** runs automatically as you type (500 ms debounce) and displays errors below the form before you attempt to save.

**Raw YAML fallback** — receivers using an integration type not listed above (e.g. VictorOps, Telegram, MSTeams) are shown as an editable YAML textarea. Their configuration is preserved verbatim on round-trip; AlertLens never silently drops unknown keys.

**Delete guard** — before deleting a receiver AlertLens checks which routes reference it. If any routes do, an inline confirmation lists each referencing route's matchers and depth. You can proceed ("Delete anyway") or cancel.

---

## Save & Deploy

The **Save & Deploy** tab is the final step in the build → review → publish workflow. It is always accessible from the Config Builder tab bar, regardless of which editor tab is active.

### Diff preview

Before saving, AlertLens fetches a unified diff between the current live Alertmanager configuration and the proposed YAML assembled by the builder. The diff is rendered with green/red line highlighting using the existing `YamlDiffViewer` component. If there are no changes, the Save button is disabled and a "No changes" message is shown instead.

### Save modes

Three save modes are available. Modes whose backend pusher is not configured are shown as disabled with an explanatory tooltip.

#### Disk

AlertLens writes `alertmanager.yml` directly to the file path you specify.

!!! info "Filesystem access"
    AlertLens must have write access to the file path. In Docker, mount the directory as a volume.

#### GitHub / GitLab

AlertLens commits and pushes the updated configuration to a Git repository via the forge API. Required fields:

| Field | Description |
|---|---|
| Repository | `owner/repo` (GitHub) or `namespace/project` (GitLab) |
| Branch | Target branch (e.g., `main`) |
| File path | Path inside the repo (e.g., `config/alertmanager.yml`) |
| Commit message | Free text |
| Author name / email | Commit attribution |

On success, a confirmation banner is shown with a clickable link to the commit on the forge.

Configure tokens in your [AlertLens config](../configuration.md#gitops):

```yaml
gitops:
  github:
    token: ""   # ALERTLENS_GITOPS_GITHUB_TOKEN
  gitlab:
    token: ""   # ALERTLENS_GITOPS_GITLAB_TOKEN
```

### Save history

Below the save form, the **Save History** section lists all saves made since the last process restart, newest-first. Each row shows:

- Timestamp (RFC 3339)
- Save mode badge (disk / github / gitlab)
- Actor (the role of the user who triggered the save)
- An **Expand diff** button — clicking it fetches a diff between the saved YAML and the current live config and renders it inline

!!! note "Session-scoped history"
    Save history is in-memory and resets on process restart. Up to 50 saves are retained per Alertmanager instance (oldest evicted on overflow). Persistent history is tracked in feature 024.

---

## Validation

AlertLens validates configuration at two points:

**Inline (as you edit)** — receiver and time interval forms call a dedicated validate endpoint 500 ms after the last keystroke. Errors appear inline below the form so you catch mistakes before attempting to save.

**Pre-save** — every full configuration generated by the builder is validated using the official `github.com/prometheus/alertmanager/config` package before it is written or pushed. Invalid configurations are rejected with a clear error message — they never reach your Alertmanager.
