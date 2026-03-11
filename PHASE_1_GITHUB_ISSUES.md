# 📌 Phase 1 Visualization — Issues GitHub à Créer

**Pour mapper le plan de décomposition aux issues GitHub**

---

## 📋 Issues Proposées

### FEATURE 1: Alert Kanban/List Views

#### Issue #25: Feature: Alert List & Kanban Views
```
Title: Phase 1: Alert visualization — Kanban and List views

Body:
## Objective
Provide two visualization modes for active alerts:
- **Kanban board:** Columns by severity (critical, warning, info)
- **List view:** Dense table with sort/filter capabilities

## Requirements
- Kanban columns: critical | warning | info | (other)
- List table: sortable columns (name, severity, labels, duration)
- Toggle between views
- Responsive design (mobile, tablet, desktop)

## Acceptance Criteria
- [ ] Kanban board renders with correct severity grouping
- [ ] List table displays all required columns
- [ ] View toggle functional
- [ ] Responsive on all breakpoints
- [ ] Tests: unit + visual regression
- [ ] < 500ms load for 1000 alerts

## Implementation Notes
- Backend: existing GET /api/alerts enhanced with grouping
- Frontend: Svelte components (alerts/kanban, alerts/list)
- D3 not required; CSS Grid sufficient

## Related
- #26 (Filtering)
- #27 (Multi-instance)
```

**Assignee:** Developer (priority: 1)  
**Effort:** 5 days  
**Label:** `feature`, `visualization`, `phase-1`

---

#### Issue #26: Feature: Alert Filtering & Grouping
```
Title: Phase 1: Alert filtering with Alertmanager matchers and label grouping

Body:
## Objective
Enable users to filter and group alerts using Alertmanager's native matcher syntax.

## Requirements
- Filter syntax: label=value, label!=value, label=~regex, label!~regex
- Multi-filter support (AND logic)
- Group by: any label (team, environment, severity, etc.)
- Filter builder UI: freeform + visual suggestions
- Persist filter in URL (querystring)

## Acceptance Criteria
- [ ] Matcher syntax validation works
- [ ] Filters applied correctly to alert list
- [ ] Grouping recalculates on filter change
- [ ] UI suggests available labels
- [ ] URL reflects current filters
- [ ] Tests: filter logic, edge cases

## Implementation Notes
- Backend: enhance GET /api/alerts query parsing
- Frontend: AlertFilter component, reactive store

## Related
- #25 (Alert Views)
- #20 (Context Alerts) — context feature separate
```

**Assignee:** Developer  
**Effort:** 2 days  
**Label:** `feature`, `phase-1`

---

### FEATURE 2: Multi-instance Aggregation

#### Issue #27: Feature: Multi-instance Aggregation & Filtering
```
Title: Phase 1: Multi-Alertmanager aggregation and instance filtering

Body:
## Objective
Support multiple Alertmanager/Mimir instances in a single unified view, with per-instance filtering.

## Requirements
- Concurrent fetch from all configured instances
- Graceful error handling: N/M instances fail → still show results from others
- Alert origin badge: show source instance for each alert
- Instance filter dropdown: "All" | "prod-eu" | "prod-us" | etc.
- Instance health status indicator

## Acceptance Criteria
- [ ] Pool.FetchAlertsAll() fetches concurrently
- [ ] Instance timeout (5s) doesn't block others
- [ ] Alert.Instance field populated correctly
- [ ] Instance filter dropdown functional
- [ ] Status endpoint shows instance health
- [ ] Tests: concurrent fetch, error handling, timeout

## Performance
- 3 instances × 500 alerts each: < 2s total fetch time

## Implementation Notes
- Backend: concurrent goroutines with sync.Mutex
- Frontend: instance selector dropdown + badge display

## Related
- #25 (Alert Views)
```

**Assignee:** Developer  
**Effort:** 4 days  
**Label:** `feature`, `phase-1`, `multi-instance`

---

### FEATURE 3: Routing Tree Visualizer

#### Issue #28: Feature: Interactive Routing Tree Visualizer
```
Title: Phase 1: Alertmanager routing tree visualization and node interaction

Body:
## Objective
Provide an interactive graph visualization of the Alertmanager routing tree, with the ability to:
- View routing hierarchy (tree layout)
- Click node → see matching alerts
- Display node properties (receiver, matchers, timing)

## Requirements
- D3.js-based tree visualization
- Node display: receiver name, matcher count
- Edge display: parent → child relationships
- Click node → fetch matching alerts (GET /api/routing-tree/node/{id}/alerts)
- Zoom & pan controls
- Tooltip on hover (matchers, group-by, timing)

## Acceptance Criteria
- [ ] Routing tree endpoint returns valid JSON structure
- [ ] D3 tree renders all nodes + edges
- [ ] Click node fetches and displays matching alerts
- [ ] Node detail panel shows all properties
- [ ] Zoom & pan functional
- [ ] Responsive on large trees (100+ nodes)
- [ ] Tests: tree parsing, node matching, D3 rendering

## Performance
- 100-node tree renders in < 1s

## Implementation Notes
- Backend: GET /api/routing-tree (parse routing config)
- Backend: GET /api/routing-tree/node/{id}/alerts (node-level matching)
- Frontend: D3 integration (tree layout + SVG rendering)
- Frontend: RouteNodeDetail component (properties + alerts)

## Related
- #25 (Alert Views) — for displaying matching alerts
```

**Assignee:** Developer  
**Effort:** 6 days  
**Label:** `feature`, `visualization`, `phase-1`

---

### FEATURE 4: Silences & Bulk Actions

#### Issue #29: Feature: Silences Management & Bulk Actions
```
Title: Phase 1: Silence creation, management, and bulk alert actions

Body:
## Objective
Enable users to silence alerts and perform bulk actions:
- Create silence from any alert (1-click)
- Manage silences (list, filter, expire)
- Bulk actions: select multiple alerts → silence/ack all

## Requirements
- POST /api/silences: create silence with matchers + duration
- GET /api/silences: list silences (active, expired, all)
- DELETE /api/silences/{id}: expire silence
- POST /api/actions/bulk-silence: silence multiple alerts at once
- SilenceForm UI: duration picker (1h, 4h, EOD, custom), matcher editor
- SilencesList page: filter + manage active/expired
- Bulk selector: checkboxes on alert cards/rows

## Duration options
- Pre-defined: 1h, 4h, till end of day, weekend
- Custom: date/time picker

## Acceptance Criteria
- [ ] Create silence with matchers and duration
- [ ] List silences with filtering (active/expired/all)
- [ ] 1-click silence from alert card
- [ ] Bulk silence selected alerts (creates single silence if possible)
- [ ] SilenceForm duration picker functional
- [ ] Tests: silence CRUD, bulk operations, matcher logic
- [ ] Real-time: silence list updates when new silence created

## Implementation Notes
- Backend: enhance internal/api/handlers/silences.go
- Frontend: SilenceForm, SilencesList, AlertBulkActions components
- Use Alertmanager API v2 POST /api/v2/silences

## Related
- #25 (Alert Views) — for alert selection
- #4 (Ack visual) — optional enhancement
```

**Assignee:** Developer  
**Effort:** 5 days  
**Label:** `feature`, `operations`, `phase-1`

---

### FEATURE 5: Configuration Builder

#### Issue #30: Feature: Configuration Builder — Routing Tree Editor
```
Title: Phase 1: Visual routing tree editor with form-based configuration

Body:
## Objective
Enable users to visually edit the Alertmanager routing tree without touching YAML.

## Requirements
- Form-based route builder (nested routes)
- Add/edit/delete routes
- Route fields: matchers, receiver, group-by, group timing, continue flag
- Mute/active time intervals selector (multiselect)
- YAML preview (live, syntax-highlighted)
- Diff viewer: preview changes before saving

## Acceptance Criteria
- [ ] Create/edit routes via form
- [ ] Nested routes support (add child routes)
- [ ] YAML preview updates in real-time
- [ ] All route properties editable (matchers, receiver, timing)
- [ ] Tests: form validation, YAML generation, diff logic

## Implementation Notes
- Backend: no changes (routes already parsed in #28)
- Frontend: RouteBuilder component (deeply nested forms)
- Frontend: YAMLPreview component (syntax highlighting)
- Use @sveltejs/form or custom form handling

## Related
- #31 (Receivers & Time Intervals)
- #32 (Save & History)
```

**Assignee:** Developer  
**Effort:** 3 days (routing editor)  
**Label:** `feature`, `config`, `phase-1`

---

#### Issue #31: Feature: Configuration Builder — Receivers & Time Intervals
```
Title: Phase 1: Receiver form builder and time interval scheduler

Body:
## Objective
Provide form-based editors for Alertmanager receivers and time intervals, eliminating YAML editing.

## Part A: Receiver Form Builder

### Requirements
- Receiver type selector (Slack, PagerDuty, Email, Webhook, OpsGenie, VictorOps, etc.)
- Type-specific forms:
  - **Slack:** webhook URL, channel, message template
  - **PagerDuty:** integration key, severity mapping
  - **Email:** to, subject, HTML template
  - **Webhook:** URL, method (POST/PUT), headers, body
  - **OpsGenie:** API key, priority
- Field validation (required, format)
- List/manage receivers: add, edit, delete

## Part B: Time Interval Editor

### Requirements
- Time interval definition (name, timezone)
- Conditions (at least one):
  - **Times:** start/end hour:minute
  - **Weekdays:** Mon-Sun multiselect
  - **Days of month:** number or range picker
  - **Months:** Jan-Dec multiselect
  - **Years:** year input or range
- Cron-like preview: "Mon-Fri 9:00-17:00 UTC"
- List/manage intervals: add, edit, delete

## Acceptance Criteria
- [ ] Receiver form supports all receiver types
- [ ] Form validation prevents invalid fields
- [ ] Time interval builder accepts complex schedules
- [ ] Cron preview is human-readable
- [ ] Tests: form validation, time logic

## Implementation Notes
- Frontend: ReceiverForm + ReceiverList components
- Frontend: TimeIntervalEditor component
- Use Svelte form handling (forms, validation)

## Related
- #30 (Routing Editor)
- #32 (Save & History)
```

**Assignee:** Developer  
**Effort:** 3 days  
**Label:** `feature`, `config`, `phase-1`

---

#### Issue #32: Feature: Configuration Builder — Save & History
```
Title: Phase 1: Configuration save (disk/git) and change history

Body:
## Objective
Enable users to save config changes and view/rollback to previous versions.

## Requirements
- Config save modes:
  - **Disk:** Write to file, optional webhook trigger
  - **Git:** Commit + push to GitHub/GitLab
- Pre-save validation (official AM parser)
- Structured diff viewer (side-by-side YAML)
- Change history (list last N versions)
- Rollback: restore older config version
- Atomic write: no partial writes on error
- Audit logging: who, what, when, diff summary

## Save flow
1. User edits config (routing + receivers + time intervals)
2. Click "Save"
3. Validation check
4. Show diff (changes highlighted)
5. Select save mode (disk vs git)
6. Confirm → save → webhook trigger (optional)

## Git mode options
- **Branch:** target branch (e.g., main)
- **Commit message:** auto-generated or custom
- **File path:** e.g., config/alertmanager.yml

## Disk mode options
- **Path:** write location (configurable)
- **Backup:** option to backup old version

## Acceptance Criteria
- [ ] Config validation before save
- [ ] Atomic write (no partial files)
- [ ] Git mode: commit + push succeeds
- [ ] Disk mode: file written correctly
- [ ] Diff viewer shows changes clearly
- [ ] Rollback restores previous version
- [ ] Webhook triggered post-save
- [ ] Tests: atomic write, validation, git push, rollback

## Performance
- Save 5000-line YAML: < 3s

## Implementation Notes
- Backend: PUT /api/config (validate + save)
- Backend: GET /api/config/history (list versions)
- Backend: POST /api/config/rollback/{version}
- Frontend: ConfigReview page (multi-step: preview → diff → save mode → confirm)
- Frontend: DiffViewer component (side-by-side display)

## Related
- #30 (Routing Editor)
- #31 (Receivers & Time Intervals)
- #23 (Config history — overlaps, coordinate)
```

**Assignee:** Developer  
**Effort:** 4 days  
**Label:** `feature`, `config`, `phase-1`

---

## 📊 Summary Table

| Issue # | Feature | Title | Effort | Dependency | Label |
|---------|---------|-------|--------|-----------|-------|
| #25 | 1 | Alert List & Kanban Views | 5d | — | visualization |
| #26 | 1 | Alert Filtering & Grouping | 2d | #25 | feature |
| #27 | 2 | Multi-instance Aggregation | 4d | #25 (optional) | multi-instance |
| #28 | 3 | Routing Tree Visualizer | 6d | #25 (optional) | visualization |
| #29 | 4 | Silences & Bulk Actions | 5d | #25 | operations |
| #30 | 5 | Config Builder — Routing | 3d | #28 (optional) | config |
| #31 | 5 | Config Builder — Receivers/Time | 3d | #30 | config |
| #32 | 5 | Config Builder — Save/History | 4d | #30, #31 | config |

**Total effort:** 32 days (or ~6 weeks with 1 dev, ~3 weeks with 2 devs in parallel)

---

## 🔀 Suggested Execution Order

### Batch 1 (Week 1 — Foundation)
- **#25:** Alert views (Kanban + List)
- **#26:** Filtering & grouping

### Batch 2 (Weeks 2-3 — Enrichment, parallel)
- **#27:** Multi-instance (Dev 1)
- **#28:** Routing tree (Dev 2)

### Batch 3 (Week 3-4 — Operations)
- **#29:** Silences & bulk actions

### Batch 4 (Weeks 4-5 — Configuration)
- **#30:** Routing editor
- **#31:** Receivers & time intervals
- **#32:** Save & history

---

## 📝 Mapping to Existing GitHub Issues

| Phase 1 Visualization | Existing GitHub Issue | Overlap |
|------------------------|----------------------|---------|
| #25-26 (Alerts) | None (new) | — |
| #27 (Multi-instance) | None (new) | — |
| #28 (Routing tree) | None (new) | — |
| #29 (Silences) | None (new) | #4 (Ack visual, related) |
| #30-32 (Config builder) | #21 (Config editor) | Partial overlap; #21 scope unclear |

**Note:** Existing #20-24 in GitHub don't directly map to Phase 1 Visualization features. Suggest creating new #25-32 for clarity.

---

**End of Document**

Generated: 2026-03-09 | Planner Agent
