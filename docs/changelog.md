# Changelog

All notable changes to AlertLens are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
AlertLens uses [Semantic Versioning](https://semver.org/).

---

## [Unreleased]

---

## [v0.2.0] — 2026-03-07

### Changed

- Project moved to the **AlertLens** GitHub organization (`github.com/AlertLens/AlertLens`).
- Go module path updated from `github.com/gaetanars/alertlens` to `github.com/alertlens/alertlens`.
- Docker image renamed from `ghcr.io/gaetanars/alertlens` to `ghcr.io/alertlens/alertlens`.
- Helm OCI chart path updated from `ghcr.io/gaetanars/chart/alertlens` to `ghcr.io/alertlens/chart/alertlens`.
- Documentation site moved from `gaetanars.github.io/AlertLens` to `alertlens.github.io/AlertLens`.

### Fixed

- Helm chart OCI registry path corrected from `ghcr.io/alertlens/charts/alertlens` to `ghcr.io/alertlens/chart/alertlens` in the release workflow, chart README, and Kubernetes deployment documentation.

---

## [v0.1.0] — 2026-02-27

### Added

- **Alert Visualization** — Kanban and dense list view with Alertmanager native matcher syntax filtering and label-based grouping.
- **Multi-instance aggregation** — Connect multiple Alertmanager or Grafana Mimir instances; alerts are aggregated into a single view with per-alert source indicators.
- **Routing Tree Visualizer** — Interactive graph representation of the `alertmanager.yml` route hierarchy; click a node to see matching active alerts. Nodes display inline badges for `mute_time_intervals` (orange) and `active_time_intervals` (green).
- **1-click Silences** — Create Alertmanager silences directly from active alerts with pre-filled matchers and a human-friendly duration picker.
- **Bulk Actions** — Select multiple alerts and apply silence or visual ack in one operation.
- **Visual Ack** — Stateless acknowledgement mechanism implemented on top of Alertmanager silences with reserved labels (`alertlens_ack_type`, `alertlens_ack_by`, `alertlens_ack_comment`). Alerts remain visible and display a badge showing the acknowledger.
- **Configuration Builder** — Guided UI for editing the routing tree, receivers, and time intervals, validated with the official Prometheus library before any write.
- **Time Intervals Manager** — Admin tab to manage the `time_intervals` root section of `alertmanager.yml`. Supports all spec fields: `weekdays`, `times`, `days_of_month`, `months`, `years`, `location` (timezone).
- **Route-level time interval selectors** — The visual route editor exposes `mute_time_intervals` and `active_time_intervals` on child routes, with color-coded chip selectors (orange = mute, green = active). Not available on the root route (Alertmanager constraint).
- **Disk-write mode** — Write `alertmanager.yml` atomically to a configured file path, with optional webhook trigger.
- **GitOps mode** — Push `alertmanager.yml` changes to GitHub or GitLab via API with configurable branch, path, and commit message, plus optional webhook trigger.
- **Admin mode** — Password-protected admin session (JWT, in-memory, no persistence). Read-only access is always public.
- **Rate limiting** — Per-IP rate limiter on the login endpoint (5 requests/minute).
- **Single binary** — SvelteKit frontend embedded via `go:embed`. Zero runtime dependencies.
- **Docker image** — Multi-stage Dockerfile based on `gcr.io/distroless/static-debian12`.
- **Grafana Mimir support** — `X-Scope-OrgID` header configurable per instance.
- **TLS skip verify** — Per-instance option for trusted internal environments.

### Architecture

- Backend: Go 1.25, `go-chi/chi` router, `go.uber.org/zap` logger
- Frontend: SvelteKit + Tailwind CSS + shadcn/svelte
- Auth: `golang-jwt/jwt/v5`, `golang.org/x/time` rate limiter
- GitOps: `google/go-github/v66`, `xanzy/go-gitlab`
- Config validation: `prometheus/alertmanager/config`
- Diff: `sergi/go-diff/diffmatchpatch`
