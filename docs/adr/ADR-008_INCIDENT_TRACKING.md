# ADR-008: Incident Tracking — Immutable Ledger & State Machine

**Status:** Accepted  
**Date:** 2026-03-10  
**Implemented by:** Developer Agent  
**Related issues:** Phase 1 — Dashboard-Backbone track

---

## Context

AlertLens aggregates alerts from Alertmanager but lacked a structured way to
track the *operational response* to those alerts. When an alert fires, the
on-call engineer needs to:

1. Acknowledge they've seen it (ACK)
2. Signal they're actively investigating (INVESTIGATING)
3. Record when it's resolved (RESOLVED)
4. Leave an audit trail of who did what and when

A naive mutable-record approach (single row per incident, updated in-place)
loses history and makes audit trails impossible without a separate changelog
table.

**SPECS constraint:** AlertLens must remain stateless (no database). An
in-process in-memory store satisfies the Phase 1 requirement while providing
the right abstractions for a future persistent backend.

---

## Decision

### 1. Immutable Event Ledger

Each incident maintains an **append-only** list of `IncidentEvent` records.
Events are never mutated or deleted after creation. The current status is
always derived from the most recent status-carrying event.

```
Incident
 └── Events (append-only ledger)
      ├── [1] CREATED   → status: OPEN
      ├── [2] ACK        → status: ACK
      ├── [3] COMMENT    → status: (unchanged)
      ├── [4] INVESTIGATING → status: INVESTIGATING
      └── [5] RESOLVED   → status: RESOLVED
```

**Benefits:**
- Complete audit trail with actor and message per event
- Timestamps are monotonically non-decreasing
- No UPDATE operations needed — only appends
- Duration metrics derivable from event timestamps (e.g. MTTD, MTTR)

### 2. State Machine

Valid transitions enforced at the store layer:

```
       ┌─────────────────────────────┐
       │                             ▼
OPEN ──→ ACK ──→ INVESTIGATING ──→ RESOLVED
  │       │            │              │
  │       └────────────┘              │
  │                                   │
  └──────────────── (reopen) ─────────┘
```

| From         | Allowed targets                        |
|--------------|----------------------------------------|
| OPEN         | ACK, INVESTIGATING, RESOLVED           |
| ACK          | INVESTIGATING, RESOLVED, OPEN (reopen) |
| INVESTIGATING| RESOLVED, OPEN (reopen)                |
| RESOLVED     | OPEN (reopen)                          |

Invalid transitions return HTTP 409 Conflict.

### 3. API Design

```
GET  /api/incidents              → list (paginated, filterable by status/fingerprint)
POST /api/incidents              → create (requires silencer role)
GET  /api/incidents/{id}         → full incident including timeline
GET  /api/incidents/{id}/timeline → event log only (lightweight polling)
POST /api/incidents/{id}/events  → add ACK / INVESTIGATING / RESOLVED / REOPENED / COMMENT
```

Authentication follows the existing RBAC model:
- **Read** → `viewer` role minimum
- **Write** → `silencer` role minimum (accountability requires identity)

### 4. In-Memory Store

`internal/incident.Store` uses a `sync.RWMutex`-protected map.
All mutations return deep copies so callers cannot race on the stored data.

**Trade-offs:**
- ✅ Zero external dependencies, fits stateless constraint
- ✅ Fast (all operations O(1) or O(n) at most)
- ❌ Lost on process restart — acceptable for Phase 1 MVP
- Future: swap `Store` implementation for a BoltDB / SQLite backend

### 5. Frontend Architecture

- **`$lib/api/incidents.ts`** — typed API client (fetch wrappers + convenience helpers)
- **`$lib/stores/incidents.ts`** — Svelte writable/derived stores, actions, 30s polling
- **`IncidentTimeline.svelte`** — vertical timeline, colour-coded by event kind
- **`IncidentCard.svelte`** — compact list card with quick-action buttons
- **`IncidentStatusBadge.svelte`** — animated status pill
- **`AddEventForm.svelte`** — modal form, shows only valid transitions
- **`/incidents` route** — full dashboard: list + side-panel detail + create modal

---

## Consequences

**Positive:**
- Full audit trail from day one
- State machine prevents nonsensical transitions at the API level
- Frontend enforces same transitions via AddEventForm (shows only valid options)
- Active incident count badge in Navbar gives always-visible situational awareness
- MTTD/MTTR computable from event timestamps for future SLO reporting

**Negative:**
- In-memory store resets on restart — teams must re-open incidents after redeploys
- No persistence hook yet (pluggable store interface makes this straightforward to add)

---

## Alternatives Considered

| Option | Rejected because |
|--------|-----------------|
| Mutable single-row per incident | No history, no audit trail |
| External DB (PostgreSQL) | Violates stateless constraint for Phase 1 |
| File-based JSON store | Adds I/O complexity; in-memory is sufficient for MVP |
| Reuse Alertmanager silences for ACK state | Silences are suppression tools, not incident lifecycle records |
