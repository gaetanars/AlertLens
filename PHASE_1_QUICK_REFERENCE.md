# ⚡ Phase 1 Visualization — Quick Reference

**One-page reference for planning, tracking, and daily standup.**

---

## 📊 Feature Matrix

```
┌────────────────────────────────────────────────────────────────────────┐
│ FEATURE 1: Alert Kanban/List Views                                    │
├────────────────────────────────────────────────────────────────────────┤
│ ⭐ FOUNDATION (blocking all others)                                    │
│ Duration: 5 days                                                       │
│ Depends: None                                                          │
│ Blockers for: #2, #3, #4, #5                                          │
│                                                                        │
│ Tasks:                                                                 │
│  ✓ Backend: GET /api/alerts (grouping, filtering)                    │
│  ✓ Frontend: Kanban board (CSS Grid, severity columns)               │
│  ✓ Frontend: List table (sort, pagination)                           │
│  ✓ Frontend: Filter builder UI                                       │
│  ✓ Tests: Filter logic, grouping, responsive design                 │
│                                                                        │
│ GitHub Issues: #25 (Views), #26 (Filtering)                          │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│ FEATURE 2: Multi-instance Aggregation                                 │
├────────────────────────────────────────────────────────────────────────┤
│ ⭐⭐ PARALLEL (can start after #1)                                     │
│ Duration: 4 days                                                       │
│ Depends: #1 (optional, improves with #1 done)                        │
│ Blockers for: None (enrichment feature)                               │
│                                                                        │
│ Tasks:                                                                 │
│  ✓ Backend: Concurrent Alertmanager pool fetch                       │
│  ✓ Backend: Alert instance metadata (label each alert)               │
│  ✓ Frontend: Instance selector dropdown                              │
│  ✓ Frontend: Instance badge on alerts                                │
│  ✓ Tests: Concurrent fetch, timeout handling, error recovery        │
│                                                                        │
│ GitHub Issue: #27                                                     │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│ FEATURE 3: Routing Tree Visualizer                                    │
├────────────────────────────────────────────────────────────────────────┤
│ ⭐⭐ PARALLEL (can start after #1)                                     │
│ Duration: 6 days                                                       │
│ Depends: #1 (optional, improves with #1 done)                        │
│ Blockers for: #5 (Config Builder, optional)                          │
│                                                                        │
│ Tasks:                                                                 │
│  ✓ Backend: GET /api/routing-tree (parse config, build tree JSON)   │
│  ✓ Backend: GET /api/routing-tree/node/{id}/alerts (node matching) │
│  ✓ Frontend: D3.js tree visualization (hierarchy layout)             │
│  ✓ Frontend: Node detail panel (properties, matching alerts)        │
│  ✓ Frontend: Zoom & pan controls                                    │
│  ✓ Tests: Tree parsing, node matching, D3 rendering, performance   │
│                                                                        │
│ GitHub Issue: #28                                                     │
│ Tech decision: D3.js (vs. Cytoscape, ELK) → ADR needed              │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│ FEATURE 4: Silences + Bulk Actions                                    │
├────────────────────────────────────────────────────────────────────────┤
│ ⭐⭐⭐ SEQUENTIAL (starts after #1 complete)                           │
│ Duration: 5 days                                                       │
│ Depends: #1 (alert selection, display)                               │
│ Blockers for: #5 (Config Builder)                                    │
│                                                                        │
│ Tasks:                                                                 │
│  ✓ Backend: POST /api/silences (create with matchers, duration)     │
│  ✓ Backend: GET /api/silences (list with filters)                   │
│  ✓ Backend: DELETE /api/silences/{id} (expire)                      │
│  ✓ Backend: POST /api/actions/bulk-silence                          │
│  ✓ Frontend: SilenceForm (duration picker, matcher editor)          │
│  ✓ Frontend: SilencesList (manage active/expired)                   │
│  ✓ Frontend: Bulk action selector (checkboxes)                      │
│  ✓ Tests: Silence CRUD, bulk operations, matcher logic              │
│                                                                        │
│ GitHub Issue: #29                                                     │
└────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────┐
│ FEATURE 5: Configuration Builder                                      │
├────────────────────────────────────────────────────────────────────────┤
│ ⭐⭐⭐⭐⭐ SEQUENTIAL (starts after #4 complete)                         │
│ Duration: 10 days (can be split across 2 devs)                       │
│ Depends: #4 (save pattern, form framework)                           │
│ Blockers for: Phase 1 release (largest feature)                      │
│                                                                        │
│ Sub-features (can be parallelized):                                   │
│  ▪ Routing Editor (#30, 3d)      [Dev 1]                             │
│  ▪ Receivers & Time Intervals (#31, 3d) [Dev 2]                     │
│  ▪ Save & History (#32, 4d)      [Dev 1+2]                           │
│                                                                        │
│ Tasks:                                                                 │
│  ✓ Backend: GET /api/config (parse to JSON)                          │
│  ✓ Backend: POST /api/config/preview (validate + diff)               │
│  ✓ Backend: PUT /api/config (atomic save, disk/git)                 │
│  ✓ Backend: GET /api/config/history (versions + rollback)           │
│  ✓ Frontend: RouteBuilder (nested forms for routes)                 │
│  ✓ Frontend: ReceiverForm (type-specific receiver forms)            │
│  ✓ Frontend: TimeIntervalEditor (schedule builder)                  │
│  ✓ Frontend: YAMLPreview (syntax-highlighted)                       │
│  ✓ Frontend: DiffViewer (side-by-side changes)                      │
│  ✓ Frontend: ConfigReview (multi-step save flow)                    │
│  ✓ Tests: Config validation, YAML diff, atomic write, git push     │
│                                                                        │
│ GitHub Issues: #30 (Routing), #31 (Receivers/Time), #32 (Save)      │
│ Tech decisions: Form framework (Svelte/FormKit?), storage strategy   │
│ Security: YAML injection prevention (official parser)                │
│ Coordination: Requires RBAC (#24) middleware ready                   │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 🔗 Dependency Graph (ASCII)

```
START
  │
  ├─→ WEEK 1: Feature #1 (5d) ─┐
  │                             │
  │                    ┌────────┴────┐
  │                    │             │
  │   WEEK 2-3:   ┌─→ #2 (4d) ─┐    │
  │   Parallel    │            │    │
  │               └─→ #3 (6d) ─┤    │
  │                            │    │
  │        After #1 (dependent)│    │
  │                            └────┤
  │                                 ↓
  │            WEEK 3-4: Feature #4 (5d) ─┐
  │                                       │
  │                                       ↓
  │            WEEK 4-5: Feature #5 (10d)
  │                 #30 + #31 + #32
  │
  └──────────────────────────────────────────→ COMPLETE (Phase 1 MVP)


CRITICAL PATH: #1 → #4 → #5 = 20 days minimum
PARALLEL GAINS: #2 || #3 = 6 days (vs 10 sequential) = saves 4 days
TOTAL: 30 days with 1 dev, ~15-18 days with 2 devs (parallel)
```

---

## 📅 3-Week Sprint Plan (2 Developers)

### Sprint 1: Week 1 (Days 1-5)
**Goal:** Feature #1 (Alert Views) MVP ready

| Dev | Task | Days | Status |
|-----|------|------|--------|
| 1 | GET /api/alerts + grouping | 1 | ⬜ |
| 1 | Kanban board component | 1.5 | ⬜ |
| 1 | List table component | 0.5 | ⬜ |
| 1 | Filter builder UI | 1 | ⬜ |
| 1 | Tests + responsive | 1 | ⬜ |
| 2 | Review & merge | 0.5 | ⬜ |

**Deliverable:** MVP alert views with filtering/grouping

---

### Sprint 2: Weeks 2-3 (Days 6-15)
**Goal:** Features #2, #3, #4 (partial) ready

| Dev | Task | Days | Status |
|-----|------|------|--------|
| 1 | GET /api/silences + CRUD | 0.75 | ⬜ |
| 1 | SilenceForm component | 1.5 | ⬜ |
| 1 | SilencesList + bulk selector | 1.5 | ⬜ |
| 1 | Tests + merge | 0.75 | ⬜ |
| 2 | Concurrent pool fetch (#2) | 1 | ⬜ |
| 2 | Instance selector (#2) | 0.75 | ⬜ |
| 2 | Routing tree parser (#3) | 1 | ⬜ |
| 2 | D3 visualization (#3) | 2 | ⬜ |
| 2 | Node detail + matching (#3) | 1 | ⬜ |
| 2 | Tests + merge | 1.5 | ⬜ |

**Deliverables:** Feature #4 (silences), Feature #2, Feature #3

---

### Sprint 3: Weeks 4-5 (Days 16-25+)
**Goal:** Feature #5 (Config Builder) MVP ready

| Dev | Task | Days | Status |
|-----|------|------|--------|
| 1 | Routing editor form | 2 | ⬜ |
| 1 | Config endpoints (GET/POST/PUT) | 1 | ⬜ |
| 2 | Receivers form | 1.5 | ⬜ |
| 2 | Time intervals editor | 1.5 | ⬜ |
| 1 | YAML preview + diff viewer | 1.5 | ⬜ |
| 1 | Config review flow | 1 | ⬜ |
| 1,2 | Config save (disk/git) | 1 | ⬜ |
| 1,2 | Tests + merge | 2 | ⬜ |

**Deliverable:** Feature #5 MVP, all 5 features complete

---

## 📋 Issue Creation Checklist

```
Phase 1 Visualization Issues to Create:

[ ] #25 Feature: Alert List & Kanban Views
[ ] #26 Feature: Alert Filtering & Grouping
[ ] #27 Feature: Multi-instance Aggregation
[ ] #28 Feature: Routing Tree Visualizer
[ ] #29 Feature: Silences & Bulk Actions
[ ] #30 Feature: Config Builder — Routing Editor
[ ] #31 Feature: Config Builder — Receivers & Time Intervals
[ ] #32 Feature: Config Builder — Save & History

Label all with: phase-1, visualization (where applicable)
```

---

## 🎯 Technical Decisions Needed

| Decision | Question | Options | Default Rec. | ADR? |
|----------|----------|---------|------|-----|
| **D3.js** | Routing tree lib? | D3 / Cytoscape / ELK | D3.js | ✅ |
| **Forms** | Form framework? | Custom / @sveltejs/form / FormKit | Custom + Zod | ✅ |
| **Config storage** | Rollback via? | Git history / file backups | Git history (git mode), backups (disk) | ✅ |
| **Real-time** | WebSocket or polling? | WebSocket / Polling | Polling v1 | ❌ |
| **RBAC timing** | When to start Config Builder? | After #24 / in parallel | After #24 | ✅ |

---

## 🚨 Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|-----------|------------|
| D3 learning curve | +2-3 days | Medium | Pair program, allocate time |
| Config form complexity | +3-4 days | Medium | Break into sub-features (#30, #31, #32) |
| YAML injection (security) | High | Low | Use official parser, validate input |
| Multi-instance error handling | Partial feature | Low | Graceful degradation, thorough testing |
| Large alert performance | Poor UX | Low | Pagination, optimization as needed |

---

## ✅ Definition of Done (per Feature)

```
For each feature (e.g., #25):

Frontend:
  ✓ Component renders correctly (desktop + mobile)
  ✓ Responsive design (breakpoints tested)
  ✓ Accessibility (WCAG 2.1 AA minimum)
  ✓ No console errors/warnings
  ✓ TypeScript strict mode (if applicable)

Backend:
  ✓ Endpoints implemented + tested
  ✓ Error handling graceful
  ✓ Input validation complete
  ✓ Security review passed

Tests:
  ✓ Unit tests: ≥80% coverage
  ✓ Integration tests: key workflows
  ✓ Visual regression: component renders
  ✓ Performance: benchmarks met

Code Quality:
  ✓ Linter passes (go vet, svelte-check)
  ✓ Code review: 2 approvals
  ✓ No high-severity findings

Documentation:
  ✓ API docs updated
  ✓ Inline comments for complex logic
  ✓ User guide (if new feature)
```

---

## 📞 Daily Standup Template

```
🎯 PHASE 1 VISUALIZATION STANDUP

Feature: [#25-#32]
Assignee: [Dev name]
Status: [On track / At risk / Complete]

✅ Completed yesterday:
  - Task X
  - Task Y

🔄 Working on today:
  - Task Z

🚧 Blockers / Help needed:
  - (None / Issue X / Question Y)

📊 Health: Green / Yellow / Red
```

---

## 📚 Document Cross-references

| Document | Purpose | Link |
|----------|---------|------|
| **PHASE_1_VISUALIZATION_PLAN.md** | Full detailed plan (40 KB) | Complete decomposition |
| **PHASE_1_GITHUB_ISSUES.md** | Issues + templates (14 KB) | Ready to create #25-32 |
| **PHASE_1_EXECUTIVE_SUMMARY.md** | C-level overview | Decisions + timeline |
| **PHASE_1_QUICK_REFERENCE.md** | This file | Daily reference |

---

## 🚀 Go/No-Go Checklist

Before starting development:

- [ ] Gaëtan approves roadmap & priorities
- [ ] Architect reviews plan + makes tech decisions
- [ ] Architect publishes ADRs (D3, forms, storage)
- [ ] Security approves YAML injection mitigation
- [ ] GitHub issues #25-#32 created
- [ ] Dev environment ready (docker-compose, make dev-backend/frontend)
- [ ] 2 developers assigned + scheduled
- [ ] Demo date scheduled (end of Sprint 3)

---

**Generated:** 2026-03-09 | Planner Agent  
**Status:** ✅ Ready for Architect Review & Approval
