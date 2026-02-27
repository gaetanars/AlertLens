# CLAUDE.md — AlertLens

Project guide for Claude Code and contributors.

## Project overview

AlertLens is a lightweight, stateless Go binary (+ embedded SvelteKit frontend) that provides a modern UI for Prometheus Alertmanager and Grafana Mimir: alert visualization, silence management, visual acknowledgements, and a configuration builder with GitOps push support.

Single binary, no database, Alertmanager API v2 only.

## Build

```bash
# Full build (frontend → Go binary)
make build

# Frontend only (outputs to dist/ at repo root)
cd web && npm run build

# Go only (requires dist/ to already exist)
CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=dev" -o alertlens .
```

The frontend Vite build outputs to `../dist` relative to `web/` (i.e. `dist/` at the repo root). This is the directory that Go embeds via `go:embed all:dist`.

## Test

```bash
go test ./... -race -count=1
go vet ./...
```

Tests are in `*_test.go` files colocated with the packages they test.

## Dev environment

```bash
# Spin up Alertmanager + Prometheus + MailHog + echo webhook server
docker compose -f dev/docker-compose.yml up -d

# Backend hot-reload (reads dev/alertlens/config.yaml via config.example.yaml)
make dev-backend

# Frontend hot-reload (Vite dev server, proxies API to :9000)
make dev-frontend
```

Admin password in the dev environment: `admin` (see `dev/alertlens/config.yaml`).

## Architecture

```
main.go                        — entry point, wiring, graceful shutdown
internal/
  config/                      — YAML config + env override (reflect-based)
  auth/                        — JWT service + per-IP rate limiter (5 req/min)
  alertmanager/                — API v2 client, concurrent pool, ack index
  api/
    router.go                  — chi router, global middlewares
    handlers/                  — one file per domain (alerts, silences, config, auth…)
  configbuilder/               — validate (official AM pkg), structured diff, atomic write, webhook
  gitops/                      — Pusher interface → GitHub and GitLab implementations
web/                           — SvelteKit + Tailwind + shadcn/svelte
dist/                          — compiled frontend (git-ignored, produced by `make web-build`)
```

## Key conventions

### Go

- `config.Load()` returns `(*Config, []string, error)` — the second value is non-fatal warnings (unparsable env vars).
- GitOps clients must be declared as `var ghPusher gitops.Pusher` (interface), never `*GitHubPusher` (concrete). A typed-nil concrete pointer passes a non-nil interface check — this is the typed-nil trap.
- Use `writeJSONStatus(w, status, v)` for non-200 JSON responses. Writing the status code before headers causes `Content-Type` to be lost if done naively.
- `configbuilder.GenerateDiff` returns `([]DiffHunk, bool)` — structured JSON format, never ANSI escape sequences.
- Regex patterns are compiled once and cached in a `sync.Map` inside `alertmanager/pool.go`.
- `http.MaxBytesReader` is set to 10 MiB globally in the router middleware.

### Frontend

- SvelteKit with `@sveltejs/adapter-static` — fully static build, no SSR.
- D3.js for the routing tree visualizer.
- Dark/light mode via `mode-watcher`.

### No-go list

- No database or persistent local state — AlertLens is stateless by design.
- No Alertmanager API v1 — v2 only.
- No OIDC in v1 (planned).

## CI/CD

| Trigger | Workflow | Action |
|---|---|---|
| Push / PR to `main` | `ci.yml` | `go vet` + `go test -race` + build check |
| Tag `v*` | `release.yml` | 5 cross-compiled binaries + multi-arch Docker image (`ghcr.io/gaetanars/alertlens`) |
| Push to `main` or tag `v*` | `docs.yml` | MkDocs Material → GitHub Pages (mike versioning) |

## Releasing

1. Update `docs/changelog.md` with the changes.
2. Create and push a tag:
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```
3. The `release.yml` workflow handles everything else automatically.

## What NOT to commit

- `alertlens.yaml` (personal config with secrets) — use `config.example.yaml` as template.
- `dist/` — build artifact, produced locally or in CI.
- `.claude/projects/` — personal Claude Code memory, machine-specific.
