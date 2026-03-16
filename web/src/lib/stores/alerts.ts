import { writable, derived, get } from 'svelte/store';
import type { Alert, AlertGroup, AlertsResponse, InstanceStatus, InstanceError } from '$lib/api/types';
import { fetchAlerts, fetchAlertmanagers } from '$lib/api/alerts';

// ─── Raw data stores ─────────────────────────────────────────────────────────

/** Flat alert list (unwrapped from groups). Used by AlertList / AlertTable. */
export const alerts = writable<Alert[]>([]);

/** Grouped response from the latest API call. Used by AlertKanban. */
export const alertsGrouped = writable<AlertGroup[]>([]);

/** Total alert count before pagination (from last API response). */
export const alertsTotal = writable(0);

export const instances = writable<InstanceStatus[]>([]);
export const alertsLoading = writable(false);
export const alertsError = writable<string | null>(null);
/** Partial failures: instances that failed to respond in the last fetch. */
export const alertsPartialFailures = writable<InstanceError[]>([]);

// ─── Filter/view state ───────────────────────────────────────────────────────

/** Free-form text/matcher query (client-side filter). */
export const filterQuery = writable('');

/** Filter by alertmanager instance (client-side after fetch, also sent to API). */
export const instanceFilter = writable('');

/** Severity filter sent to the API (multi-select). */
export const severityFilter = writable<string[]>([]);

/** Status filter sent to the API (multi-select). */
export const statusFilter = writable<string[]>([]);

/** Label key to group by. Sent to API as group_by param. */
export const groupByLabel = writable<string>('severity');

/** Current view mode. */
export const viewMode = writable<'kanban' | 'list'>('kanban');

/** Selected alert fingerprints for bulk actions. */
export const selectedFingerprints = writable<Set<string>>(new Set());

// ─── Pagination state ─────────────────────────────────────────────────────────

export const alertsLimit = writable(500);
export const alertsOffset = writable(0);

// ─── Derived: filtered alerts (client-side text filter) ──────────────────────

export const filteredAlerts = derived(
	[alerts, filterQuery, instanceFilter],
	([$alerts, $filter, $instance]) => {
		let result = $alerts;
		if ($instance) {
			result = result.filter((a) => a.alertmanager === $instance);
		}
		if ($filter.trim()) {
			result = result.filter((a) => matchesFilterQuery(a, $filter.trim()));
		}
		return result;
	}
);

// ─── Derived: client-side grouped alerts (for kanban/grouping with text filter) ─

export const filteredGrouped = derived(
	[alertsGrouped, filterQuery, instanceFilter],
	([$groups, $filter, $instance]) => {
		if (!$filter.trim() && !$instance) return $groups;
		// Re-filter each group's alerts.
		return $groups
			.map((g) => {
				let a = g.alerts;
				if ($instance) a = a.filter((al) => al.alertmanager === $instance);
				if ($filter.trim()) a = a.filter((al) => matchesFilterQuery(al, $filter.trim()));
				return { ...g, alerts: a, count: a.length };
			})
			.filter((g) => g.count > 0);
	}
);

// ─── Derived: alerts grouped by label (client-side, legacy for Kanban) ────────

export const groupedAlerts = derived(
	[filteredAlerts, groupByLabel],
	([$alerts, $groupBy]) => {
		const groups = new Map<string, Alert[]>();
		for (const alert of $alerts) {
			const key = alert.labels[$groupBy] ?? '(none)';
			if (!groups.has(key)) groups.set(key, []);
			groups.get(key)!.push(alert);
		}
		return groups;
	}
);

// ─── Load functions ──────────────────────────────────────────────────────────

/**
 * Load alerts from the API, applying current filter/group/pagination state.
 */
export async function loadAlerts() {
	alertsLoading.set(true);
	alertsError.set(null);

	const $severityFilter = get(severityFilter);
	const $statusFilter = get(statusFilter);
	const $groupByLabel = get(groupByLabel);
	const $instanceFilter = get(instanceFilter);
	const $limit = get(alertsLimit);
	const $offset = get(alertsOffset);

	try {
		const resp = await fetchAlerts({
			active: true,
			severity: $severityFilter.length > 0 ? $severityFilter : undefined,
			status: $statusFilter.length > 0 ? $statusFilter : undefined,
			groupBy: $groupByLabel ? [$groupByLabel] : undefined,
			instance: $instanceFilter || undefined,
			limit: $limit,
			offset: $offset
		});

		// Store both flat and grouped forms.
		const flat = resp.groups.flatMap((g) => g.alerts);
		alerts.set(flat);
		alertsGrouped.set(resp.groups);
		alertsTotal.set(resp.total);
		alertsPartialFailures.set(resp.partial_failures ?? []);
	} catch (e) {
		alertsError.set(e instanceof Error ? e.message : 'Failed to load alerts');
	} finally {
		alertsLoading.set(false);
	}
}

export async function loadInstances() {
	try {
		const data = await fetchAlertmanagers();
		instances.set(data ?? []);
	} catch {
		// silent — instances are non-critical for display
	}
}

<<<<<<< issue-46-multi-alertmanager-aggregation
// ─── Derived: available label keys for group-by dropdown ─────────────────────

/**
 * All unique label keys present in the current alert set, sorted alphabetically.
 * Used to populate the dynamic group-by dropdown.
 */
export const availableGroupByLabels = derived(alerts, ($alerts) => {
	const keys = new Set<string>();
	for (const alert of $alerts) {
		for (const key of Object.keys(alert.labels)) {
			keys.add(key);
=======
// ─── Derived: available label keys across current alerts ─────────────────────

/**
 * All unique label keys present in the current (unfiltered) alert set.
 * Used to populate the group-by dropdown dynamically.
 */
export const availableLabels = derived(alerts, ($alerts) => {
	const keys = new Set<string>();
	for (const a of $alerts) {
		for (const k of Object.keys(a.labels)) {
			keys.add(k);
>>>>>>> main
		}
	}
	return Array.from(keys).sort();
});

<<<<<<< issue-46-multi-alertmanager-aggregation
// ─── Matcher-syntax helpers ──────────────────────────────────────────────────

// Operator order matters: longer tokens (=~, !=, !~) must come before = to avoid
// partial matches. The regex also handles optional surrounding quotes on the value.
const MATCHER_RE = /(\w+)(=~|!~|!=|=)["']?([^"',\s}]*)["']?/g;

/**
 * Validates a matcher query string.
 * Returns an error message if any matcher contains an invalid regex, or null if valid.
 *
 * Exported for use in Vitest tests.
 */
export function validateMatcherSyntax(query: string): string | null {
	if (!query.trim()) return null;
	const re = new RegExp(MATCHER_RE.source, 'g');
	let match;
	while ((match = re.exec(query)) !== null) {
=======
// ─── Matcher-syntax parser/validator ─────────────────────────────────────────

/**
 * Regex matching all four Alertmanager matcher operators in precedence order.
 * `!~` and `=~` must be checked before `!=` and `=` to avoid partial matching.
 */
const MATCHER_RE = /(\w+)(=~|!~|!=|=)["']?([^"',\s}]*)["']?/g;

/**
 * Validate a filter query string.
 * Returns an error message if the query contains invalid syntax (e.g. a
 * bad regex inside `=~` / `!~` matchers), or `null` when the query is valid.
 */
export function validateFilterQuery(query: string): string | null {
	const trimmed = query.trim();
	if (!trimmed) return null;

	MATCHER_RE.lastIndex = 0;
	let match;
	let anyMatcher = false;

	while ((match = MATCHER_RE.exec(trimmed)) !== null) {
		anyMatcher = true;
>>>>>>> main
		const [, , op, val] = match;
		if (op === '=~' || op === '!~') {
			try {
				new RegExp(val);
			} catch {
				return `Invalid regex in matcher: ${val}`;
			}
		}
	}
<<<<<<< issue-46-multi-alertmanager-aggregation
	return null;
}

function matchesFilterQuery(alert: Alert, query: string): boolean {
	// Parse matchers: key=val, key!=val, key=~regex, key!~regex
	const re = new RegExp(MATCHER_RE.source, 'g');
	let match;
	let found = false;
	let parsed = false;
	while ((match = re.exec(query)) !== null) {
=======

	// If no matcher was parsed but the query is non-empty it's a plain-text
	// substring search — that's always valid.
	void anyMatcher;
	return null;
}

// ─── Simple matcher-syntax filter (subset of AM syntax) ─────────────────────

function matchesFilterQuery(alert: Alert, query: string): boolean {
	// Parse matchers: key=val, key!=val, key=~regex, key!~regex (Alertmanager native)
	MATCHER_RE.lastIndex = 0;
	let match;
	let found = false;
	let parsed = false;
	while ((match = MATCHER_RE.exec(query)) !== null) {
>>>>>>> main
		parsed = true;
		const [, key, op, val] = match;
		const labelVal = alert.labels[key] ?? '';
		let ok = false;
		if (op === '=') ok = labelVal === val;
		else if (op === '!=') ok = labelVal !== val;
		else if (op === '=~') {
			try {
				ok = new RegExp(val).test(labelVal);
			} catch {
				ok = false;
			}
		} else if (op === '!~') {
			try {
				ok = !new RegExp(val).test(labelVal);
			} catch {
<<<<<<< issue-46-multi-alertmanager-aggregation
				ok = false;
=======
				ok = true; // bad regex → treat as no match constraint
>>>>>>> main
			}
		}
		if (!ok) return false;
		found = true;
	}
	if (parsed) return found;
<<<<<<< issue-46-multi-alertmanager-aggregation
	// Fallback: substring search across all labels and annotations
=======
	// Fallback: substring search across all labels + annotations
>>>>>>> main
	const lower = query.toLowerCase();
	return (
		Object.values(alert.labels).some((v) => v.toLowerCase().includes(lower)) ||
		Object.values(alert.annotations).some((v) => v.toLowerCase().includes(lower))
	);
}
