# Roadmap — AlertLens

_Last updated: 2026-03-19_

## Milestones

### Milestone 1 — Core MVP (Done)

| ID  | Feature | Description | Status | Depends on | Issues |
|-----|---------|-------------|--------|------------|--------|
| 001 | alert-visualization | Kanban + List views, URL-synced state | [x] | — | #45 |
| 002 | alert-filtering | Matcher syntax filtering, label grouping, debounce | [x] | 001 | #46 |
| 003 | multi-alertmanager | Aggregate multiple AM/Mimir instances, instance filter | [x] | 001 | #47 |
| 004 | silences-management | Create, expire, bulk-silence, visual ack | [x] | 001 | #39 #49 |
| 005 | routing-tree-visualizer | Interactive D3 route graph, click-to-match alerts | [x] | 001 | #38 #48 |
| 006 | security-foundation | JWT, CSRF, MFA, RBAC, CSP (ADR-005) | [x] | — | #30 #31 #32 #33 |

### Milestone 2 — Config Builder (In progress)

| ID  | Feature | Description | Status | Depends on | Issues |
|-----|---------|-------------|--------|------------|--------|
| 007 | config-builder-routing | CRUD for routing tree via guided forms + YAML preview | [x] | 006 | #50 |
| 008 | config-builder-receivers | CRUD for receivers and time intervals | [x] | 007 | #51 |
| 009 | config-builder-save-history | Save to Alertmanager + change history / diff view | [x] | 007 008 | #52 |
| 010 | auth-rbac-fixes | Fix default role bug, login UX, role validation at startup | [x] | 006 | #87 #88 #89 #90 #91 #92 |
| 011 | activity-log | Requalify incident tracking as Session Activity Log (ADR-009) | [ ] | 006 | #97 |

### Milestone 3 — Essential Companion

| ID  | Feature | Description | Status | Depends on | Issues |
|-----|---------|-------------|--------|------------|--------|
| 012 | sso-oidc | SSO / OIDC integration for authentication | [ ] | 006 | #95 |
| 013 | routing-simulator | Test alert labels against live routing tree in-UI | [ ] | 005 007 | #99 |
| 014 | gitops-drift-detection | Detect drift between live AM config and Git source of truth | [ ] | 009 | #100 |
| 015 | webhook-receiver | Webhook endpoint for real-time alert ingestion | [ ] | 006 | #96 |
| 016 | config-linter | Alertmanager config linter with in-UI feedback | [ ] | 007 | #103 |
| 017 | route-coverage-report | Identify unrouted alerts and dead routes | [ ] | 005 007 | #104 |
| 018 | observability | `/metrics` Prometheus endpoint + structured request logging | [ ] | — | #67 |

### Milestone 4 — Intelligence & Scale

| ID  | Feature | Description | Status | Depends on | Issues |
|-----|---------|-------------|--------|------------|--------|
| 019 | silence-recommendations | Suggest silences based on alert firing patterns | [ ] | 002 004 | #105 |
| 020 | alert-notifications | Slack and Microsoft Teams alert summary notifications | [ ] | 015 | #101 |
| 021 | alert-correlation | Correlation and grouping visualization across instances | [ ] | 003 | #98 |
| 022 | multi-tenant-dashboard | Scoped views by team, namespace, or tenant | [ ] | 003 012 | #106 |
| 023 | receiver-test | Send a test alert to any receiver from the UI | [ ] | 008 | #102 |
| 024 | activity-log-persistence | Persist activity log to disk (SQLite / BoltDB) | [ ] | 011 | #61 |

## Cross-cutting tracks

| ID  | Feature | Description | Status | Depends on | Issues |
|-----|---------|-------------|--------|------------|--------|
| 025 | jwt-secret-rotation | Rotate JWT secret on password change without invalidating all sessions | [ ] | 006 | #65 |
| 026 | response-caching | ETag / cache headers for read-heavy endpoints | [ ] | — | #66 |
| 027 | vite8-migration | Migrate frontend to Vite 8 (ecosystem upgrade) | [ ] | — | #107 |
| 028 | e2e-tests-multi-role | Playwright E2E tests for multi-role login UX | [ ] | 010 | #90 #91 #92 |

## Dependency graph

```
006 → 007 → 008 → 009
006 → 010
006 → 011 → 024
001 → 002
001 → 003 → 021
001 → 004 → 019
001 → 005 → 013
           005 + 007 → 013
           005 + 007 → 017
007 → 016
009 → 014
006 → 012 → 022
           003 + 012 → 022
015 → 020
```

## Legend

- `[ ]` Not started
- `[~]` In progress (spec/plan/tasks/implement underway)
- `[x]` Done and archived in `specs/_archive/`
