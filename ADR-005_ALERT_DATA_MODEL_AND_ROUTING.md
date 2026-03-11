# ADR-005: Alert Data Model & Routing/Persistence Strategy

**Status:** Proposed  
**Date:** 2026-03-10  
**Decision Maker:** Architect  
**Implementation Feature:** #1-4 (Alert Views, Multi-instance, Routing, Silences)

---

## Context

AlertLens aggregates alerts from multiple Alertmanager instances and presents them through filtering, grouping, and visualization. The system must:

1. **Represent alerts and groups consistently** across REST API responses and client state
2. **Preserve user view preferences** (filters, sorting, grouping) without server-side persistence
3. **Route alert views** using URL parameters for bookmarkability and browser history
4. **Maintain type safety** for alert matchers, labels, and filter expressions
5. **Support multi-instance operations** atomically and with graceful degradation

**Current State:**
- Alert/AlertGroup types are defined in `web/src/lib/api/types.ts`
- URL query parameters documented in `web/src/lib/api/alerts.ts` (AlertsParams)
- No formal persistence layer for user preferences
- Filtering logic scattered across frontend and backend

**Scope:**
This ADR formalizes the **alert data model**, **URL-based state routing**, and **client-side caching strategy** for Phase 1. Server-side persistence (e.g., saved views, preference snapshots) is out of scope for MVP.

---

## Design Decisions

### 1. Alert Data Model

#### Interface Definitions

**Alert (Alert Item)**
```typescript
interface Alert {
  // Unique identification
  fingerprint: string;                // Alert deduplication key (SHA256 of labels)
  alertmanager: string;              // Source Alertmanager instance name
  instance_id?: string;              // Alias for alertmanager (Feature #2+)
  
  // Content
  labels: Record<string, string>;    // Key-value labels (severity, alertname, etc.)
  annotations: Record<string, string>; // Human-readable context
  generatorURL: string;              // Link to Prometheus rule
  
  // Lifecycle
  state: string;                      // "firing" | "resolved" (AM native)
  startsAt: string;                   // ISO 8601 timestamp
  endsAt: string;                     // ISO 8601 timestamp (or "0001-01-01T..." for ongoing)
  updatedAt?: string;                 // Last state change timestamp
  
  // Routing & Suppression
  receivers: { name: string }[];      // Which receivers will handle this alert
  status: AlertStatus;                // Computed view-layer state
  
  // Optional: Acknowledgment
  ack?: Ack;                          // Visual or silence acknowledgment
}

interface AlertStatus {
  state: 'active' | 'suppressed' | 'unprocessed';  // View-layer state
  silencedBy: string[];               // Silence IDs that suppress this alert
  inhibitedBy: string[];              // Alert fingerprints that inhibit this one
}

interface Ack {
  active: boolean;
  by: string;                         // Username
  comment: string;
  silence_id: string;                 // References Silence ID if ACK is via silence
}
```

**Rationale:**
- `fingerprint` is Alertmanager's native deduplication key → use directly
- `alertmanager` (instance name) + `fingerprint` = globally unique key
- `state` is computed on backend (AM-native + view layer logic) → immutable in response
- `ack` is optional (null when no acknowledgment)
- `updatedAt` added for client-side cache invalidation (Feature #2+)

---

**AlertGroup (Grouped View)**
```typescript
interface AlertGroup {
  labels: Record<string, string>;     // Grouping label values
  alerts: Alert[];                    // Alerts matching this group
  count: number;                      // Total count in this group (for pagination)
}
```

**Rationale:**
- Groups are always **view-specific** (computed based on `groupBy` parameter)
- Labels contain only the keys requested in `group_by` parameter
- Count enables pagination within groups (future Feature #5)

---

**AlertsResponse (Paginated Results)**
```typescript
interface AlertsResponse {
  groups: AlertGroup[];
  total: number;                      // Total alerts across all groups (before pagination)
  limit: number;                      // Pagination size used
  offset: number;                     // Pagination offset used
  partial_failures?: InstanceError[]; // Degradation: per-instance errors
}

interface InstanceError {
  instance: string;
  error: string;
}
```

**Rationale:**
- Grouped response enables "group first" UI (Kanban, tree view)
- `total` allows client to calculate pagination controls
- `partial_failures` enables graceful degradation (show healthy instances while noting errors)

---

### 2. URL-Based Routing & State Preservation

**Route Pattern:**
```
/alerts?<params>
/alerts/<group-hash>  (future: detail panel route)
```

**Query Parameter Schema**

| Parameter | Type | Semantics | Encoding | Default | Notes |
|-----------|------|-----------|----------|---------|-------|
| `filter` | `string[]` | Alertmanager matcher expression(s) | Repeated param | `[]` | Format: `severity="critical"`, `alertname=~"Foo.*"` |
| `instance` | `string` | Filter to single Alertmanager | Query string | Unset (all) | URL-safe instance name |
| `silenced` | `boolean` | Include silenced alerts | Literal: `true`/`false` | Unset (all) | Filters `status.silencedBy` |
| `inhibited` | `boolean` | Include inhibited alerts | Literal: `true`/`false` | Unset (all) | Filters `status.inhibitedBy` |
| `active` | `boolean` | Include active alerts only | Literal: `true`/`false` | Unset (all) | Filters by `status.state === 'active'` |
| `severity` | `string[]` | View-layer severity filter | Repeated param | `[]` | Values: `critical`, `warning`, `info` |
| `status` | `string[]` | View-layer state filter | Repeated param | `[]` | Values: `active`, `suppressed`, `unprocessed` |
| `group_by` | `string[]` | Grouping dimensions | Repeated param | `[]` | Label keys to group by (e.g., `group_by=severity&group_by=alertname`) |
| `limit` | `number` | Pagination size | Numeric | `500` | Range: 1–1000 |
| `offset` | `number` | Pagination offset | Numeric | `0` | Must be `<= total` |
| `sort_by` | `string` | Sort key (future Feature #4) | `alertname`, `severity`, `startsAt` | `startsAt:desc` | Reserved for future |

---

**URL Construction Examples**

**Example 1: Critical alerts, grouped by severity**
```
/alerts?status=active&severity=critical&group_by=severity&limit=50
```
- Fetch only active, critical alerts
- Group by `severity` label
- Return 50 per page

**Example 2: Silenced alerts by instance**
```
/alerts?instance=prometheus-prod&silenced=true&group_by=alertname&offset=50&limit=25
```
- Only alerts from `prometheus-prod`
- Include silenced alerts
- Group by `alertname`, pagination offset 50

**Example 3: Complex matcher + grouping**
```
/alerts?filter=severity="critical"&filter=env="prod"&group_by=service&group_by=severity&limit=100
```
- Alertmanager-native matcher: `severity="critical"` AND `env="prod"`
- Two-level grouping
- Flatten to 100-alert pages

---

**Query Parameter Encoding Rules**

```typescript
// Standard URL encoding
encodeURIComponent(value)  // "my filter" → "my%20filter"

// Repeated params (use URLSearchParams)
const params = new URLSearchParams();
params.append('filter', 'severity="critical"');
params.append('filter', 'env="prod"');
params.append('group_by', 'severity');
params.append('group_by', 'instance');

// URL: /alerts?filter=...&filter=...&group_by=...&group_by=...

// Parsing (browser API)
const q = new URLSearchParams(window.location.search);
const filters = q.getAll('filter');  // Returns string[]
const groupBy = q.getAll('group_by');
```

**Rationale:**
- URLSearchParams is native & standard across browsers
- Repeated params (`filter=...&filter=...`) are HTTP standard
- No custom encoding needed; `encodeURIComponent` handles special characters in matchers

---

### 3. State Management: Client-Side Routing

**No server-side session/persistence in MVP.**

**Client-side pattern:**

```typescript
// Route: /alerts?filter=...&group_by=...
// SvelteKit handles URL ↔ component sync

// In +page.svelte (Svelte routing)
import { page } from '$app/stores';

let filters: string[] = [];
let groupBy: string[] = [];

// Subscribe to URL changes
page.subscribe(p => {
  const q = new URLSearchParams(p.url.search);
  filters = q.getAll('filter');
  groupBy = q.getAll('group_by');
  
  // Trigger API fetch with new params
  loadAlerts({ filter: filters, groupBy: groupBy });
});

// When user updates filters via UI:
function applyFilters(newFilters: string[]) {
  const q = new URLSearchParams();
  newFilters.forEach(f => q.append('filter', f));
  groupBy.forEach(g => q.append('group_by', g));
  
  // SvelteKit pushes new URL to browser history
  goto(`/alerts?${q.toString()}`);
}
```

**Why URL-based routing?**
1. **Bookmarkable:** Share filtered views by link
2. **Browser history:** Back/forward buttons work
3. **Shareable:** Link to others: `"Check out these prod errors: /alerts?instance=prometheus-prod&status=active"`
4. **Stateless:** No server-side sessions needed
5. **Cache-friendly:** Same URL = same response (HTTP caching enabled)

---

### 4. Client-Side Caching Strategy

**Three-tier caching:**

```typescript
// Tier 1: In-memory view store
export const alertsStore = writable<AlertsResponse>({
  groups: [],
  total: 0,
  limit: 500,
  offset: 0,
});

// Tier 2: localStorage (30 min TTL)
function cacheResponse(key: string, data: AlertsResponse, ttlMs = 30 * 60 * 1000) {
  localStorage.setItem(key, JSON.stringify({
    data,
    timestamp: Date.now() + ttlMs,
  }));
}

function getCachedResponse(key: string): AlertsResponse | null {
  const cached = localStorage.getItem(key);
  if (!cached) return null;
  
  const { data, timestamp } = JSON.parse(cached);
  if (Date.now() > timestamp) {
    localStorage.removeItem(key);
    return null;  // Expired
  }
  
  return data;
}

// Tier 3: Server polling (5s intervals, with If-None-Match ETag)
async function fetchAlerts(params: AlertsParams) {
  const key = JSON.stringify(params);  // Cache key
  
  // Check local cache first
  let cached = getCachedResponse(key);
  if (cached) {
    alertsStore.set(cached);
    return;
  }
  
  // Fetch from server
  const response = await fetch(`/api/alerts?${queryString(params)}`, {
    headers: {
      // Send ETag if available (304 Not Modified = skip update)
      'If-None-Match': sessionStorage.getItem(`etag:${key}`),
    },
  });
  
  if (response.status === 304) {
    // Server returned 304: use cached data
    cached = getCachedResponse(key);
    if (cached) alertsStore.set(cached);
    return;
  }
  
  const data = await response.json();
  
  // Update all three tiers
  cacheResponse(key, data);
  sessionStorage.setItem(`etag:${key}`, response.headers.get('ETag'));
  alertsStore.set(data);
}
```

**Cache Key Strategy:**
- Serialize all params to deterministic string: `JSON.stringify(sortByKey(params))`
- Example: `{"filter":["a=b"],"groupBy":["severity"],"limit":500}` (sorted keys)

**TTL Rules:**
- **Local cache (localStorage):** 30 min (user may return after break)
- **Session cache (sessionStorage):** Lifetime of browser tab
- **Polling interval:** 5 sec (Feature #4: real-time; configurable)

---

### 5. Matcher Validation & Type Safety

**Matcher structure:**
```typescript
interface Matcher {
  name: string;       // Label key: "severity", "alertname", etc.
  value: string;      // Label value or regex pattern
  isRegex: boolean;   // Prometheus regex syntax (e.g., "Foo.*")
  isEqual: boolean;   // true: `name="value"` | false: `name!="value"`
}
```

**URL filter format (Alertmanager matcher syntax):**
```
severity="critical"      (exact match)
severity!="warning"      (not equal)
env=~"prod-.*"          (regex match)
service!~"test.*"       (not regex match)
```

**Client-side validation:**
```typescript
import { z } from 'zod';

const MatcherSchema = z.object({
  name: z.string().min(1).max(200),  // Label key
  value: z.string().max(500),        // Value (allow empty for regex)
  isRegex: z.boolean(),
  isEqual: z.boolean(),
});

const AlertsParamsSchema = z.object({
  filter: z.array(z.string()).optional(),
  instance: z.string().optional(),
  silenced: z.boolean().optional(),
  inhibited: z.boolean().optional(),
  active: z.boolean().optional(),
  severity: z.array(z.string()).optional(),
  status: z.enum(['active', 'suppressed', 'unprocessed']).array().optional(),
  groupBy: z.array(z.string()).optional(),
  limit: z.number().int().min(1).max(1000).optional(),
  offset: z.number().int().min(0).optional(),
});

// Validate params before API call
const params = AlertsParamsSchema.parse(urlQueryParams);
```

**Server-side validation (Go backend):**
```go
type AlertsParams struct {
    Filter   []string `query:"filter"`
    Instance string   `query:"instance"`
    Silenced *bool    `query:"silenced"`
    GroupBy  []string `query:"group_by"`
    Limit    int      `query:"limit" validate:"min=1,max=1000"`
    Offset   int      `query:"offset" validate:"min=0"`
}

// In handler:
if err := validate.Struct(params); err != nil {
    return c.JSON(400, map[string]string{"error": err.Error()})
}
```

---

### 6. Multi-Instance Aggregation

**Atomic Semantics:**

When an alert is present in 2+ Alertmanager instances:
- **Fingerprints are global** (same alert + labels = same fingerprint)
- **Deduplicate by fingerprint** (show once, list all instances in `receivers`)
- **Suppress by any instance:** If silenced in one AM, shows as `silencedBy` in aggregated view

**Example:**
```json
{
  "fingerprint": "abc123",
  "labels": {"alertname": "DiskFull", "instance": "server1"},
  "alertmanager": "prometheus-prod",  // Primary instance
  "receivers": [
    {"name": "pagerduty"},
    {"name": "slack"}
  ],
  "status": {
    "state": "suppressed",
    "silencedBy": ["silence-xyz"]
  }
}
```

**Graceful Degradation (Partial Failures):**
```json
{
  "groups": [{ ... }],
  "total": 1500,
  "partial_failures": [
    {
      "instance": "prometheus-dr",
      "error": "connection timeout"
    }
  ]
}
```

**Client Behavior:**
- Show alerts from healthy instances
- Display banner: "⚠ 1 instance unavailable (prometheus-dr)"
- Do not fail entire response

---

## Persistence Strategy: MVP (No Server-Side State)

**Phase 1 (MVP): No persistence.**

Rationale:
1. Simplifies backend (stateless HTTP)
2. URL-based routing = sufficient for user needs
3. Can add server-side saved views in Phase 2 without breaking API
4. localStorage provides lightweight client-side cache

**Future (Phase 2+): Optional Persistence**

When needed:
```typescript
interface SavedView {
  id: string;
  name: string;
  description: string;
  params: AlertsParams;
  createdAt: string;
  updatedAt: string;
  tags: string[];  // "critical", "prod", etc.
}

// Endpoints (not in Phase 1)
POST   /api/views       // Save current view
GET    /api/views       // List saved views
GET    /api/views/:id   // Load view
PUT    /api/views/:id   // Update view
DELETE /api/views/:id   // Delete view
```

**Storage:**
- Database (same as alerts): timestamp-based cleanup (30 day retention)
- User association: via session/JWT token
- No ACL needed for MVP (all users see all views)

---

## Matcher Filter Encoding Details

**Filter Parameter Format (Alertmanager Matcher Syntax):**

The `filter` parameter accepts **Prometheus matcher expressions**:

```
severity="critical"          # Literal match
env!="test"                  # Not equal
service=~"web-.*"            # Regex match
cluster!~"local|dev"         # Regex not match
```

**Implementation in Frontend:**
```typescript
// Parse filter string into Matcher objects (optional, for UI)
function parseFilter(filter: string): Matcher | null {
  const regex = /^(\w+)(=~|!=|=|!~)(.+)$/;
  const match = filter.match(regex);
  
  if (!match) return null;
  
  const [, name, op, value] = match;
  return {
    name,
    value,
    isRegex: op.includes('~'),
    isEqual: op.startsWith('='),
  };
}

// Reconstruct filter string from Matcher
function stringifyMatcher(m: Matcher): string {
  const op = m.isRegex
    ? (m.isEqual ? '=~' : '!~')
    : (m.isEqual ? '=' : '!=');
  return `${m.name}${op}${m.value}`;
}
```

**Validation Rules:**
- Label name: `[a-zA-Z_][a-zA-Z0-9_]*` (Prometheus label name syntax)
- Value: Any UTF-8 string; regex must be valid RE2 syntax
- Max filter length: 500 characters
- Max filter count: 10 per request (to prevent DoS)

---

## Type Safety & Codec Strategy

**Use Zod for runtime validation:**
- Frontend: Validate URL params before API call
- Backend: Validate request body
- Shared types: TypeScript interfaces + Zod schemas

**No custom codecs needed** (standard JSON + URL encoding).

**Error Handling:**
```typescript
try {
  const params = AlertsParamsSchema.parse(queryParams);
  const response = await api.get('/alerts', { params });
} catch (error) {
  if (error instanceof z.ZodError) {
    // Validation error: highlight invalid fields to user
    console.error('Invalid filter:', error.issues);
  } else {
    // Network error
    console.error('Fetch error:', error);
  }
}
```

---

## API Endpoint Specification

**GET /api/alerts**

**Request:**
```
Query Parameters: (all optional)
  filter[]=severity="critical"   (repeatable)
  instance=prometheus-prod
  silenced=true
  inhibited=false
  active=true
  severity[]=critical,warning    (repeatable)
  status[]=active                (repeatable)
  group_by[]=severity            (repeatable)
  limit=500
  offset=0
```

**Response (200 OK):**
```json
{
  "groups": [
    {
      "labels": { "severity": "critical" },
      "alerts": [ ... ],
      "count": 42
    }
  ],
  "total": 128,
  "limit": 500,
  "offset": 0,
  "partial_failures": [
    {
      "instance": "prometheus-dr",
      "error": "connection refused"
    }
  ]
}
```

**Response (400 Bad Request):**
```json
{
  "error": "invalid query parameter",
  "details": "limit must be <= 1000"
}
```

**Response (429 Too Many Requests):**
```json
{
  "error": "rate limited",
  "retry_after": 60
}
```

---

## Testing Strategy

### Unit Tests (Frontend)
- Matcher parsing: `parseFilter()` → Matcher object
- URL serialization: `AlertsParams` → query string
- Cache validation: expired entries removed
- Zod schema validation: invalid params rejected

### Integration Tests
- End-to-end filter flow: UI → URL update → API call → response handling
- Pagination: `offset` + `limit` behaves correctly
- Multi-instance degradation: partial failures display correctly
- Grouping: `group_by=severity,instance` produces correct groups

### E2E Tests (Playwright)
- User applies filter via UI → URL updates
- User clicks back button → filters restored
- Share link with others → same view appears
- Instance goes offline → warning banner displays

---

## Security Considerations

1. **YAML Injection (Config Filter):** Not applicable to alerts filter (uses Prometheus matcher syntax)
2. **Regex DoS:** Limit regex length to 100 chars; validate RE2 syntax on backend
3. **Query Param Injection:** Use URL encoding; no custom parsing
4. **XSS in Labels:** HTML-encode all alert label values before rendering
5. **CSV Export (Future):** Sanitize labels for CSV format

---

## Success Criteria

- [ ] `Alert` and `AlertGroup` interfaces finalized and documented
- [ ] URL query parameter schema defined (filter, groupBy, pagination, etc.)
- [ ] Zod schemas created for runtime validation
- [ ] Cache strategy implemented (localStorage 30-min TTL)
- [ ] Multi-instance deduplication working
- [ ] Graceful degradation on partial failures
- [ ] Unit tests for matcher parsing & URL encoding (>90% coverage)
- [ ] E2E tests for filter → URL → API flow
- [ ] Documented in code (inline comments + README)
- [ ] No breaking API changes when pivoting to server-side persistence (Phase 2)

---

## Migration Path (Phase 2+)

**To add server-side persistence without breaking clients:**

1. Add optional `view_id` parameter:
   ```
   GET /api/alerts?view_id=saved-view-123
   ```
   Backend resolves view_id → expanded params

2. New endpoints for saved views (non-breaking):
   ```
   POST /api/views           # Save current params as named view
   GET  /api/views           # List saved views
   PUT  /api/views/:id       # Update view
   DELETE /api/views/:id     # Delete view
   ```

3. Clients that understand `view_id` use it; older clients continue with URL params

---

## Dependency Analysis

- **Zod:** Runtime schema validation (already in dependencies)
- **SvelteKit:** Page route + stores (already in dependencies)
- **URLSearchParams:** Native browser API (no dependency)
- **localStorage:** Native browser API (no dependency)

---

## Related Documents

- [ADR-001](./ADR-001_ROUTING_TREE_VISUALIZATION.md): Routing tree visualization (separate concern)
- [ADR-003](./ADR-003_CONFIG_STORAGE_STRATEGY.md): Config storage (different from alert routing)
- [ARCHITECTURE_DESIGN_PHASE_1](./ARCHITECTURE_DESIGN_PHASE_1.md): Full system overview

---

## Sign-off

- **Architect:** ✅ Ready for formal review
- **Backend Developer:** ⬜ Pending implementation
- **Frontend Developer:** ⬜ Pending implementation
- **Security:** ⬜ Pending review
- **QA:** ⬜ Pending test planning

---

**End of ADR-005**
