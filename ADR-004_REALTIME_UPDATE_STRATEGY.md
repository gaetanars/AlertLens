# ADR-004: Real-time Update Strategy

**Status:** Approved (Polling for MVP)  
**Date:** 2026-03-09  
**Decision Maker:** Architect  
**Implementation Features:** All (for live alert/silence updates)

---

## Context

Phase 1 features display real-time data (alerts, silences, routing tree). Users expect:

1. **Alert updates:** New/resolved alerts appear without page refresh
2. **Silence status:** Created silences reflected immediately on alert list
3. **Silence expiry:** Expired silences automatically fade from display
4. **Routing tree:** Reflect config changes (post-Feature #5)

**Current approach:** All data fetched from backend API (stateless).

**Problem:** Frontend doesn't know when backend data changes → users must manually refresh or wait for polling.

**Requirements:**
- Display updates within acceptable latency (5-30 seconds acceptable for MVP)
- Work in containerized/scalable environment
- No client-side persistence (stateless)
- Simple implementation (Phase 1 timeline constraint)
- Extensible to WebSocket later

**Constraints:**
- Cannot require external messaging system (Kafka, Redis)
- Should not add significant server-side state
- Must work with horizontal scaling (multiple instances)

---

## Options Considered

### Option A: Client-side Polling (Recommended for MVP)

**Description:**

Frontend periodically polls backend API endpoints:

```
GET /api/alerts (every 5s)
GET /api/silences (every 5s)
GET /api/routing-tree (every 30s, if config watching enabled)
```

**Mechanism:**

```typescript
// In Svelte store (alertsStore)
setInterval(async () => {
  const data = await fetch('/api/alerts');
  alertsStore.set(data);
}, 5000); // 5 second poll interval
```

**Pros:**
- ✅ **Simple:** No server-side changes needed
- ✅ **Scalable:** Works with multiple backend instances (no session state)
- ✅ **Reliable:** HTTP is well-understood, no connection issues
- ✅ **Browser-compatible:** Works everywhere (no WebSocket support needed)
- ✅ **Fast implementation:** Can be added in parallel with features
- ✅ **Easy debugging:** Can inspect HTTP requests in dev tools
- ✅ **Works offline/degraded:** Graceful fallback

**Cons:**
- ❌ **Latency:** 5-second delay before seeing new alerts (acceptable for MVP)
- ❌ **Bandwidth:** More requests sent (mitigated by short payload for /api/alerts)
- ❌ **Server load:** Multiple concurrent polls from many users (still acceptable)
- ❌ **Not "real-time":** Strict real-time requires < 1s latency

**Estimated Effort:**
- Implementation: 0.5 days (add interval logic to stores)
- No backend changes needed
- Total: 0.5 days

**Bandwidth estimate:**
- GET /api/alerts: ~50KB (1000 alerts, uncompressed) → gzip ~5KB
- Poll interval: 5s
- Per user per minute: 12 requests × 5KB = 60KB/min
- 100 concurrent users: 6MB/min ≈ 100KB/s (acceptable)

---

### Option B: WebSocket Real-time Push (Post-MVP Enhancement)

**Description:**

Server maintains WebSocket connections. When data changes, server pushes updates to connected clients.

```
1. Client connects: WebSocket /ws
2. Server stores connection reference
3. When alert created: server sends to all connected clients
4. Client updates UI immediately
```

**Mechanism:**

```go
// Backend: WebSocket endpoint
func (h *Handler) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := websocket.Upgrade(w, r, nil)
    defer conn.Close()

    clients.Add(conn)
    
    for {
        // Read messages / handle disconnection
    }
}

// When alert created:
func (h *Handler) CreateSilence(...) {
    // ... create silence ...
    
    // Broadcast to all connected clients
    for client := range clients {
        client.WriteJSON(SilenceCreatedEvent{...})
    }
}
```

**Pros:**
- ✅ **True real-time:** Updates within milliseconds
- ✅ **Lower bandwidth:** Only push when data changes
- ✅ **Better UX:** Immediate feedback
- ✅ **Scalable with pub/sub:** Can use Redis Pub/Sub for multi-instance

**Cons:**
- ❌ **Complex:** Requires connection management, error handling
- ❌ **Server state:** Maintains connections (complicates scaling)
- ❌ **Infrastructure:** Requires Redis (or similar) for multi-instance
- ❌ **Browser compatibility:** Older browsers need fallback
- ❌ **Implementation effort:** 2-3 days for MVP WebSocket
- ❌ **Debugging:** Harder to troubleshoot connection issues

**Estimated Effort:**
- Backend: 1.5 days (WebSocket handler, broadcast logic)
- Frontend: 1 day (WebSocket client, error handling)
- Testing: 1 day (connection stability, reconnection)
- Total: 3.5 days

**Bandwidth estimate:**
- Per update: ~1KB
- If 100 alerts/min across all instances: 100 updates × 1KB = 100KB/min
- Much lower than polling for active instances

---

### Option C: Server-Sent Events (SSE)

**Description:**

Hybrid approach: client opens long-lived HTTP connection, server sends events as they occur.

```
GET /api/alerts/stream
→ Server-Sent Events (text/event-stream)
→ Client receives events in real-time
```

**Pros:**
- ✅ Simpler than WebSocket (uses standard HTTP)
- ✅ Real-time updates
- ✅ Works with proxies/load balancers

**Cons:**
- ❌ Still requires connection management
- ❌ No built-in request/response (WebSocket allows bidirectional)
- ❌ Browser compatibility (not IE)
- ❌ Implementation effort: 2 days

**Not recommended** (WebSocket is better if doing real-time)

---

### Option D: Hybrid Polling + Webhook

**Description:**

Frontend polls periodically, but backend can also push via webhook for critical events.

**Cons:**
- ❌ Complexity of two systems
- ❌ Webhook infrastructure needed
- ❌ Overkill for MVP

**Not recommended**

---

## Decision

**✅ APPROVED: Client-side Polling (Option A) for MVP**

**With planned enhancement to WebSocket (Option B) in Phase 2**

**Rationale:**

1. **MVP timeline priority:**
   - Polling can be implemented in 0.5 days
   - WebSocket would require 3.5 days
   - Phase 1 has tight timeline (30 days total)

2. **Acceptable for users:**
   - 5-second latency is acceptable for alert monitoring
   - Not real-time, but "near-real-time" sufficient for MVP
   - Users can still manually refresh if needed

3. **Scalability is simple:**
   - No server-side connection state
   - Works with multiple backend instances immediately
   - No Redis or message queue needed

4. **Easy migration path:**
   - Polling implementation doesn't block WebSocket later
   - Can add WebSocket without removing polling (graceful upgrade)
   - Same UI, different backend mechanism

5. **Operational simplicity:**
   - No connection management complexity
   - Fewer failure modes
   - Easy debugging

6. **Resource efficient for MVP scale:**
   - With < 50 concurrent users, polling is acceptable
   - At 100 users, might consider WebSocket optimization
   - Clear threshold for upgrade decision

---

## Implementation Details

### Frontend Polling Implementation

**File:** `web/src/stores/alertsStore.ts`

```typescript
import { writable } from 'svelte/store';

export const alertsStore = writable<Alert[]>([]);
export const silencesStore = writable<Silence[]>([]);
export const routingTreeStore = writable<RoutingNode | null>(null);

// Polling configuration
const POLL_INTERVALS = {
  alerts: 5000,      // 5 seconds
  silences: 5000,    // 5 seconds
  routingTree: 30000 // 30 seconds (less critical)
};

let pollIntervals = [];

export function startPolling() {
  // Poll alerts
  pollIntervals.push(
    setInterval(async () => {
      try {
        const response = await fetch('/api/alerts');
        if (response.ok) {
          const data = await response.json();
          alertsStore.set(data.groups || []);
        }
      } catch (error) {
        console.error('Failed to fetch alerts:', error);
        // Don't set empty array on error, keep stale data
      }
    }, POLL_INTERVALS.alerts)
  );

  // Poll silences
  pollIntervals.push(
    setInterval(async () => {
      try {
        const response = await fetch('/api/silences');
        if (response.ok) {
          const data = await response.json();
          silencesStore.set(data.silences || []);
        }
      } catch (error) {
        console.error('Failed to fetch silences:', error);
      }
    }, POLL_INTERVALS.silences)
  );

  // Poll routing tree (optional, less frequent)
  pollIntervals.push(
    setInterval(async () => {
      try {
        const response = await fetch('/api/routing-tree');
        if (response.ok) {
          const data = await response.json();
          routingTreeStore.set(data);
        }
      } catch (error) {
        console.error('Failed to fetch routing tree:', error);
      }
    }, POLL_INTERVALS.routingTree)
  );
}

export function stopPolling() {
  pollIntervals.forEach(id => clearInterval(id));
  pollIntervals = [];
}
```

**Integration in layout:**

```svelte
<!-- web/src/routes/+layout.svelte -->
<script>
  import { onMount } from 'svelte';
  import { startPolling, stopPolling } from '../stores';

  onMount(() => {
    startPolling();
    return () => stopPolling();
  });
</script>

<!-- Rest of layout... -->
```

### Optimization: Smart Polling

**Reduce unnecessary polls:**

```typescript
// Only poll if window is focused
let isWindowFocused = true;

window.addEventListener('focus', () => {
  isWindowFocused = true;
  // Optionally fetch immediately on re-focus
});

window.addEventListener('blur', () => {
  isWindowFocused = false;
});

// In poll function:
if (isWindowFocused) {
  // Do polling
}
```

**Backoff on error:**

```typescript
let consecutiveErrors = 0;

async function pollAlerts() {
  try {
    const response = await fetch('/api/alerts');
    if (response.ok) {
      const data = await response.json();
      alertsStore.set(data);
      consecutiveErrors = 0; // reset
    }
  } catch (error) {
    consecutiveErrors++;
    
    // Back off: 5s → 10s → 30s
    const backoffInterval = 5000 * Math.pow(2, Math.min(consecutiveErrors - 1, 2));
    
    // Re-schedule with backoff
    setTimeout(pollAlerts, backoffInterval);
  }
}
```

### Backend Considerations

**Optimize payload sizes:**

```go
// GET /api/alerts should return minimal data
type AlertResponse struct {
    Groups []AlertGroup `json:"groups"`
    Total  int          `json:"total"`
    // Omit unnecessary fields
}

// Enable gzip compression in middleware
middleware.Compress()(mux)
```

**Add caching headers:**

```go
w.Header().Set("Cache-Control", "no-cache, must-revalidate")
w.Header().Set("ETag", generateETag(data))
```

Client can use ETag to avoid re-processing identical data:

```typescript
// In poll function
const response = await fetch('/api/alerts', {
  headers: {
    'If-None-Match': lastETag
  }
});

if (response.status === 304) {
  // Not modified, skip update
  return;
}

lastETag = response.headers.get('ETag');
```

---

### Testing Strategy

```typescript
// Test polling start/stop
it('should start polling on mount', () => {
  render(Layout);
  expect(setInterval).toHaveBeenCalled();
});

// Test fetch on interval
it('should fetch alerts every 5 seconds', fakeTimers((timers) => {
  startPolling();
  timers.tick(5000);
  expect(fetch).toHaveBeenCalledWith('/api/alerts');
});

// Test error handling
it('should handle fetch errors gracefully', async () => {
  global.fetch = jest.fn().mockRejectedValue(new Error('Network error'));
  startPolling();
  // Should not crash, should retry
});
```

---

## Migration Path to WebSocket (Phase 2)

**When to migrate:**
- User feedback indicates 5s latency is too slow
- Production scale requires optimization
- Features benefit from bidirectional communication

**Migration steps:**
1. Implement WebSocket endpoint in parallel with Feature #6
2. Add `useWebSocket` flag to store
3. Fallback: if WebSocket fails, revert to polling
4. Remove polling once WebSocket is stable in production

**No breaking changes to component code** (same store interface)

---

## Security Considerations

1. **Rate limiting:** Prevent client from polling faster than configured interval
   - Backend: rate-limit by IP if polling > 1 req/sec per endpoint
   
2. **Authentication:** Each poll request authenticated (standard API security)

3. **CSRF:** Not applicable (GET requests, no state change)

4. **XSS:** Polling doesn't introduce new XSS vectors

---

## Performance Baseline

**Measurement points:**

```
Scenario: 50 concurrent users, 5-second poll interval
- Requests/min: 50 × 12 = 600 requests/min = 10 req/s
- Per request: 10KB average (5KB gzipped alerts)
- Bandwidth: 100KB/s = 360MB/hour
- Server CPU: Minimal (stateless endpoint)
```

**Acceptable for MVP.** WebSocket would reduce to ~10-50 KB/s with push model.

---

## Dependencies & Coordination

- **Depends on:** None (HTTP API already exists)
- **Affects:** All features benefit from live updates
- **Can be added:** Anytime after feature API endpoints ready

---

## Success Criteria

- [ ] Polling starts on app mount
- [ ] Polling stops on app unmount
- [ ] Alerts update every 5 seconds
- [ ] Silences update every 5 seconds
- [ ] Error handling doesn't crash app
- [ ] Window blur optimization works (reduce CPU)
- [ ] ETag caching reduces bandwidth
- [ ] Tests pass (≥80% coverage)
- [ ] No memory leaks on repeated polls

---

## Timeline

- **Duration:** 0.5 days (can be added in parallel with features)
- **Sprint:** Sprint 1 or 2 (flexible, non-blocking)

---

## Related ADRs

- ADR-001: Routing Tree Visualization
- ADR-002: Form Framework Selection
- ADR-003: Config Storage & Rollback Strategy

---

## Approval Sign-off

- **Architect:** ✅ Approved 2026-03-09 (polling for MVP)
- **Developer:** ⬜ To confirm on implementation
- **Operations:** ✅ Polling is operationally simple
- **Future:** ✅ WebSocket planned for Phase 2

---

## Notes

1. **Polling is predictable:** Easy to understand and debug
2. **Network-efficient with compression:** Gzip reduces most traffic
3. **User-friendly optimization:** Pause when browser blurred
4. **Clear upgrade path:** WebSocket doesn't require rewrite

---

**End of ADR-004**
