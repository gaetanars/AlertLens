import { writable, derived } from 'svelte/store';
import type { Alert, InstanceStatus } from '$lib/api/types';
import { fetchAlerts, fetchAlertmanagers } from '$lib/api/alerts';

// ─── Raw data stores ─────────────────────────────────────────────────────────

export const alerts = writable<Alert[]>([]);
export const instances = writable<InstanceStatus[]>([]);
export const alertsLoading = writable(false);
export const alertsError = writable<string | null>(null);

// ─── Filter/view state ───────────────────────────────────────────────────────

export const filterQuery = writable('');
export const instanceFilter = writable('');
export const groupByLabel = writable('severity');
export const viewMode = writable<'kanban' | 'list'>('kanban');
export const selectedFingerprints = writable<Set<string>>(new Set());

// ─── Derived: filtered alerts ────────────────────────────────────────────────

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

// ─── Derived: alerts grouped by label ────────────────────────────────────────

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

export async function loadAlerts() {
	alertsLoading.set(true);
	alertsError.set(null);
	try {
		const data = await fetchAlerts({ active: true });
		alerts.set(data ?? []);
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

// ─── Simple matcher-syntax filter (subset of AM syntax) ─────────────────────

function matchesFilterQuery(alert: Alert, query: string): boolean {
	// Parse simple matchers like: key="val", key=~"regex", key!="val"
	const matcherRe = /(\w+)(=~|!=|=)["']?([^"',\s}]*)["']?/g;
	let match;
	let found = false;
	let parsed = false;
	while ((match = matcherRe.exec(query)) !== null) {
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
		}
		if (!ok) return false;
		found = true;
	}
	if (parsed) return found;
	// Fallback: substring search across all labels
	const lower = query.toLowerCase();
	return (
		Object.values(alert.labels).some((v) => v.toLowerCase().includes(lower)) ||
		Object.values(alert.annotations).some((v) => v.toLowerCase().includes(lower))
	);
}
