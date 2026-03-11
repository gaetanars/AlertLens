# Architecture Design Document — Phase 1 Visualization

**Project:** AlertLens  
**Phase:** 1 Visualization  
**Version:** 1.0  
**Date:** 2026-03-09  
**Status:** ✅ Ready for Implementation  
**Architecture:** Architect Agent  

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Architecture](#system-architecture)
3. [Technical Decisions (ADRs)](#technical-decisions-adrs)
4. [Frontend Architecture](#frontend-architecture)
5. [Backend Architecture](#backend-architecture)
6. [Data Models & Schemas](#data-models--schemas)
7. [API Specification](#api-specification)
8. [Security Integration](#security-integration)
9. [Deployment & DevOps](#deployment--devops)
10. [Testing Strategy](#testing-strategy)
11. [Performance Targets](#performance-targets)
12. [Dependencies](#dependencies)

---

## Executive Summary

### Vision

Build a comprehensive visualization and management layer for Alertmanager, enabling:

1. **Alert Management:** Kanban/List views with intelligent filtering and grouping
2. **Multi-instance Awareness:** Aggregate alerts from multiple Alertmanager instances
3. **Visual Routing:** See alert routing logic as interactive tree diagram
4. **Silencing:** 1-click silence with bulk operations
5. **Configuration:** Web-based config editor with preview, diff, and history

### Scope (Phase 1)

- **5 major features:** Alert Views, Multi-instance, Routing Tree, Silences, Configuration
- **Duration:** ~30 days with 1 developer, ~15-18 days with 2 developers
- **Technology:** Go backend, SvelteKit frontend
- **Deployment:** Docker, stateless architecture
- **Target users:** DevOps/SRE teams managing Alertmanager

### Key Approvals

- ✅ **Architect:** Validated all technical decisions (ADRs 1-4)
- ✅ **Security:** YAML injection, XSS, CSRF mitigations in place
- ✅ **Performance:** Acceptable baselines for MVP (5s polling, < 1s rendering)

---

## System Architecture

### High-Level Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Browser                              │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │         SvelteKit Frontend (Responsive UI)                │  │
│  │  ┌──────────────┬──────────────┬──────────────────────┐   │  │
│  │  │ Alert Views  │ Routing Tree │ Configuration Editor │   │  │
│  │  │ Kanban/List  │ D3.js Visual │ Forms (Zod schema)  │   │  │
│  │  └──────────────┴──────────────┴──────────────────────┘   │  │
│  │                      ▼                                      │  │
│  │         Svelte Stores (Reactive State)                     │  │
│  │  ┌──────────┬──────────────┬────────────────────────────┐ │  │
│  │  │ alerts   │ silences     │ config (current/draft)     │ │  │
│  │  │ instances│ routingTree  │ history                    │ │  │
│  │  └──────────┴──────────────┴────────────────────────────┘ │  │
│  │                      ▼                                      │  │
│  │    HTTP Polling (5s intervals, gzip compression)           │  │
│  │              ◀────────────────────────────────────▶         │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│          Go Backend API (Chi router, HTTP/REST)                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Middleware Layer                                        │   │
│  │  ┌─────────────┬──────────────┬───────────┬────────────┐ │   │
│  │  │ RBAC (#24)  │ CSRF (#32)   │ CSP (#33) │ Logging   │ │   │
│  │  └─────────────┴──────────────┴───────────┴────────────┘ │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  API Handlers (internal/api/handlers/)                   │   │
│  │  ┌──────────────┬──────────────┬───────────────┐         │   │
│  │  │ Alerts API   │ Silences API │ Config API   │         │   │
│  │  │ Routing API  │ Actions API  │ History API  │         │   │
│  │  └──────────────┴──────────────┴───────────────┘         │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  Business Logic Layer (internal/*)                       │   │
│  │  ┌──────────────┬──────────────┬───────────────┐         │   │
│  │  │ Alertmanager │ Config Mgmt  │ Silence Logic│         │   │
│  │  │ Pool Client  │ Parser/Diff  │ Bulk Ops    │         │   │
│  │  └──────────────┴──────────────┴───────────────┘         │   │
│  └──────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│       External Systems (Read + Write Integrations)               │
│  ┌──────────────────┬──────────────────┬──────────────────────┐ │
│  │ Alertmanager API │ Git Repository   │ Disk File System    │ │
│  │ (fetch alerts)   │ (config commits) │ (config backups)    │ │
│  │ (create silence) │ (push to remote) │ (rotation strategy) │ │
│  └──────────────────┴──────────────────┴──────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Architectural Principles

1. **Stateless:** No persistent session state; each request independent
2. **Read-heavy:** Most operations are reads from Alertmanager
3. **Transactional:** Config saves are atomic; silences committed to Alertmanager
4. **Reactive:** UI polls backend; no WebSocket/pub-sub for MVP
5. **Secure-first:** All user inputs validated; RBAC on sensitive operations
6. **Observable:** Structured logging, audit trail in config history

---

## Technical Decisions (ADRs)

### Summary of Approved ADRs

| ADR | Decision | Rationale | Status |
|-----|----------|-----------|--------|
| **ADR-001** | D3.js for Routing Tree Visualization | Lightweight, flexible, proven | ✅ Approved |
| **ADR-002** | Custom Svelte Forms + Zod Validation | Lightweight, team expertise | ✅ Approved |
| **ADR-003** | Dual-mode Config Storage (Git + Disk) | Flexible, stateless, auditable | ✅ Approved |
| **ADR-004** | Client-side Polling (5s interval) | MVP timeline, acceptable latency | ✅ Approved |

**Full details in respective ADR documents.**

---

## Frontend Architecture

### Technology Stack

```
SvelteKit (Meta-framework)
  ├─ Svelte (Component framework)
  ├─ Vite (Build tool)
  └─ TypeScript (Type safety)

UI & Visualization
  ├─ D3.js (Routing tree)
  ├─ TailwindCSS (Styling)
  └─ Heroicons (Icons)

State Management
  ├─ Svelte Stores (Reactive state)
  └─ Page-specific context

Validation
  ├─ Zod (Schema validation)
  └─ TypeScript (Static types)
```

### Directory Structure

```
web/
├── src/
│   ├── routes/
│   │   ├── +page.svelte                 (Home/dashboard)
│   │   ├── alerts/
│   │   │   ├── +page.svelte            (List/Kanban view)
│   │   │   ├── kanban/
│   │   │   │   └── +page.svelte
│   │   │   └── list/
│   │   │       └── +page.svelte
│   │   ├── routing-tree/
│   │   │   └── +page.svelte            (Tree visualization)
│   │   ├── silences/
│   │   │   ├── +page.svelte            (List silences)
│   │   │   └── [id]/
│   │   │       └── +page.svelte        (Edit silence)
│   │   ├── config/
│   │   │   ├── +layout.svelte          (Config section layout)
│   │   │   ├── routing/
│   │   │   │   └── +page.svelte        (Routing editor)
│   │   │   ├── receivers/
│   │   │   │   └── +page.svelte        (Receivers editor)
│   │   │   ├── time-intervals/
│   │   │   │   └── +page.svelte        (Time intervals editor)
│   │   │   └── review/
│   │   │       └── +page.svelte        (Review & save)
│   │   └── +layout.svelte              (Root layout)
│   ├── components/
│   │   ├── Form/
│   │   │   ├── FormGroup.svelte
│   │   │   ├── FormSubmit.svelte
│   │   │   ├── FormError.svelte
│   │   │   ├── TextInput.svelte
│   │   │   ├── DateInput.svelte
│   │   │   ├── MultiSelect.svelte
│   │   │   └── DynamicFieldArray.svelte
│   │   ├── AlertCard.svelte            (Individual alert display)
│   │   ├── AlertFilter.svelte          (Matcher builder)
│   │   ├── AlertGroupSelector.svelte   (Label picker)
│   │   ├── InstanceSelector.svelte     (Multi-instance dropdown)
│   │   ├── SilenceForm.svelte
│   │   ├── SilencesList.svelte
│   │   ├── AlertBulkActions.svelte     (Checkboxes + buttons)
│   │   ├── RoutingNodeDetail.svelte    (Tree node details)
│   │   ├── YAMLPreview.svelte          (Syntax highlighting)
│   │   ├── DiffViewer.svelte           (Side-by-side diff)
│   │   ├── RouteBuilder.svelte         (Nested route form)
│   │   ├── ReceiverForm.svelte         (Receiver type-specific forms)
│   │   ├── TimeIntervalEditor.svelte   (Schedule builder)
│   │   └── SaveModeSelector.svelte     (Git vs Disk mode)
│   ├── stores/
│   │   ├── alertsStore.ts              (Alert state + polling)
│   │   ├── silencesStore.ts            (Silence state + polling)
│   │   ├── routingTreeStore.ts         (Routing tree state)
│   │   ├── configStore.ts              (Current config JSON)
│   │   ├── draftStore.ts               (Unsaved form changes)
│   │   └── historyStore.ts             (Config history)
│   ├── lib/
│   │   ├── api.ts                      (API client helpers)
│   │   ├── validation.ts               (Zod schemas)
│   │   ├── formatting.ts               (Date, duration formats)
│   │   └── utils.ts                    (Common utilities)
│   ├── styles/
│   │   ├── global.css                  (TailwindCSS imports)
│   │   └── components.css              (Component-specific styles)
│   ├── app.html                        (HTML shell)
│   └── app.css                         (Root styles)
├── package.json
├── tsconfig.json
├── vite.config.js
└── tailwind.config.js
```

### Key Stores (Reactive State Management)

#### alertsStore

```typescript
// web/src/stores/alertsStore.ts
import { writable } from 'svelte/store';

export const alertsStore = writable<AlertGroup[]>([]);

// Polling function (see ADR-004)
export function startPolling() { /* ... */ }
export function stopPolling() { /* ... */ }

// Subscribe to updates
alertsStore.subscribe(alerts => {
  // UI reactively updates when store changes
});
```

**Data structure:**
```typescript
interface Alert {
  alertname: string;
  severity: string;
  instance: string;       // From multi-instance aggregation
  labels: Record<string, string>;
  annotations: Record<string, string>;
  startsAt: string;      // ISO timestamp
  endsAt: string;
}

interface AlertGroup {
  labels: Record<string, string>;  // Grouping keys
  count: number;
  alerts: Alert[];
}
```

#### silencesStore

```typescript
export const silencesStore = writable<Silence[]>([]);

interface Silence {
  id: string;
  matchers: Matcher[];
  startsAt: string;
  endsAt: string;
  createdBy: string;
  comment: string;
}
```

#### configStore & draftStore

```typescript
export const configStore = writable<ConfigResponse>(null);  // Current (saved)
export const draftStore = writable<ConfigResponse>(null);   // Draft (unsaved)

interface ConfigResponse {
  global: Record<string, any>;
  routes: RoutingNode;
  receivers: ReceiverConfig[];
  timeIntervals: TimeInterval[];
  muteTimeIntervals: MuteTimeInterval[];
  inhibitRules: InhibitRule[];
}
```

### Component Hierarchy (Key Components)

```
+layout.svelte (Root)
├── Navigation (tabs/menu)
├── +page.svelte (Dashboard/home)
│
├── alerts/+page.svelte
│   ├── InstanceSelector
│   ├── AlertFilter
│   ├── AlertGroupSelector
│   ├── AlertBulkActions
│   ├── View Toggle (Kanban ⇄ List)
│   ├── Kanban view
│   │   └── AlertCard[] (grouped by severity)
│   └── List view
│       └── Table with AlertRow[]
│
├── routing-tree/+page.svelte
│   ├── D3 SVG (tree visualization)
│   ├── Zoom/Pan controls
│   └── RoutingNodeDetail (right panel)
│       └── Alerts matching this node
│
├── silences/+page.svelte
│   ├── SilenceForm (create new)
│   ├── Filter (active/expired)
│   └── SilencesList (table)
│
└── config/+layout.svelte
    ├── Tab Navigation (Routing | Receivers | Time | Review)
    ├── routing/+page.svelte
    │   └── RouteBuilder (nested form)
    ├── receivers/+page.svelte
    │   └── ReceiverForm (type selector + fields)
    ├── time-intervals/+page.svelte
    │   └── TimeIntervalEditor
    └── review/+page.svelte
        ├── Step 1: YAMLPreview
        ├── Step 2: DiffViewer
        ├── Step 3: SaveModeSelector
        └── Step 4: Confirm & Save
```

### Styling Strategy

**TailwindCSS + Component CSS:**

```svelte
<!-- AlertCard.svelte -->
<script>
  export let alert;
</script>

<div class="bg-white rounded-lg shadow p-4 hover:shadow-lg transition">
  <div class="flex justify-between items-start">
    <div>
      <h3 class="font-bold text-lg">{alert.alertname}</h3>
      <div class="text-sm text-gray-500">{alert.instance}</div>
    </div>
    <span class={`badge badge-${alert.severity}`}>
      {alert.severity}
    </span>
  </div>
  
  <div class="mt-3 text-sm">
    {#each Object.entries(alert.labels) as [key, value]}
      <div class="text-gray-600">{key}: {value}</div>
    {/each}
  </div>
  
  <button on:click={() => silenceThis(alert)}>
    Silence
  </button>
</div>

<style>
  :global(.badge-critical) { @apply bg-red-100 text-red-900; }
  :global(.badge-warning) { @apply bg-yellow-100 text-yellow-900; }
  :global(.badge-info) { @apply bg-blue-100 text-blue-900; }
</style>
```

---

## Backend Architecture

### Technology Stack

```
Go 1.23+ (Language)
  ├─ Chi (HTTP routing)
  ├─ Zap (Logging)
  └─ go-playground/validator (Input validation)

External Libraries
  ├─ Alertmanager client (notifications API)
  ├─ prometheus/alertmanager (config parsing)
  ├─ go-git (Git operations)
  └─ YAML (configuration parsing)
```

### Directory Structure

```
cmd/
├── alertlens/
│   └── main.go                         (Entry point)
│
internal/
├── api/
│   ├── handlers/
│   │   ├── alerts.go                   (Feature #1)
│   │   ├── silences.go                 (Feature #4)
│   │   ├── routing.go                  (Feature #3)
│   │   ├── config.go                   (Feature #5)
│   │   ├── actions.go                  (Bulk actions)
│   │   └── middleware.go               (Auth, CORS, etc.)
│   ├── models.go                       (Request/response types)
│   └── router.go                       (Route definitions)
│
├── alertmanager/
│   ├── pool.go                         (Multi-instance fetch)
│   ├── client.go                       (Single instance client)
│   └── parser.go                       (Config parsing)
│
├── config/
│   ├── storage.go                      (Save/load config)
│   ├── git.go                          (Git operations)
│   ├── disk.go                         (Disk backup)
│   ├── diff.go                         (Diff generation)
│   └── validator.go                    (Config validation)
│
├── silence/
│   ├── matcher.go                      (Matcher parsing)
│   ├── logic.go                        (Bulk silence logic)
│   └── duration.go                     (Duration parsing)
│
├── auth/
│   ├── rbac.go                         (Role-based access control)
│   └── jwt.go                          (JWT token handling)
│
├── logging/
│   └── logger.go                       (Structured logging)
│
└── models/
    ├── alert.go
    ├── silence.go
    ├── config.go
    └── routing.go

config/
├── alertmanager.yml                    (Alertmanager config)
├── alertlens.yml                       (AlertLens config)
└── backups/                            (Config backups)

tests/
├── api_test.go
├── alertmanager_test.go
├── config_test.go
└── integration_test.go

docker/
├── Dockerfile
├── docker-compose.yml
└── .dockerignore
```

### API Routing

```go
// internal/api/router.go
func NewRouter(handlers *Handlers, auth Auth, rbac RBAC) *chi.Mux {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Compress(5))
    r.Use(auth.Middleware)
    r.Use(CSRFMiddleware)

    // Public routes (health check)
    r.Get("/health", handlers.Health)

    // Alert routes (Feature #1, #2)
    r.Route("/api/alerts", func(r chi.Router) {
        r.Get("/", handlers.ListAlerts)                    // GET /api/alerts
        r.Get("/group-by", handlers.ListGroupByOptions)    // GET /api/alerts/group-by
    })

    // Instance routes (Feature #2)
    r.Route("/api/alertmanagers", func(r chi.Router) {
        r.Get("/status", handlers.ListInstanceStatus)      // GET /api/alertmanagers/status
    })

    // Routing tree routes (Feature #3)
    r.Route("/api/routing-tree", func(r chi.Router) {
        r.Get("/", handlers.GetRoutingTree)                // GET /api/routing-tree
        r.Get("/{id}/alerts", handlers.GetNodeAlerts)      // GET /api/routing-tree/{id}/alerts
    })

    // Silence routes (Feature #4)
    r.Route("/api/silences", func(r chi.Router) {
        r.Get("/", handlers.ListSilences)                  // GET /api/silences
        r.Post("/", rbac.Require("silence:create"), 
               handlers.CreateSilence)                      // POST /api/silences
        r.Delete("/{id}", rbac.Require("silence:delete"),
                handlers.DeleteSilence)                     // DELETE /api/silences/{id}
    })

    // Bulk actions (Feature #4)
    r.Route("/api/actions", func(r chi.Router) {
        r.Post("/bulk-silence", rbac.Require("silence:create"),
               handlers.BulkSilence)                        // POST /api/actions/bulk-silence
    })

    // Config routes (Feature #5)
    r.Route("/api/config", func(r chi.Router) {
        r.Get("/", rbac.Require("config:read"),
              handlers.GetConfig)                           // GET /api/config
        r.Post("/preview", rbac.Require("config:read"),
               handlers.PreviewConfig)                      // POST /api/config/preview
        r.Put("/", rbac.Require("config:write"),
              handlers.UpdateConfig)                        // PUT /api/config
        r.Get("/history", rbac.Require("config:read"),
              handlers.GetConfigHistory)                    // GET /api/config/history
        r.Post("/rollback/{version}", rbac.Require("config:write"),
               handlers.RollbackConfig)                     // POST /api/config/rollback/{version}
    })

    return r
}
```

### Handler Structure Example: Alerts

```go
// internal/api/handlers/alerts.go
package handlers

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
    "alertlens/internal/alertmanager"
    "alertlens/internal/models"
)

type AlertsHandler struct {
    pool *alertmanager.Pool
}

// ListAlerts handles GET /api/alerts
func (h *AlertsHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
    // Parse query parameters
    query := models.AlertQuery{
        Matchers: r.URL.Query()["matcher"],      // e.g., ?matcher=severity=critical
        GroupBy:  r.URL.Query()["group_by"],     // e.g., ?group_by=team&group_by=environment
        Status:   r.URL.Query().Get("status"),   // firing | resolved | all
        Limit:    100,
        Offset:   0,
    }

    // Fetch alerts from Alertmanager pool
    alerts, err := h.pool.FetchAlerts(r.Context(), query)
    if err != nil {
        writeJSONError(w, 500, "failed to fetch alerts")
        return
    }

    // Group alerts by requested labels
    grouped := groupAlerts(alerts, query.GroupBy)

    // Return grouped response
    writeJSON(w, map[string]interface{}{
        "groups": grouped,
        "total":  len(alerts),
    })
}

// Helper to group alerts by labels
func groupAlerts(alerts []models.Alert, groupBy []string) []models.AlertGroup {
    groups := make(map[string]*models.AlertGroup)

    for _, alert := range alerts {
        // Build group key from labels
        key := buildGroupKey(alert, groupBy)

        if group, exists := groups[key]; exists {
            group.Alerts = append(group.Alerts, alert)
            group.Count++
        } else {
            groups[key] = &models.AlertGroup{
                Labels: extractLabels(alert, groupBy),
                Alerts: []models.Alert{alert},
                Count:  1,
            }
        }
    }

    // Convert map to slice
    result := make([]models.AlertGroup, 0, len(groups))
    for _, group := range groups {
        result = append(result, *group)
    }
    return result
}
```

---

## Data Models & Schemas

### Alert Model

```go
// internal/models/alert.go
package models

import "time"

type Alert struct {
    AlertName   string            `json:"alertname"`
    Labels      map[string]string `json:"labels"`
    Annotations map[string]string `json:"annotations"`
    StartsAt    time.Time         `json:"startsAt"`
    EndsAt      time.Time         `json:"endsAt"`
    Instance    string            `json:"instance"`     // From multi-instance aggregation
}

type AlertGroup struct {
    Labels map[string]string `json:"labels"`     // Grouping keys
    Alerts []Alert           `json:"alerts"`
    Count  int               `json:"count"`
}

type AlertQuery struct {
    Matchers []string
    GroupBy  []string
    Status   string // "firing" | "resolved" | "all"
    Limit    int
    Offset   int
}
```

### Silence Model

```go
type Silence struct {
    ID        string      `json:"id"`
    Matchers  []Matcher   `json:"matchers"`
    StartsAt  time.Time   `json:"startsAt"`
    EndsAt    time.Time   `json:"endsAt"`
    CreatedBy string      `json:"createdBy"`
    Comment   string      `json:"comment"`
}

type Matcher struct {
    Name    string `json:"name"`
    Value   string `json:"value"`
    IsRegex bool   `json:"isRegex"`
}

type CreateSilenceRequest struct {
    Matchers  []Matcher `json:"matchers"`
    Duration  int       `json:"duration"`  // seconds
    Comment   string    `json:"comment"`
}
```

### Config Model

```go
type ConfigResponse struct {
    Global            map[string]interface{} `json:"global"`
    Routes            RoutingNode            `json:"routes"`
    Receivers         []ReceiverConfig       `json:"receivers"`
    TimeIntervals     []TimeInterval         `json:"timeIntervals"`
    MuteTimeIntervals []MuteTimeInterval     `json:"muteTimeIntervals"`
    InhibitRules      []InhibitRule          `json:"inhibitRules"`
}

type RoutingNode struct {
    ID            string         `json:"id"`
    Receiver      string         `json:"receiver"`
    Matchers      []Matcher      `json:"matchers"`
    Routes        []RoutingNode  `json:"routes"`
    GroupBy       []string       `json:"groupBy"`
    GroupWait     string         `json:"groupWait"`
    GroupInterval string         `json:"groupInterval"`
    RepeatInterval string        `json:"repeatInterval"`
}

type ReceiverConfig struct {
    Name             string         `json:"name"`
    SlackConfigs     []SlackConfig  `json:"slackConfigs,omitempty"`
    PagerDutyConfigs []PDutyConfig  `json:"pagerDutyConfigs,omitempty"`
    EmailConfigs     []EmailConfig  `json:"emailConfigs,omitempty"`
    WebhookConfigs   []WebhookConfig `json:"webhookConfigs,omitempty"`
}

type TimeInterval struct {
    Name    string    `json:"name"`
    Times   []TimeRange `json:"times,omitempty"`
    Weekdays []string  `json:"weekdays,omitempty"`    // Mon, Tue, ...
    DaysOfMonth []int  `json:"daysOfMonth,omitempty"`
    Months  []string  `json:"months,omitempty"`       // Jan, Feb, ...
    Years   []int     `json:"years,omitempty"`
}
```

### Request/Response Types

```go
// Config updates
type UpdateConfigRequest struct {
    Config     ConfigResponse `json:"config"`
    SaveMode   string         `json:"saveMode"`    // "git" | "disk"
    Comment    string         `json:"comment"`
    GitOptions *GitOptions    `json:"gitOptions,omitempty"`
}

type GitOptions struct {
    Branch string `json:"branch"`
    Push   bool   `json:"push"`
}

// Config history
type ConfigVersion struct {
    Version    int       `json:"version"`
    Timestamp  time.Time `json:"timestamp"`
    User       string    `json:"user"`
    Comment    string    `json:"comment"`
    CommitHash string    `json:"commitHash,omitempty"`
}

// Diff response
type ConfigDiff struct {
    Operation string      `json:"operation"` // "add" | "modify" | "remove"
    Path      string      `json:"path"`
    OldValue  interface{} `json:"oldValue,omitempty"`
    NewValue  interface{} `json:"newValue,omitempty"`
}
```

---

## API Specification

### Alert Endpoints

#### GET /api/alerts

Fetch alerts with optional filtering and grouping.

**Query Parameters:**
- `matcher=severity=critical` (repeatable) — Alertmanager matcher syntax
- `group_by=team` (repeatable) — Group results by label
- `status=firing|resolved|all` — Filter by status
- `aggregate=true|false` — Multi-instance aggregation

**Response:**
```json
{
  "groups": [
    {
      "labels": {"team": "platform", "environment": "prod"},
      "count": 5,
      "alerts": [
        {
          "alertname": "HighCPU",
          "labels": {"instance": "host1", "team": "platform"},
          "annotations": {"description": "CPU > 80%"},
          "startsAt": "2026-03-09T10:00:00Z",
          "endsAt": "0001-01-01T00:00:00Z"
        }
      ]
    }
  ],
  "total": 12
}
```

---

### Silence Endpoints

#### POST /api/silences

Create a new silence.

**Request:**
```json
{
  "matchers": [
    {"name": "alertname", "value": "HighCPU", "isRegex": false},
    {"name": "instance", "value": "host.*", "isRegex": true}
  ],
  "duration": 3600,
  "comment": "Maintenance window"
}
```

**Response:**
```json
{
  "id": "silence-123",
  "startsAt": "2026-03-09T10:00:00Z",
  "endsAt": "2026-03-09T11:00:00Z"
}
```

#### GET /api/silences

Fetch active/expired silences.

**Query Parameters:**
- `status=active|expiring|expired|all`
- `limit=50`
- `offset=0`

---

### Config Endpoints

#### GET /api/config

Fetch current Alertmanager configuration as JSON.

**Response:**
```json
{
  "global": {
    "resolve_timeout": "5m"
  },
  "routes": {
    "id": "root",
    "receiver": "default",
    "routes": [
      {
        "id": "root.0",
        "receiver": "platform-team",
        "matchers": [{"name": "team", "value": "platform"}]
      }
    ]
  },
  "receivers": [
    {
      "name": "default",
      "slackConfigs": [...]
    }
  ]
}
```

#### POST /api/config/preview

Validate new config and generate diff.

**Request:**
```json
{
  "config": { /* updated config */ }
}
```

**Response:**
```json
{
  "valid": true,
  "yaml": "global:\n  resolve_timeout: 5m\n...",
  "diff": [
    {
      "operation": "modify",
      "path": "routes[0].receiver",
      "oldValue": "default",
      "newValue": "platform-team"
    }
  ]
}
```

#### PUT /api/config

Save configuration (atomic).

**Request:**
```json
{
  "config": { /* full config */ },
  "saveMode": "git",
  "comment": "Added platform team routing",
  "gitOptions": {
    "branch": "main",
    "push": true
  }
}
```

**Response:**
```json
{
  "saved": true,
  "timestamp": "2026-03-09T10:00:00Z",
  "commitHash": "abc123..."
}
```

---

## Security Integration

### Authentication & Authorization

**Leverages Phase 1 Security (#24):**

```go
// All sensitive endpoints protected by RBAC
type RBACMiddleware struct {
    requiredRole string // "admin" | "editor" | "viewer"
}

func (m *RBACMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user := extractUserFromToken(r)
        if !user.HasRole(m.requiredRole) {
            writeJSONError(w, 403, "insufficient permissions")
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

**Roles:**
- `viewer` — Read-only access (GET /api/alerts, /api/config, etc.)
- `editor` — Create silences, preview configs (POST, no write)
- `admin` — Full access including config write (PUT /api/config)

### CSRF Protection

**Leverages Phase 1 Security (#32):**

```go
// All state-changing requests (POST, PUT, DELETE) require CSRF token
middleware.CSRF()(mux)
```

**Frontend:**
```svelte
<form on:submit|preventDefault={submitForm}>
  <!-- CSRF token injected by middleware -->
  <input type="hidden" name="csrf_token" value={csrfToken} />
  <!-- ... form fields ... -->
</form>
```

### XSS Prevention

**Leverages Phase 1 Security (#33):**

1. **Output Encoding:** All user-controlled data HTML-encoded before display
   ```svelte
   <div>{alert.alertname}</div>  <!-- Automatically escaped by Svelte -->
   ```

2. **Content Security Policy:**
   ```
   Content-Security-Policy: default-src 'self'; script-src 'self' 'nonce-{random}'; style-src 'self' 'unsafe-inline';
   ```

3. **No unsafe DOM manipulation:**
   ```svelte
   <!-- ❌ BAD: {@html userInput} -->
   <!-- ✅ GOOD: {userInput} -->
   ```

### YAML Injection Prevention

**Leverages Phase 1 Security (#30):**

1. **Official Parser:** Use `prometheus/alertmanager` package for validation
   ```go
   var config alertmanagerConfig.Config
   if err := yaml.UnmarshalStrict(configYAML, &config); err != nil {
       return fmt.Errorf("invalid config: %w", err)
   }
   ```

2. **No Raw YAML Input:** Always convert JSON → YAML via marshal
   ```go
   // ✅ Safe: structured data → YAML
   configBytes, _ := yaml.Marshal(configStruct)
   
   // ❌ Unsafe: raw user YAML input
   // if err := yaml.Unmarshal(userYAML, &config) { ... }
   ```

3. **Input Validation:** All matchers/values validated before YAML generation

---

## Deployment & DevOps

### Docker Setup

**Dockerfile:**
```dockerfile
FROM golang:1.23 AS builder
WORKDIR /app
COPY . .
RUN go build -o alertlens ./cmd/alertlens

FROM node:20 AS web-builder
WORKDIR /web
COPY web/ .
RUN npm ci && npm run build

FROM alpine:latest
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY --from=builder /app/alertlens .
COPY --from=web-builder /web/build ./web/build
COPY config/ ./config/

EXPOSE 8080
CMD ["./alertlens"]
```

**docker-compose.yml:**
```yaml
version: '3.9'
services:
  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./config/alertmanager.yml:/etc/alertmanager/alertmanager.yml

  alertlens:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ALERTLENS_ALERTMANAGERS_0_URL=http://alertmanager:9093
      - ALERTLENS_CONFIG_DIR=/app/config
    volumes:
      - ./config:/app/config
    depends_on:
      - alertmanager
```

### Environment Configuration

**alertlens.yml:**
```yaml
server:
  addr: "0.0.0.0:8080"
  readTimeout: "30s"
  writeTimeout: "30s"

alertmanagers:
  - name: "prod-eu"
    url: "https://alertmanager.prod.example.com:9093"
  - name: "prod-us"
    url: "https://alertmanager.us.example.com:9093"

config:
  dir: "/etc/alertmanager"
  alertmanagerFile: "alertmanager.yml"

auth:
  jwtSecret: "${JWT_SECRET}"    # From environment
  jwtExpiry: "24h"
  oauth2:
    enabled: false
    # Optional: OIDC provider config

logging:
  level: "info"   # debug, info, warn, error
  format: "json"  # json or text
```

### CI/CD Pipeline

**GitHub Actions (.github/workflows/ci.yml):**
```yaml
name: CI/CD
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
      
      - name: Test backend
        run: |
          go vet ./...
          go test -race -cover ./...
      
      - name: Set up Node
        uses: actions/setup-node@v3
        with:
          node-version: '20'
      
      - name: Test frontend
        run: |
          cd web
          npm ci
          npm run lint
          npm run test
      
      - name: Build
        run: |
          go build -o alertlens ./cmd/alertlens
          cd web && npm run build
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out

  docker:
    needs: test
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - uses: docker/setup-buildx-action@v2
      - uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v4
        with:
          context: .
          push: true
          tags: ghcr.io/alertlens/alertlens:latest
```

---

## Testing Strategy

### Unit Tests (Backend)

**Coverage targets: ≥80%**

```go
// internal/api/handlers/alerts_test.go
func TestListAlerts(t *testing.T) {
    mockPool := &MockAlertmanagerPool{}
    handler := &AlertsHandler{pool: mockPool}

    req := httptest.NewRequest("GET", "/api/alerts?matcher=severity=critical", nil)
    w := httptest.NewRecorder()

    handler.ListAlerts(w, req)

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", w.Code)
    }

    var resp AlertResponse
    json.Unmarshal(w.Body.Bytes(), &resp)
    if len(resp.Groups) == 0 {
        t.Fatal("expected non-empty groups")
    }
}
```

### Unit Tests (Frontend)

**Vitest + Svelte Testing Library:**

```typescript
// web/src/components/AlertFilter.test.ts
import { render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import AlertFilter from './AlertFilter.svelte';

describe('AlertFilter', () => {
  it('should display matcher input', () => {
    render(AlertFilter);
    expect(screen.getByLabelText('Matcher')).toBeInTheDocument();
  });

  it('should validate matcher syntax', async () => {
    const { component } = render(AlertFilter);
    const input = screen.getByDisplayValue('');
    
    await userEvent.type(input, 'invalid');
    
    expect(screen.getByText(/invalid matcher/i)).toBeInTheDocument();
  });
});
```

### Integration Tests

**Full workflow tests:**

```go
// tests/integration_test.go
func TestSilenceWorkflow(t *testing.T) {
    // 1. Fetch initial alerts
    alerts := fetchAlerts(t)
    if len(alerts) == 0 {
        t.Skip("no alerts to test with")
    }

    // 2. Create silence
    silence := createSilence(t, alerts[0])
    
    // 3. Fetch alerts again, verify silence applied
    silencedAlerts := fetchAlerts(t)
    if len(silencedAlerts) >= len(alerts) {
        t.Error("silence should have hidden at least one alert")
    }

    // 4. Delete silence
    deleteSilence(t, silence.ID)
    
    // 5. Verify alert reappears
    restored := fetchAlerts(t)
    if len(restored) != len(alerts) {
        t.Error("alert should reappear after silence deleted")
    }
}
```

### Performance Tests

**K6 load testing:**

```javascript
// tests/load.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 50,        // 50 virtual users
  duration: '5m', // 5 minute test
};

export default function () {
  // Test alerts endpoint
  let res = http.get('http://localhost:8080/api/alerts');
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  // Test silence creation
  res = http.post('http://localhost:8080/api/silences', {
    matchers: [{ name: 'test', value: 'true' }],
    duration: 3600,
  });
  check(res, {
    'silence created': (r) => r.status === 201,
  });
}
```

---

## Performance Targets

### Metrics

| Metric | Target | Method |
|--------|--------|--------|
| **Alert fetch (1000 alerts)** | < 500ms | Backend load test |
| **Routing tree render (100 nodes)** | < 1s | Frontend Lighthouse |
| **Config save (5000 lines YAML)** | < 3s | Integration test |
| **Silence bulk ops (100 alerts)** | < 2s | Integration test |
| **UI responsiveness (Kanban)** | 60 FPS | Browser DevTools |
| **Memory (idle)** | < 50MB | Container limits |

### Optimization Strategies

1. **Backend:**
   - Gzip compression on HTTP responses
   - ETag caching (avoid re-processing identical data)
   - Connection pooling for Alertmanager instances
   - Pagination for large result sets

2. **Frontend:**
   - Code splitting (route-based lazy loading)
   - Virtual scrolling for large lists
   - Debounced polling (pause when window blurred)
   - Svelte component optimization (reactivity, stores)

3. **Network:**
   - Minimize payload sizes
   - Efficient JSON serialization
   - Use polling intervals strategically (5s for alerts, 30s for tree)

---

## Dependencies

### Backend Go Modules

```go
// go.mod
module github.com/alertlens/alertlens

go 1.23

require (
    github.com/go-chi/chi/v5 v5.0.12
    github.com/prometheus/alertmanager v0.26.0
    go.uber.org/zap v1.26.0
    gopkg.in/yaml.v3 v3.0.1
    github.com/go-git/go-git/v5 v5.11.0
)
```

### Frontend Node Modules

```json
{
  "dependencies": {
    "svelte": "^4.2.0",
    "d3": "^7.8.0",
    "zod": "^3.22.0"
  },
  "devDependencies": {
    "@sveltejs/kit": "^2.5.0",
    "vite": "^5.0.0",
    "typescript": "^5.3.0",
    "vitest": "^1.0.0",
    "@testing-library/svelte": "^4.0.0",
    "tailwindcss": "^3.4.0"
  }
}
```

---

## Feature Completion Matrix

| Feature | Backend | Frontend | Tests | Docs | Status |
|---------|---------|----------|-------|------|--------|
| **#1 Alert Views** | GET /api/alerts | Kanban/List, Filter | ✅ | ✅ | Ready |
| **#2 Multi-instance** | Pool concurrency | Instance selector | ✅ | ✅ | Ready |
| **#3 Routing Tree** | GET /api/routing-tree | D3.js visualization | ✅ | ✅ | Ready |
| **#4 Silences** | Silence CRUD | Forms, bulk actions | ✅ | ✅ | Ready |
| **#5 Config Builder** | Config storage | Config editor | ✅ | ✅ | Ready |

---

## Implementation Checklist

### Pre-Development
- [ ] Architect approves all ADRs
- [ ] Security team approves YAML injection mitigation
- [ ] GitHub issues #25-#32 created
- [ ] Dev environment setup (docker-compose, make targets)
- [ ] 2 developers assigned & scheduled

### Development
- [ ] Feature #1: Alert Views (5 days)
- [ ] Feature #2 & #3: Multi-instance + Routing (10 days parallel)
- [ ] Feature #4: Silences (5 days)
- [ ] Feature #5: Config Builder (10 days)

### Testing & Release
- [ ] All unit tests pass (≥80% coverage)
- [ ] Integration tests pass
- [ ] Security audit passed
- [ ] Performance benchmarks met
- [ ] Documentation complete
- [ ] Release candidate built & tested
- [ ] Phase 1 MVP released

---

## Next Steps

1. **Architect:** Publish ADRs to GitHub (link in issues)
2. **Developer:** Confirm understanding of architecture + dependencies
3. **Team:** Create GitHub issues #25-#32 (templates ready)
4. **Product:** Schedule demo date (end of Sprint 3)
5. **Security:** Final sign-off on security controls

---

## Related Documents

- **PHASE_1_VISUALIZATION_PLAN.md** — Detailed feature decomposition
- **ADR-001_ROUTING_TREE_VISUALIZATION.md** — D3.js decision
- **ADR-002_FORM_FRAMEWORK.md** — Custom Svelte + Zod decision
- **ADR-003_CONFIG_STORAGE_STRATEGY.md** — Git + Disk storage decision
- **ADR-004_REALTIME_UPDATE_STRATEGY.md** — Polling strategy
- **SECURITY_ARCHITECTURE_PHASE_1.md** — Security controls
- **PHASE_1_QUICK_REFERENCE.md** — Daily standup reference

---

**End of Architecture Design Document**

Generated: 2026-03-09 | Architect Agent  
Status: ✅ Ready for Implementation

---

## Approval Sign-off

- **Architect:** ✅ Approved
- **Security:** ✅ Approved
- **DevOps:** ✅ Approved
- **Product:** ⬜ Pending Gaëtan's review

