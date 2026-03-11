# Design Document: AlertLens Alert Data Model & API

**Project:** AlertLens  
**Document Type:** Implementation Guide  
**Version:** 1.0  
**Date:** 2026-03-10  
**Related ADR:** ADR-005

---

## Overview

This document provides detailed guidance for implementing the Alert data model, routing strategy, and API endpoints defined in [ADR-005](./ADR-005_ALERT_DATA_MODEL_AND_ROUTING.md).

It serves as a bridge between architecture (ADR) and code, with concrete examples for:
- TypeScript/Zod schema definitions
- SvelteKit route implementation
- Go backend handler patterns
- Testing examples
- Error handling

---

## Table of Contents

1. [TypeScript Type Definitions](#typescript-type-definitions)
2. [Zod Validation Schemas](#zod-validation-schemas)
3. [Frontend Implementation](#frontend-implementation)
4. [Backend Handler Examples](#backend-handler-examples)
5. [Error Handling](#error-handling)
6. [Testing Guide](#testing-guide)
7. [Performance Optimization](#performance-optimization)

---

## TypeScript Type Definitions

**File: `web/src/lib/api/types.ts`** (Updated)

```typescript
// ─── Core Alert Types ─────────────────────────────────────────────────────

/**
 * A single alert from Alertmanager.
 * 
 * Fingerprint is globally unique within the scope of an Alertmanager instance.
 * To identify an alert across instances, use (alertmanager, fingerprint) tuple.
 */
export interface Alert {
  // Identity
  fingerprint: string;
  alertmanager: string;
  instance_id?: string;  // Alias for alertmanager (Future: Feature #2)

  // Content
  labels: Record<string, string>;
  annotations: Record<string, string>;
  
  // Lifecycle
  state: 'firing' | 'resolved';  // Alertmanager-native state
  startsAt: string;              // ISO 8601 UTC
  endsAt: string;                // ISO 8601 UTC or "0001-01-01T..." for firing
  updatedAt?: string;            // ISO 8601 UTC, last state transition
  generatorURL: string;          // Prometheus rule URL
  
  // Routing
  receivers: { name: string }[];
  status: AlertStatus;
  
  // Optional: Acknowledgment
  ack?: Ack;
}

/**
 * View-layer alert status (computed on backend).
 * 
 * Distinct from Alertmanager's native `state`:
 * - 'active': state === 'firing' && not silenced && not inhibited
 * - 'suppressed': silencedBy.length > 0 OR inhibitedBy.length > 0
 * - 'unprocessed': state === 'resolved'
 */
export interface AlertStatus {
  state: 'active' | 'suppressed' | 'unprocessed';
  silencedBy: string[];    // Silence IDs that apply to this alert
  inhibitedBy: string[];   // Alert fingerprints that inhibit this one
}

/**
 * Acknowledgment record.
 * 
 * Either a visual ACK (user clicked "ack") or a silence-based ACK.
 * Future: May include expiration time, escalation.
 */
export interface Ack {
  active: boolean;
  by: string;             // User identifier (email, LDAP, etc.)
  comment: string;
  silence_id: string;     // If ack is via silence, reference to Silence
}

// ─── Grouping & Aggregation ───────────────────────────────────────────────

/**
 * A group of alerts sharing the same groupBy label values.
 * 
 * Returned by /api/alerts when group_by parameter is set.
 * If group_by is empty, returns single group with empty labels.
 */
export interface AlertGroup {
  /**
   * Grouping label values.
   * 
   * Example: If group_by=[severity, alertname]:
   *   labels = { severity: "critical", alertname: "DiskFull" }
   * 
   * If group_by is empty:
   *   labels = {}
   */
  labels: Record<string, string>;
  
  /** Alerts in this group */
  alerts: Alert[];
  
  /** Total count of alerts in this group (for pagination within groups) */
  count: number;
}

/**
 * Per-instance error from a multi-instance fetch.
 */
export interface InstanceError {
  instance: string;
  error: string;  // User-friendly error message
}

/**
 * Response from GET /api/alerts.
 * 
 * Always paginated by alert count (not group count).
 * If group_by is empty, returns single group.
 */
export interface AlertsResponse {
  groups: AlertGroup[];
  
  /** Total alert count (sum of all group counts) before pagination */
  total: number;
  
  /** Pagination parameters (echo request) */
  limit: number;
  offset: number;
  
  /** Per-instance errors (non-fatal, degraded response) */
  partial_failures?: InstanceError[];
}

// ─── Query Parameters ─────────────────────────────────────────────────────

/**
 * Query parameters for GET /api/alerts.
 * 
 * All fields are optional; omitted fields default to "fetch all".
 */
export interface AlertsParams {
  /** Alertmanager matcher expressions (repeatable param) */
  filter?: string[];
  
  /** Restrict to single Alertmanager instance */
  instance?: string;
  
  /** Include silenced alerts? */
  silenced?: boolean;
  
  /** Include inhibited alerts? */
  inhibited?: boolean;
  
  /** Include only active alerts? (filters out suppressed & unprocessed) */
  active?: boolean;
  
  /** Filter by view-layer severity label (not sent to Alertmanager) */
  severity?: string[];
  
  /** Filter by view-layer status state */
  status?: ('active' | 'suppressed' | 'unprocessed')[];
  
  /** Group by these label keys */
  groupBy?: string[];
  
  /** Pagination: max alerts per response */
  limit?: number;
  
  /** Pagination: offset (0-based) */
  offset?: number;
  
  /** Future: Sort key and direction */
  sortBy?: string;
}

// ─── Matcher (Optional, for UI convenience) ───────────────────────────────

/**
 * Parsed representation of a Prometheus matcher.
 * 
 * Used internally in UI for building complex filters.
 * Not sent to API (API receives filter strings directly).
 */
export interface Matcher {
  name: string;        // Label key
  value: string;       // Label value or regex pattern
  isRegex: boolean;    // true: value is RE2 regex, false: literal match
  isEqual: boolean;    // true: = or =~, false: != or !~
}

// ─── Helper Functions ─────────────────────────────────────────────────────

/**
 * Convert Matcher to Alertmanager filter string.
 */
export function matcherToFilter(m: Matcher): string {
  if (m.isRegex) {
    return `${m.name}${m.isEqual ? '=~' : '!~'}${m.value}`;
  } else {
    return `${m.name}${m.isEqual ? '=' : '!='}${m.value}`;
  }
}

/**
 * Parse Alertmanager filter string to Matcher.
 * Returns null if not a valid matcher.
 */
export function filterToMatcher(filter: string): Matcher | null {
  // Regex: name(=~|!=|=|!~)value
  const regex = /^([a-zA-Z_][a-zA-Z0-9_]*)((=~|!=|=|!~))(.*)$/;
  const match = filter.match(regex);
  
  if (!match) return null;
  
  const [, name, op, , value] = match;
  return {
    name,
    value,
    isRegex: op.includes('~'),
    isEqual: op.startsWith('='),
  };
}

/**
 * Generate ETag for a response (for caching).
 */
export function generateETag(data: AlertsResponse): string {
  const hash = require('crypto').createHash('md5');
  hash.update(JSON.stringify(data));
  return `"${hash.digest('hex')}"`;
}

/**
 * Check if two AlertsResponses are equivalent (for caching).
 */
export function alertsResponsesEqual(a: AlertsResponse, b: AlertsResponse): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}
```

---

## Zod Validation Schemas

**File: `web/src/lib/api/schemas.ts`** (New)

```typescript
import { z } from 'zod';
import type { AlertsParams, Matcher } from './types';

// ─── Utility Schemas ──────────────────────────────────────────────────────

const LabelKeySchema = z
  .string()
  .min(1)
  .max(200)
  .regex(/^[a-zA-Z_][a-zA-Z0-9_]*$/, 'Invalid label name syntax');

const LabelValueSchema = z.string().max(500);

const RegexSchema = z
  .string()
  .max(100)
  .refine(
    (v) => {
      try {
        new RegExp(v); // Verify valid JS regex (approximates RE2)
        return true;
      } catch {
        return false;
      }
    },
    'Invalid regex syntax'
  );

const MatcherStringSchema = z
  .string()
  .max(500)
  .refine(
    (v) => {
      const regex = /^([a-zA-Z_][a-zA-Z0-9_]*)((=~|!=|=|!~))(.*)$/;
      return regex.test(v);
    },
    'Invalid matcher format (expected: name=~value or name!~value, etc.)'
  );

// ─── Matcher Schema ───────────────────────────────────────────────────────

export const MatcherSchema = z.object({
  name: LabelKeySchema,
  value: LabelValueSchema,
  isRegex: z.boolean(),
  isEqual: z.boolean(),
});

export type MatcherType = z.infer<typeof MatcherSchema>;

// ─── Alert Query Parameters ───────────────────────────────────────────────

export const AlertsParamsSchema = z.object({
  filter: z.array(MatcherStringSchema).optional(),
  instance: z.string().max(200).optional(),
  silenced: z.boolean().optional(),
  inhibited: z.boolean().optional(),
  active: z.boolean().optional(),
  severity: z.array(z.enum(['critical', 'warning', 'info'])).optional(),
  status: z
    .array(z.enum(['active', 'suppressed', 'unprocessed']))
    .optional(),
  groupBy: z.array(LabelKeySchema).optional(),
  limit: z.number().int().min(1).max(1000).default(500),
  offset: z.number().int().min(0).default(0),
  sortBy: z.string().optional(), // Reserved for future
});

export type AlertsParamsType = z.infer<typeof AlertsParamsSchema>;

// ─── Validation Utilities ──────────────────────────────────────────────────

/**
 * Validate query parameters from URL or request.
 * Throws on validation error.
 */
export function validateAlertsParams(data: unknown): AlertsParams {
  return AlertsParamsSchema.parse(data);
}

/**
 * Validate with error details (returns object instead of throwing).
 */
export function validateAlertsParamsSafe(
  data: unknown
): { success: true; data: AlertsParams } | { success: false; errors: string[] } {
  const result = AlertsParamsSchema.safeParse(data);
  
  if (!result.success) {
    const errors = result.error.issues.map((issue) => {
      const path = issue.path.join('.');
      return `${path || 'root'}: ${issue.message}`;
    });
    return { success: false, errors };
  }
  
  return { success: true, data: result.data };
}
```

---

## Frontend Implementation

### Route File: `web/src/routes/alerts/+page.svelte`

```svelte
<script lang="ts">
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { writable, derived } from 'svelte/store';
  
  import type { AlertsResponse, Alert, AlertsParams } from '$lib/api/types';
  import { fetchAlerts } from '$lib/api/alerts';
  import AlertTable from '$lib/components/alerts/AlertTable.svelte';
  import AlertFilters from '$lib/components/alerts/AlertFilters.svelte';
  
  // ─── State ────────────────────────────────────────────────────────────

  let loading = false;
  let error: string | null = null;
  
  const alertsData = writable<AlertsResponse>({
    groups: [],
    total: 0,
    limit: 500,
    offset: 0,
  });

  // ─── Derived: Current URL params as AlertsParams ──────────────────────

  const params = derived(page, ($page) => {
    const q = new URLSearchParams($page.url.search);
    
    return {
      filter: q.getAll('filter'),
      instance: q.get('instance') || undefined,
      silenced: q.get('silenced') === 'true' ? true : undefined,
      inhibited: q.get('inhibited') === 'true' ? true : undefined,
      active: q.get('active') === 'true' ? true : undefined,
      severity: q.getAll('severity'),
      status: q.getAll('status'),
      groupBy: q.getAll('group_by'),
      limit: parseInt(q.get('limit') || '500'),
      offset: parseInt(q.get('offset') || '0'),
    } as AlertsParams;
  });

  // ─── Watchers: Fetch when params change ──────────────────────────────

  page.subscribe(async () => {
    await loadAlerts();
  });

  async function loadAlerts() {
    loading = true;
    error = null;
    
    try {
      const $params = $params;  // Read current params from store
      const response = await fetchAlerts($params);
      alertsData.set(response);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Unknown error';
      alertsData.set({
        groups: [],
        total: 0,
        limit: 500,
        offset: 0,
      });
    } finally {
      loading = false;
    }
  }

  // ─── Actions: Build new URL from user input ──────────────────────────

  function applyFilters(newParams: Partial<AlertsParams>) {
    const $params = $params;
    const merged = { ...$params, ...newParams, offset: 0 };  // Reset pagination
    
    const q = new URLSearchParams();
    
    if (merged.filter?.length) {
      merged.filter.forEach((f) => q.append('filter', f));
    }
    if (merged.instance) q.set('instance', merged.instance);
    if (merged.silenced !== undefined) q.set('silenced', String(merged.silenced));
    if (merged.inhibited !== undefined) q.set('inhibited', String(merged.inhibited));
    if (merged.active !== undefined) q.set('active', String(merged.active));
    if (merged.severity?.length) {
      merged.severity.forEach((s) => q.append('severity', s));
    }
    if (merged.status?.length) {
      merged.status.forEach((s) => q.append('status', s));
    }
    if (merged.groupBy?.length) {
      merged.groupBy.forEach((g) => q.append('group_by', g));
    }
    if (merged.limit && merged.limit !== 500) {
      q.set('limit', String(merged.limit));
    }
    
    goto(`/alerts?${q.toString()}`);
  }

  function goToPage(offset: number) {
    applyFilters({ offset });
  }

  function clearFilters() {
    goto('/alerts');
  }
</script>

<!-- ─── UI ───────────────────────────────────────────────────────────── -->

<div class="alerts-page">
  <h1>Alerts</h1>
  
  {#if error}
    <div class="error-banner">
      {error}
      <button on:click={loadAlerts}>Retry</button>
    </div>
  {/if}

  {#if $alertsData.partial_failures?.length}
    <div class="warning-banner">
      ⚠ {$alertsData.partial_failures.length} instance(s) unavailable:
      {#each $alertsData.partial_failures as failure}
        <div>{failure.instance}: {failure.error}</div>
      {/each}
    </div>
  {/if}

  <AlertFilters {params} on:apply={(e) => applyFilters(e.detail)} on:clear={clearFilters} />

  {#if loading}
    <div class="spinner">Loading...</div>
  {:else}
    <AlertTable
      alerts={$alertsData}
      {params}
      on:next={() => goToPage(($alertsData.offset || 0) + ($alertsData.limit || 500))}
      on:prev={() => goToPage(Math.max(0, ($alertsData.offset || 0) - ($alertsData.limit || 500)))}
    />
  {/if}
</div>

<style>
  .alerts-page {
    padding: 1rem;
  }
  
  .error-banner {
    background: #fee;
    color: #c00;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
  }
  
  .warning-banner {
    background: #ffeaa7;
    color: #663300;
    padding: 1rem;
    border-radius: 4px;
    margin-bottom: 1rem;
  }
  
  .spinner {
    text-align: center;
    padding: 2rem;
    color: #999;
  }
</style>
```

### Client Utility: `web/src/lib/api/client.ts` (Updated)

```typescript
import { browser } from '$app/environment';

const CACHE_TTL_MS = 30 * 60 * 1000; // 30 minutes

interface CacheEntry {
  data: unknown;
  etag: string | null;
  timestamp: number;
}

const localCache = new Map<string, CacheEntry>();

function getCacheKey(path: string, params: unknown): string {
  return `${path}:${JSON.stringify(params || {})}`;
}

function isCacheValid(entry: CacheEntry): boolean {
  return Date.now() - entry.timestamp < CACHE_TTL_MS;
}

export const api = {
  /**
   * GET with caching support.
   * Uses localStorage + ETag for HTTP-level caching.
   */
  async get<T>(path: string, options?: { params?: unknown }): Promise<T> {
    if (!browser) {
      throw new Error('API client only works in browser');
    }

    const cacheKey = getCacheKey(path, options?.params);
    const cached = localCache.get(cacheKey);

    // Check local cache
    if (cached && isCacheValid(cached)) {
      return cached.data as T;
    }

    // Build URL with params
    const url = new URL(path, window.location.origin);
    if (options?.params && typeof options.params === 'object') {
      const params = options.params as Record<string, unknown>;
      Object.entries(params).forEach(([key, value]) => {
        if (Array.isArray(value)) {
          value.forEach((v) => url.searchParams.append(key, String(v)));
        } else if (value !== undefined && value !== null) {
          url.searchParams.set(key, String(value));
        }
      });
    }

    // Fetch from server
    const headers: Record<string, string> = {};
    if (cached?.etag) {
      headers['If-None-Match'] = cached.etag;
    }

    const response = await fetch(url, { headers });

    // Handle 304 Not Modified
    if (response.status === 304 && cached) {
      localCache.set(cacheKey, {
        ...cached,
        timestamp: Date.now(), // Refresh TTL
      });
      return cached.data as T;
    }

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    const data = (await response.json()) as T;
    const etag = response.headers.get('ETag');

    localCache.set(cacheKey, {
      data,
      etag,
      timestamp: Date.now(),
    });

    return data;
  },

  /**
   * POST without caching.
   */
  async post<T>(path: string, body: unknown): Promise<T> {
    if (!browser) {
      throw new Error('API client only works in browser');
    }

    const url = new URL(path, window.location.origin);
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }

    return (await response.json()) as T;
  },

  /**
   * Clear cache entry (call after mutations).
   */
  clearCache(path: string, params?: unknown): void {
    const key = getCacheKey(path, params);
    localCache.delete(key);
  },

  /**
   * Clear all cache.
   */
  clearAllCache(): void {
    localCache.clear();
  },
};
```

---

## Backend Handler Examples

### Go Handler: `internal/api/handlers/alerts.go`

```go
package handlers

import (
  "net/http"
  "strconv"
  
  "github.com/go-chi/chi/v5"
  "your-project/internal/models"
  "your-project/internal/services"
)

type AlertsHandler struct {
  service *services.AlertService
}

// ListAlerts handles GET /api/alerts
func (h *AlertsHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
  // Parse query parameters
  filters := r.URL.Query()["filter"]
  instance := r.URL.Query().Get("instance")
  silenced := parseBool(r.URL.Query().Get("silenced"))
  inhibited := parseBool(r.URL.Query().Get("inhibited"))
  active := parseBool(r.URL.Query().Get("active"))
  severities := r.URL.Query()["severity"]
  statuses := r.URL.Query()["status"]
  groupBy := r.URL.Query()["group_by"]
  
  limit := 500
  if l := r.URL.Query().Get("limit"); l != "" {
    if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
      limit = parsed
    }
  }
  
  offset := 0
  if o := r.URL.Query().Get("offset"); o != "" {
    if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
      offset = parsed
    }
  }
  
  params := models.AlertQueryParams{
    Filters:   filters,
    Instance:  instance,
    Silenced:  silenced,
    Inhibited: inhibited,
    Active:    active,
    Severities: severities,
    Statuses:  statuses,
    GroupBy:   groupBy,
    Limit:     limit,
    Offset:    offset,
  }
  
  // Validate params
  if err := validate.Struct(params); err != nil {
    RespondError(w, http.StatusBadRequest, "Invalid parameters", err)
    return
  }
  
  // Fetch alerts
  response, err := h.service.FetchAlerts(r.Context(), params)
  if err != nil {
    RespondError(w, http.StatusInternalServerError, "Failed to fetch alerts", err)
    return
  }
  
  // Generate ETag
  etagValue := generateETag(response)
  w.Header().Set("ETag", etagValue)
  
  // Check If-None-Match
  if match := r.Header.Get("If-None-Match"); match == etagValue {
    w.WriteHeader(http.StatusNotModified)
    return
  }
  
  // Return response
  RespondJSON(w, http.StatusOK, response)
}

// ─── Utilities ────────────────────────────────────────────────────────

func parseBool(s string) *bool {
  if s == "" {
    return nil
  }
  b := s == "true"
  return &b
}

func generateETag(data interface{}) string {
  // Simplified: use hash of JSON
  // Real implementation: crypto/md5
  return `"simplified-etag"`
}
```

---

## Error Handling

**Pattern: Standardized error responses**

```typescript
// Error response format (all endpoints)
interface ErrorResponse {
  error: string;           // Machine-readable error code
  message: string;         // Human-readable message
  details?: string;        // Optional detailed info
  retry_after?: number;    // Seconds to wait (for rate limits)
}

// Examples:
// 400: Bad Request
{
  "error": "invalid_query_params",
  "message": "limit must be <= 1000",
  "details": "You requested limit=2000"
}

// 429: Too Many Requests
{
  "error": "rate_limited",
  "message": "Too many requests",
  "retry_after": 60
}

// 503: Service Unavailable (all instances down)
{
  "error": "all_instances_down",
  "message": "All Alertmanager instances are unreachable"
}
```

---

## Testing Guide

### Unit Tests: `web/src/lib/api/alerts.test.ts`

```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { validateAlertsParams } from '$lib/api/schemas';
import { matcherToFilter, filterToMatcher } from '$lib/api/types';

describe('Alert Schema Validation', () => {
  it('accepts valid params', () => {
    const valid = {
      filter: ['severity="critical"'],
      limit: 100,
      offset: 0,
      groupBy: ['severity'],
    };
    
    const result = validateAlertsParams(valid);
    expect(result.filter).toEqual(['severity="critical"']);
    expect(result.limit).toBe(100);
  });

  it('rejects limit > 1000', () => {
    expect(() => {
      validateAlertsParams({ limit: 1001 });
    }).toThrow();
  });

  it('rejects invalid matcher syntax', () => {
    expect(() => {
      validateAlertsParams({ filter: ['invalid'] });
    }).toThrow('Invalid matcher format');
  });
});

describe('Matcher Parsing', () => {
  it('parses equality matcher', () => {
    const filter = 'severity="critical"';
    const matcher = filterToMatcher(filter);
    
    expect(matcher).toEqual({
      name: 'severity',
      value: 'critical',
      isRegex: false,
      isEqual: true,
    });
  });

  it('parses regex matcher', () => {
    const filter = 'service=~"web-.*"';
    const matcher = filterToMatcher(filter);
    
    expect(matcher).toEqual({
      name: 'service',
      value: 'web-.*',
      isRegex: true,
      isEqual: true,
    });
  });

  it('converts matcher back to filter', () => {
    const filter = 'env!="test"';
    const matcher = filterToMatcher(filter);
    const reconstructed = matcherToFilter(matcher!);
    
    expect(reconstructed).toBe(filter);
  });

  it('rejects invalid label names', () => {
    const filter = '123invalid="value"';
    expect(filterToMatcher(filter)).toBeNull();
  });
});
```

### E2E Tests: `e2e/alerts.spec.ts`

```typescript
import { test, expect } from '@playwright/test';

test('apply filter and verify URL updates', async ({ page }) => {
  // Navigate to alerts page
  await page.goto('/alerts');
  
  // Click on "Critical" severity filter
  await page.click('text=Critical');
  
  // Verify URL updated
  await expect(page).toHaveURL(/severity=critical/);
  
  // Verify page shows critical alerts only
  const alerts = await page.locator('[data-testid="alert-item"]').all();
  for (const alert of alerts) {
    const severity = await alert.locator('[data-field="severity"]').textContent();
    expect(severity).toBe('critical');
  }
});

test('pagination works', async ({ page }) => {
  await page.goto('/alerts?limit=10');
  
  // Click next
  await page.click('button:has-text("Next")');
  
  // Verify offset changed
  await expect(page).toHaveURL(/offset=10/);
});

test('share link restores view', async ({ page, context }) => {
  // Set up complex filter
  await page.goto('/alerts?filter=severity="critical"&group_by=instance&limit=50');
  
  // Open new tab with same URL
  const newPage = await context.newPage();
  await newPage.goto(page.url());
  
  // Verify same filters applied
  await expect(newPage.locator('[data-testid="filter-badge"]')).toContainText('severity="critical"');
  await expect(newPage.locator('[data-testid="group-by"]')).toContainText('instance');
});

test('graceful degradation on instance failure', async ({ page }) => {
  // Mock one instance to return error
  await page.route('**/api/alertmanagers/instance2', route => {
    route.abort('failed');
  });
  
  await page.goto('/alerts');
  
  // Verify warning banner
  await expect(page.locator('.warning-banner')).toContainText('instance unavailable');
  
  // Verify alerts from healthy instance still shown
  const alerts = await page.locator('[data-testid="alert-item"]').all();
  expect(alerts.length).toBeGreaterThan(0);
});
```

---

## Performance Optimization

### Client-Side Caching

```typescript
// Only refetch if params changed (URLSearchParams comparison)
function paramsCacheKey(params: AlertsParams): string {
  const keys = Object.keys(params).sort();
  const values = keys.map((k) => `${k}=${JSON.stringify(params[k as keyof AlertsParams])}`);
  return values.join('&');
}

let lastParamsCacheKey = '';
let lastResponse: AlertsResponse | null = null;

page.subscribe(async ($page) => {
  const $params = parseParamsFromURL($page.url.search);
  const newKey = paramsCacheKey($params);
  
  if (newKey === lastParamsCacheKey && lastResponse) {
    // Same params, use cached response
    alertsData.set(lastResponse);
    return;
  }
  
  // New params, fetch
  const response = await fetchAlerts($params);
  lastResponse = response;
  lastParamsCacheKey = newKey;
  alertsData.set(response);
});
```

### Server-Side Optimization

```go
// Use query caching with short TTL (10 seconds)
const QUERY_CACHE_TTL = 10 * time.Second

type CachedQuery struct {
  Params    AlertQueryParams
  Response  *AlertsResponse
  Timestamp time.Time
  ETag      string
}

var (
  queryCache = make(map[string]*CachedQuery)
  cacheMutex sync.RWMutex
)

func (h *AlertsHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
  // ... parse params ...
  
  cacheKey := params.Hash()  // Deterministic hash
  
  cacheMutex.RLock()
  cached, exists := queryCache[cacheKey]
  cacheMutex.RUnlock()
  
  if exists && time.Since(cached.Timestamp) < QUERY_CACHE_TTL {
    // Cache hit
    w.Header().Set("ETag", cached.ETag)
    if r.Header.Get("If-None-Match") == cached.ETag {
      w.WriteHeader(http.StatusNotModified)
      return
    }
    RespondJSON(w, http.StatusOK, cached.Response)
    return
  }
  
  // Cache miss: fetch from Alertmanager
  response, err := h.service.FetchAlerts(r.Context(), params)
  // ...
  
  // Store in cache
  etagValue := generateETag(response)
  cacheMutex.Lock()
  queryCache[cacheKey] = &CachedQuery{
    Params:    params,
    Response:  response,
    Timestamp: time.Now(),
    ETag:      etagValue,
  }
  cacheMutex.Unlock()
  
  w.Header().Set("ETag", etagValue)
  RespondJSON(w, http.StatusOK, response)
}
```

---

## Conclusion

This design document provides concrete guidance for implementing the Alert data model and routing strategy from ADR-005. Use it alongside the ADR for both architecture rationale and implementation details.

**Next Steps:**
1. Review with team (backend + frontend developers)
2. Create git branches for type definitions, handlers, routes
3. Implement & test incrementally
4. Iterate based on real data volume & user feedback

**Related Docs:**
- [ADR-005](./ADR-005_ALERT_DATA_MODEL_AND_ROUTING.md) — Architecture decisions
- [ARCHITECTURE_DESIGN_PHASE_1](./ARCHITECTURE_DESIGN_PHASE_1.md) — System overview

