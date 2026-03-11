# ADR-006 — Alert Kanban / List Views with URL-synced State

**Status:** Accepted  
**Date:** 2026-03-10  
**Deciders:** AlertLens core team  

---

## Context

AlertLens displays active Prometheus Alertmanager alerts on a single page (`/alerts`).
Two view modes are needed:

| Mode   | Use-case |
|--------|----------|
| Kanban | At-a-glance grouped view (by severity, status, team…). Preferred for NOC dashboards. |
| List   | Dense sortable table. Preferred for investigating large alert floods. |

Users reported that:

1. Refreshing the browser resets the view mode to Kanban (the default), losing filter state.
2. Sharing a URL with a colleague does not preserve the selected view, filters, or sort order.
3. The group-by label and sort column/direction are not bookmarkable.

## Decision

**Persist all user-facing view state in the URL search params** so that:

- Hard-refreshing the page restores the exact view.
- Copying and sharing the URL reproduces the same filtered, sorted view.
- The browser Back/Forward buttons navigate between meaningfully different filter states.

### URL parameter contract

| Param     | Type                        | Default    | Affects |
|-----------|-----------------------------|------------|---------|
| `view`    | `kanban` \| `list`          | `kanban`   | Which component is rendered |
| `q`       | string                      | `""`       | Free-text / matcher filter |
| `instance`| string                      | `""`       | Instance selector |
| `severity`| comma-separated string list | `""`       | Severity chip filter |
| `status`  | comma-separated string list | `""`       | Status chip filter |
| `groupBy` | string                      | `severity` | Kanban group-by label |
| `sort`    | `alertname\|severity\|startsAt\|alertmanager` | `startsAt` | List sort column |
| `sortDir` | `asc` \| `desc`             | `desc`     | List sort direction |

URL changes are pushed with `replaceState` (no new history entries for incremental
filter changes) so the history stack is not polluted. Navigating _between_ views
(`kanban` ↔ `list`) uses `pushState` to allow Back/Forward.

### Component responsibilities

- **`AlertFilters.svelte`** — reads initial state from URL on mount; writes back to URL on every change.
- **`AlertList.svelte`** — reads `sort` / `sortDir` from URL on mount; writes back on column click.
- **`+page.svelte`** — reads `view` param on mount and whenever the URL changes.
- **`alerts` store** — remains the single source of truth for data; filter/view state stores are initialised from URL params instead of hard-coded defaults.

### URL update strategy

SvelteKit's `goto(url, { replaceState: true, keepFocus: true, noScroll: true })` is
used from within SvelteKit components. For components that cannot easily import
`goto`, `history.replaceState` / `history.pushState` are used directly — safe
because the app is served as a single-page application (static adapter, `index.html`
fallback).

## Consequences

### Positive

- Shareable, bookmarkable URLs for every filter/view combination.
- Browser reload is idempotent — same URL = same view.
- No additional storage (localStorage, cookies) required.
- Aligns with the project's "no persistent local state" principle (CLAUDE.md).

### Negative / Trade-offs

- Incremental filter changes modify the URL, which may feel noisy in the address bar.
  Mitigated by `replaceState` (no history entries).
- URL length can grow when many filters are active. Acceptable for the expected filter cardinality.
- Alert sort state (column + direction) is only stored in the URL for list view;
  kanban view does not sort.

## Alternatives considered

| Alternative | Rejected because |
|-------------|-----------------|
| `localStorage` | Breaks shared URLs; violates the "no persistent state" rule. |
| Svelte store with `sessionStorage` | Same sharing problem. |
| No persistence (current state) | Breaks reload and sharing — the root bug. |
