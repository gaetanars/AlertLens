# ADR-007 — Silences + Bulk Actions

**Status:** Accepted  
**Date:** 2026-03-10  
**Deciders:** AlertLens core team  

---

## Context

AlertLens already exposes individual silence CRUD at `/api/silences` (GET, POST, PUT, DELETE).
Users on NOC dashboards need to silence **multiple alerts at once** without opening a form for
each one.  A bulk-action pattern also benefits ack workflows (visual acks).

Two failure modes must be avoided:

1. **Over-silencing** — a naive "one silence per alert" approach creates noise in the
   Alertmanager silence list and makes it hard to manage later.
2. **Under-silencing** — computing matchers too strictly leaves some alerts unsilenced.

### Related prior art in the UI

`AlertBulkActions.svelte` already computes a client-side intersection of common labels
and passes them to `SilenceForm` for the user to review.  That flow is good for
customisation but requires too many clicks for the common "quiet this alert storm now"
scenario.

---

## Decision

### 1. New API endpoint: `POST /api/v1/bulk`

A single endpoint handles both bulk silence and bulk visual-ack operations.

#### Request

```json
{
  "action": "silence",
  "alerts": [
    {
      "fingerprint": "abc123",
      "alertmanager": "prod",
      "labels": { "alertname": "HighCPU", "severity": "critical", "env": "prod" }
    },
    {
      "fingerprint": "def456",
      "alertmanager": "prod",
      "labels": { "alertname": "HighCPU", "severity": "critical", "env": "staging" }
    }
  ],
  "ends_at": "2026-03-10T23:00:00Z",
  "created_by": "alice",
  "comment": "Maintenance window"
}
```

| Field        | Type      | Required | Default          | Notes                         |
|--------------|-----------|----------|------------------|-------------------------------|
| `action`     | string    | ✅        | —                | `"silence"` or `"ack"`        |
| `alerts`     | array     | ✅        | —                | At least one item required    |
| `ends_at`    | datetime  | ❌        | `now + 1 hour`   | Must be in the future         |
| `created_by` | string    | ❌        | `"alertlens"`    |                               |
| `comment`    | string    | ❌        | `"Bulk silenced…"` |                             |

#### Response (201 Created)

```json
{
  "silence_ids": ["id1"],
  "strategy": "merged",
  "count": 1
}
```

| Field         | Notes                                                         |
|---------------|---------------------------------------------------------------|
| `silence_ids` | All Alertmanager silence IDs created                          |
| `strategy`    | `"merged"` or `"individual"` (see Smart Merge below)         |
| `count`       | `len(silence_ids)`                                            |

### 2. Smart Merge algorithm

Alerts are first **grouped by Alertmanager instance**.  Within each group:

1. **Compute label intersection** — collect labels whose key _and_ value are identical
   across **every** alert in the group.  Internal/meta labels (`alertlens_*`, `__name__`)
   are excluded.
2. **Merged path** (common matchers found):  
   Create **one** silence with the intersected equality matchers.  
   Sets `strategy = "merged"`.
3. **Individual fallback** (no common matchers):  
   Create one silence per alert using all its labels as equality matchers.  
   Sets `strategy = "individual"`.

The heuristic intentionally errs toward **fewer, broader silences** because:

- A merged silence is easier to audit and expire.
- Alertmanager de-duplicates by matcher set; the merged silence will match new
  instances of the same alert storm without extra configuration.
- Users who need precision can still use the `SilenceForm` dialog.

#### Example

```
Alert A labels: { alertname=HighCPU, severity=critical, env=prod,   pod=app-1 }
Alert B labels: { alertname=HighCPU, severity=critical, env=staging, pod=app-2 }

Intersection:   { alertname=HighCPU, severity=critical }
→ ONE merged silence: alertname="HighCPU", severity="critical"
```

### 3. Authorization

The `/api/v1/bulk` endpoint requires the `silencer` role (same as `POST /api/silences`).

### 4. Frontend UX

`AlertBulkActions.svelte` gains a **"Quick Silence (1 h)"** button that:

1. Reads the selected alerts from the `selectedFingerprints` + `alerts` stores.
2. Calls `POST /api/v1/bulk` with `ends_at = now + 1h`.
3. Shows a spinner and toast (success / error).
4. Clears the selection and refreshes alerts on success.

The existing **"Bulk silence (customize)"** button (opens `SilenceForm` pre-filled
with client-computed common matchers) is preserved for users who want control over
matchers, duration, or comment.

---

## Consequences

### Positive

- One-click bulk silencing reduces operational friction significantly.
- Smart Merge minimises silence list pollution.
- Stateless: no DB required; all state lives in Alertmanager.
- Consistent with existing silence CRUD patterns.

### Negative / Trade-offs

- Merged matchers may silence more alerts than intended (broader match).
  Mitigated by the "customize" flow and the fact that silences are always
  time-bounded.
- Individual fallback can create many silences when alerts have no common labels.
  Rare in practice (alert storms usually share `alertname`).

### Neutral

- Frontend still computes client-side intersection for the form-based flow;
  this is a different (UI-only) code path and is unaffected by this ADR.

---

## Alternatives Considered

| Option | Rejected because |
|--------|-----------------|
| Extend `POST /api/silences` with an `alert_ids` array | Mixes concerns; the current endpoint maps 1-to-1 with Alertmanager's API |
| Always create individual silences | Pollutes the silence list |
| Use Alertmanager's inhibit rules | Persistent config change, too heavy for temporary silencing |
