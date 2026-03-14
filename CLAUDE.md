# CLAUDE.md — AlertLens Project Context

This file provides architecture, development conventions, and coding guidelines for contributors and AI-assisted development tools.

---

## Architecture Overview

AlertLens is a **single, stateless Go binary** that embeds a SvelteKit SPA at build time via `go:embed`.

```
Browser ──► AlertLens (Go binary + embedded SvelteKit frontend)
                │
                ├──► Alertmanager / Grafana Mimir (Alertmanager API v2)
                └──► GitHub / GitLab API  (GitOps push)
```

- **Backend:** Go 1.25, chi router, zap logger
- **Frontend:** SvelteKit + Tailwind CSS + bits-ui (compiled to `dist/`, embedded via `all:dist`)
- **No runtime dependencies:** no database, no Node.js at runtime — all state lives in Alertmanager

The embed directive in `main.go`:
```go
//go:embed all:dist
var distFS embed.FS
```

---

## Internal Package Map

```
internal/
├── alertmanager/   — HTTP clients wrapping Alertmanager API v2; Pool managing multiple instances
├── api/            — chi router wiring, middleware, SPA fallback handler
│   └── handlers/   — one file per resource (alerts, silences, config, routing, incidents, …)
├── auth/           — JWT issuance/validation, CSRF double-submit cookie, MFA, RBAC roles, rate limiting
├── config/         — YAML config loading and validation (alertmanagers, auth, server, gitops)
├── configbuilder/  — Structured CRUD for routes, receivers, time intervals; produces raw YAML
├── gitops/         — GitHub and GitLab pusher implementations (Pusher interface)
└── incident/       — In-memory immutable ledger for incident tracking (ADR-008)
```

---

## Request Lifecycle

```
HTTP request
  └─► chi router
        └─► global middleware stack
              ├── RequestID, RealIP
              ├── zap request logger
              ├── Recoverer (panic → 500)
              ├── MaxBytesReader (10 MiB body cap)
              ├── CSP header injection
              ├── CSRF double-submit cookie (auth.CSRFMiddleware)
              └── CORS
                    └─► route-level auth middleware (RequireRole / RequireViewer)
                              └─► handler → alertmanager.Pool → Alertmanager API
```

---

## Role Hierarchy

Roles form a strict ascending hierarchy — each higher role implies all capabilities of every lower role.

| Role            | Capabilities |
|-----------------|-------------|
| `viewer`        | Read alerts, silences, routing tree |
| `silencer`      | viewer + create / update / expire silences and visual acks |
| `config-editor` | silencer + read and write Alertmanager configuration |
| `admin`         | config-editor + full control (reserved for future admin-only operations) |

Roles are defined in `internal/auth/roles.go`. The middleware shorthands live in `internal/api/router.go`.

---

## Development Commands

```bash
# Build everything (frontend then backend)
make build

# Backend only (Go binary, watches nothing — restart manually)
make dev-backend

# Frontend only (Vite dev server with HMR at http://localhost:5173)
make dev-frontend

# Full stack via Docker Compose (AlertLens UI at http://localhost:9000)
make dev-up
make dev-down
make dev-logs

# Run Go tests (with race detector)
go test ./... -race -count=1

# Run Go tests with verbose output and coverage (make target)
make test

# Run frontend tests
cd web && npm test

# Run E2E tests (requires a pre-built binary at ALERTLENS_BIN)
cd e2e && npx playwright test
```

---

## Coding Conventions

### Error handling

Wrap errors with `fmt.Errorf` and the `%w` verb to preserve the chain. Never use bare `errors.New` for a wrapped error:

```go
// correct
return fmt.Errorf("config: load file: %w", err)

// wrong — loses the original error
return errors.New("failed to load config")
```

### Logging

Use the injected `*zap.Logger` with structured fields. Never use string interpolation in log messages:

```go
// correct
logger.Error("failed to fetch alerts", zap.String("instance", name), zap.Error(err))

// wrong
logger.Error(fmt.Sprintf("failed to fetch alerts for %s: %v", name, err))
```

### HTTP handlers

Every handler receives a `*alertmanager.Pool` (or similar dependency) injected through its constructor. The `handlers` package exposes the following helpers (defined in `internal/api/handlers/auth.go`):

- `resolveClient(pool, w, instance)` — resolves the target AM client; writes a JSON error and returns nil on failure (defined in `internal/api/handlers/helpers.go`)
- `writeJSON(w, v)` — encodes `v` as JSON with a `200 OK` status code
- `writeJSONStatus(w, status, v)` — encodes `v` as JSON with an explicit status code
- `writeError(w, msg, status)` — writes a structured `{"error": "..."}` JSON response

### Tests

- **Table-driven** tests with named sub-cases (`t.Run`)
- **Parallel** where safe (`t.Parallel()` at the top of each sub-test)
- **No third-party test libraries** — stdlib `testing` only (no `testify`, no `gomock`)
- Test files live alongside the package they test (`_test.go` suffix)
- Export helpers for white-box tests go in `export_test.go` (see `internal/alertmanager/export_test.go`)

```go
func TestFoo(t *testing.T) {
    t.Parallel()
    cases := []struct {
        name string
        // ...
    }{
        {name: "happy path", /* ... */},
        {name: "error case", /* ... */},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            // assertions using t.Errorf / t.Fatalf
        })
    }
}
```

### ADRs

Document significant design decisions in `docs/adr/` following the existing format (`# ADR-NNN — Title`, Status, Date, Context, Decision, Consequences). Reference the ADR number in code comments where relevant (e.g., `// ADR-008: incident tracking`).

---

## Key Architecture Decision Records

| ADR | Title | Issues |
|-----|-------|--------|
| [ADR-005](docs/adr/ADR-005_SECURITY_FOUNDATION.md) | Security Foundation — CSRF, JWT, MFA, RBAC, CSP | #30 #31 #32 #33 |
| [ADR-006](docs/adr/ADR-006_ALERT_KANBAN_LIST_VIEWS.md) | Alert Kanban / List views with URL-synced state | — |
| [ADR-007](docs/adr/ADR-007_SILENCES_BULK_ACTIONS.md) | Bulk silence actions | — |
| [ADR-008](docs/adr/ADR-008_INCIDENT_TRACKING.md) | Incident tracking — in-memory immutable ledger | — |

---

## Configuration

The application is configured via a YAML file (default: `alertlens.yaml`) passed with `-config`:

```bash
./alertlens -config alertlens.yaml
```

Most fields can be overridden with environment variables (e.g. `ALERTLENS_ALERTMANAGERS_0_URL`). See `config.example.yaml` for the full reference.

---

## Frontend

The SvelteKit app lives in `web/`. During development, run it separately with `make dev-frontend`. The Vite dev server proxies `/api` calls to the Go backend (started with `make dev-backend`).

Production builds are written directly to `dist/` at the repo root (configured via `svelte.config.js` with `adapter-static`), where `go:embed` picks them up. The Vite `csp-hash` plugin writes `dist/csp-hash.txt` with the `sha256-` hash of any inline scripts so the Go server can inject it into the `Content-Security-Policy` header without `unsafe-inline`.
