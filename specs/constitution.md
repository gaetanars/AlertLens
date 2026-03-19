# Constitution — AlertLens

_This document captures the project's identity and founding principles. Written once, rarely changed._

## Identity

**What this project is**: AlertLens is a modern UI for Prometheus Alertmanager — a single stateless Go binary embedding a SvelteKit SPA that makes the full alert lifecycle (visualize, silence, configure, push) accessible without any runtime dependencies.

**Why it exists**: Alertmanager exposes a powerful API but an unusable built-in UI. AlertLens bridges that gap, letting SRE teams understand, visualize, and act on alerts at scale — from multi-instance aggregation to GitOps-driven configuration management.

**Who uses it**: SRE and on-call engineers managing Prometheus alerting at scale, from a single Alertmanager instance to multi-tenant Grafana Mimir deployments.

## Tech stack

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Backend | Go 1.25, chi router, zap logger | Statically compiled, low footprint, standard lib coverage |
| Frontend | SvelteKit + Tailwind CSS + bits-ui | Compiled SPA, tree-shaken, zero runtime overhead |
| Embed | `go:embed all:dist` | Single binary deployment, no Node at runtime |
| Auth | JWT + CSRF double-submit + MFA + RBAC | Defense-in-depth; no session storage needed |
| GitOps | GitHub / GitLab API pushers | Config changes committed directly to source of truth |
| Alertmanager | API v2 (Alertmanager + Grafana Mimir) | Speaks the native protocol; no shadow state |

## Non-negotiable principles

1. **Stateless**: no database, no persistent runtime state — all state lives in Alertmanager. In-memory stores (e.g. activity log) are acceptable for session-scoped data only.
2. **Single binary**: frontend is embedded via `go:embed`; Node.js is a build-time dependency only.
3. **Security first**: CSRF, CSP, RBAC enforced at every write endpoint. Every route declares its minimum required role. No shortcuts.
4. **Alertmanager-native**: read and write exclusively through Alertmanager API v2. No proprietary data formats, no shadow config.
5. **RBAC by design**: the role hierarchy (`viewer → silencer → config-editor → admin`) is enforced in middleware, not in handlers. Role checks are never optional.
6. **Error chain preservation**: errors are always wrapped with `%w`; structured zap logging with fields — no string interpolation in log messages.

## What this project is not

- Not a full IRM: no PagerDuty/Opsgenie/Grafana IRM replacement — no persistence, escalations, or post-mortems
- Not a metrics platform: AlertLens reads Alertmanager, not Prometheus metrics directly
- Not a database-backed system: no PostgreSQL, no Redis, no persistent volumes required
- Not multi-user collaborative: no real-time presence, no lock management on concurrent edits

## Minimum quality bar

- Tests: table-driven, parallel, stdlib only (`testing` package — no testify, no gomock)
- Every new Go package ships with `_test.go` covering the happy path and at least one error case
- Frontend: TypeScript strict mode; no `any` casts without justification
- CI must pass: `go test ./... -race`, `golangci-lint`, `tsc --noEmit`, frontend unit tests
- ADRs required for significant architecture decisions (docs/adr/ directory)
