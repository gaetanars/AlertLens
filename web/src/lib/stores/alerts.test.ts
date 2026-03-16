/**
 * Unit tests for the alerts store.
 * Tests the client-side filtering logic (matchesFilterQuery) and store
 * state management without requiring a live API.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import {
	alerts,
	alertsGrouped,
	alertsLoading,
	alertsError,
	alertsTotal,
	alertsPartialFailures,
	filterQuery,
	instanceFilter,
	severityFilter,
	statusFilter,
	groupByLabel,
	viewMode,
	selectedFingerprints,
	filteredAlerts,
	filteredGrouped,
	groupedAlerts,
	availableLabels,
	validateFilterQuery,
	loadAlerts
} from './alerts';
import { fetchAlerts } from '$lib/api/alerts';
import type { Alert, AlertGroup } from '$lib/api/types';

// Hoist the mock so Vitest intercepts the import inside alerts.ts.
vi.mock('$lib/api/alerts', () => ({
	fetchAlerts: vi.fn(),
	fetchAlertmanagers: vi.fn().mockResolvedValue([])
}));

// ─── Helpers ─────────────────────────────────────────────────────────────────

function makeAlert(overrides: Partial<Alert> = {}): Alert {
	return {
		fingerprint: 'fp-test',
		alertmanager: 'prod-eu',
		labels: { alertname: 'TestAlert', severity: 'critical', env: 'prod' },
		annotations: { description: 'Test alert description' },
		state: 'active',
		startsAt: '2026-03-09T10:00:00Z',
		endsAt: '0001-01-01T00:00:00Z',
		generatorURL: '',
		receivers: [],
		status: { state: 'active', silencedBy: [], inhibitedBy: [] },
		...overrides
	};
}

function makeGroup(labelVal: string, groupByKey: string, alertList: Alert[]): AlertGroup {
	return {
		labels: { [groupByKey]: labelVal },
		alerts: alertList,
		count: alertList.length
	};
}

// ─── Store state management ───────────────────────────────────────────────────

describe('alerts store — state management', () => {
	beforeEach(() => {
		alerts.set([]);
		alertsGrouped.set([]);
		alertsLoading.set(false);
		alertsError.set(null);
		filterQuery.set('');
		instanceFilter.set('');
		severityFilter.set([]);
		statusFilter.set([]);
		groupByLabel.set('severity');
		viewMode.set('kanban');
		selectedFingerprints.set(new Set());
	});

	it('initialises with empty alerts', () => {
		expect(get(alerts)).toEqual([]);
	});

	it('initialises with loading=false', () => {
		expect(get(alertsLoading)).toBe(false);
	});

	it('initialises with no error', () => {
		expect(get(alertsError)).toBeNull();
	});

	it('default view mode is kanban', () => {
		expect(get(viewMode)).toBe('kanban');
	});

	it('default groupByLabel is severity', () => {
		expect(get(groupByLabel)).toBe('severity');
	});

	it('selectedFingerprints starts empty', () => {
		expect(get(selectedFingerprints).size).toBe(0);
	});
});

// ─── filteredAlerts derived store ────────────────────────────────────────────

describe('filteredAlerts — client-side filtering', () => {
	const a1 = makeAlert({ fingerprint: 'fp1', alertmanager: 'prod-eu', labels: { alertname: 'CPUHigh', severity: 'critical', env: 'prod' } });
	const a2 = makeAlert({ fingerprint: 'fp2', alertmanager: 'prod-eu', labels: { alertname: 'MemHigh', severity: 'warning', env: 'prod' } });
	const a3 = makeAlert({ fingerprint: 'fp3', alertmanager: 'prod-us', labels: { alertname: 'DiskFull', severity: 'critical', env: 'staging' } });

	beforeEach(() => {
		alerts.set([a1, a2, a3]);
		filterQuery.set('');
		instanceFilter.set('');
	});

	it('returns all alerts when no filters applied', () => {
		expect(get(filteredAlerts)).toHaveLength(3);
	});

	it('filters by instance', () => {
		instanceFilter.set('prod-eu');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(2);
		expect(result.every((a) => a.alertmanager === 'prod-eu')).toBe(true);
	});

	it('filters by text query (alertname)', () => {
		filterQuery.set('CPUHigh');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(1);
		expect(result[0].fingerprint).toBe('fp1');
	});

	it('filters by matcher syntax severity=critical', () => {
		filterQuery.set('severity=critical');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(2);
		result.forEach((a) => expect(a.labels['severity']).toBe('critical'));
	});

	it('filters by matcher syntax env=~"prod.*"', () => {
		filterQuery.set('env=~"prod"');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(2);
	});

	it('filters by negative matcher env!="staging"', () => {
		filterQuery.set('env!="staging"');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(2);
	});

	it('combines instance + text query', () => {
		instanceFilter.set('prod-eu');
		filterQuery.set('CPUHigh');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(1);
		expect(result[0].fingerprint).toBe('fp1');
	});

	it('returns empty array when no match', () => {
		filterQuery.set('NonExistent');
		expect(get(filteredAlerts)).toHaveLength(0);
	});
});

// ─── groupedAlerts derived store ─────────────────────────────────────────────

describe('groupedAlerts — client-side grouping', () => {
	const a1 = makeAlert({ fingerprint: 'fp1', labels: { alertname: 'CPUHigh', severity: 'critical' } });
	const a2 = makeAlert({ fingerprint: 'fp2', labels: { alertname: 'MemHigh', severity: 'warning' } });
	const a3 = makeAlert({ fingerprint: 'fp3', labels: { alertname: 'DiskFull', severity: 'critical' } });

	beforeEach(() => {
		alerts.set([a1, a2, a3]);
		filterQuery.set('');
		instanceFilter.set('');
		groupByLabel.set('severity');
	});

	it('groups alerts by severity', () => {
		const groups = get(groupedAlerts);
		expect(groups.get('critical')).toHaveLength(2);
		expect(groups.get('warning')).toHaveLength(1);
	});

	it('groups alerts by custom label', () => {
		groupByLabel.set('alertname');
		const groups = get(groupedAlerts);
		expect(groups.size).toBe(3);
	});

	it('uses (none) key for missing label', () => {
		groupByLabel.set('team'); // no "team" label in test alerts
		const groups = get(groupedAlerts);
		expect(groups.get('(none)')).toHaveLength(3);
	});
});

// ─── loadAlerts — API interaction ────────────────────────────────────────────

describe('loadAlerts — API interaction', () => {
	beforeEach(() => {
		alerts.set([]);
		alertsGrouped.set([]);
		alertsLoading.set(false);
		alertsError.set(null);
		alertsTotal.set(0);
		alertsPartialFailures.set([]);
		vi.mocked(fetchAlerts).mockReset();
	});

	it('populates stores on success', async () => {
		const group = makeGroup('critical', 'severity', [makeAlert({ fingerprint: 'fp1' })]);
		vi.mocked(fetchAlerts).mockResolvedValue({ groups: [group], total: 1, limit: 500, offset: 0 });
		await loadAlerts();
		expect(get(alerts)).toHaveLength(1);
		expect(get(alertsGrouped)).toHaveLength(1);
		expect(get(alertsTotal)).toBe(1);
		expect(get(alertsError)).toBeNull();
		expect(get(alertsLoading)).toBe(false);
	});

	it('sets error state on API failure', async () => {
		vi.mocked(fetchAlerts).mockRejectedValue(new Error('Network error'));
		await loadAlerts();
		expect(get(alertsError)).toBe('Network error');
		expect(get(alertsLoading)).toBe(false);
	});

	it('sets generic error for non-Error rejection', async () => {
		vi.mocked(fetchAlerts).mockRejectedValue('unexpected string');
		await loadAlerts();
		expect(get(alertsError)).toBe('Failed to load alerts');
		expect(get(alertsLoading)).toBe(false);
	});

	it('clears previous error on successful fetch', async () => {
		alertsError.set('Previous error');
		vi.mocked(fetchAlerts).mockResolvedValue({ groups: [], total: 0, limit: 500, offset: 0 });
		await loadAlerts();
		expect(get(alertsError)).toBeNull();
	});

	it('resets alertsLoading to false after error', async () => {
		vi.mocked(fetchAlerts).mockRejectedValue(new Error('fail'));
		await loadAlerts();
		expect(get(alertsLoading)).toBe(false);
	});

	it('populates partial_failures when present', async () => {
		vi.mocked(fetchAlerts).mockResolvedValue({
			groups: [],
			total: 0,
			limit: 500,
			offset: 0,
			partial_failures: [{ instance: 'prod-eu', error: 'timeout' }]
		});
		await loadAlerts();
		expect(get(alertsPartialFailures)).toHaveLength(1);
		expect(get(alertsPartialFailures)[0].instance).toBe('prod-eu');
	});
});

// ─── severityFilter / statusFilter ───────────────────────────────────────────

describe('filter stores', () => {
	it('severityFilter stores an array', () => {
		severityFilter.set(['critical', 'warning']);
		expect(get(severityFilter)).toEqual(['critical', 'warning']);
	});

	it('statusFilter stores an array', () => {
		statusFilter.set(['active']);
		expect(get(statusFilter)).toEqual(['active']);
	});

	it('groupByLabel accepts alertmanager', () => {
		groupByLabel.set('alertmanager');
		expect(get(groupByLabel)).toBe('alertmanager');
	});

	it('groupByLabel accepts status', () => {
		groupByLabel.set('status');
		expect(get(groupByLabel)).toBe('status');
	});
});

// ─── viewMode store toggle (ADR-006, issue #45) ────────────────────────────

describe('viewMode — toggle between kanban and list', () => {
	beforeEach(() => {
		viewMode.set('kanban');
	});

	it('defaults to kanban', () => {
		expect(get(viewMode)).toBe('kanban');
	});

	it('switches to list', () => {
		viewMode.set('list');
		expect(get(viewMode)).toBe('list');
	});

	it('switches back to kanban from list', () => {
		viewMode.set('list');
		viewMode.set('kanban');
		expect(get(viewMode)).toBe('kanban');
	});

	it('toggle: kanban → list → kanban produces correct sequence', () => {
		const observed: Array<'kanban' | 'list'> = [];
		const unsub = viewMode.subscribe((v) => observed.push(v));
		viewMode.set('list');
		viewMode.set('kanban');
		unsub();
		// Initial value + two updates.
		expect(observed).toEqual(['kanban', 'list', 'kanban']);
	});

	it('setting same value does not emit a new event', () => {
		const observed: Array<'kanban' | 'list'> = [];
		const unsub = viewMode.subscribe((v) => observed.push(v));
		viewMode.set('kanban'); // same value — Svelte writable guards with safe_not_equal
		unsub();
		// Only the initial subscription emission fires; the redundant set is a no-op.
		expect(observed).toHaveLength(1);
	});

	it('accepts both valid modes without throwing', () => {
		expect(() => viewMode.set('kanban')).not.toThrow();
		expect(() => viewMode.set('list')).not.toThrow();
	});
});

// ─── filteredGrouped derived store ────────────────────────────────────────────

describe('filteredGrouped — server groups with client-side filter', () => {
	const a1 = makeAlert({ fingerprint: 'fp1', alertmanager: 'prod-eu', labels: { alertname: 'CPUHigh', severity: 'critical', env: 'prod' }, annotations: {} });
	const a2 = makeAlert({ fingerprint: 'fp2', alertmanager: 'prod-eu', labels: { alertname: 'MemHigh', severity: 'critical', env: 'prod' }, annotations: {} });
	const a3 = makeAlert({ fingerprint: 'fp3', alertmanager: 'prod-us', labels: { alertname: 'DiskFull', severity: 'warning', env: 'staging' }, annotations: {} });

	const criticalGroup = makeGroup('critical', 'severity', [a1, a2]);
	const warningGroup = makeGroup('warning', 'severity', [a3]);

	beforeEach(() => {
		alertsGrouped.set([criticalGroup, warningGroup]);
		filterQuery.set('');
		instanceFilter.set('');
	});

	it('returns all groups when no filters applied', () => {
		const result = get(filteredGrouped);
		expect(result).toHaveLength(2);
	});

	it('preserves group structure when no filters applied', () => {
		const result = get(filteredGrouped);
		expect(result[0].labels['severity']).toBe('critical');
		expect(result[0].alerts).toHaveLength(2);
		expect(result[1].labels['severity']).toBe('warning');
		expect(result[1].alerts).toHaveLength(1);
	});

	it('filters alerts within groups by instanceFilter', () => {
		instanceFilter.set('prod-eu');
		const result = get(filteredGrouped);
		// Both critical alerts are from prod-eu; warning alert is prod-us.
		expect(result).toHaveLength(1);
		expect(result[0].labels['severity']).toBe('critical');
		expect(result[0].alerts).toHaveLength(2);
	});

	it('filters alerts within groups by text query', () => {
		filterQuery.set('CPUHigh');
		const result = get(filteredGrouped);
		expect(result).toHaveLength(1);
		expect(result[0].alerts).toHaveLength(1);
		expect(result[0].alerts[0].fingerprint).toBe('fp1');
	});

	it('removes entire group when all alerts are filtered out', () => {
		filterQuery.set('DiskFull');
		const result = get(filteredGrouped);
		expect(result).toHaveLength(1);
		expect(result[0].labels['severity']).toBe('warning');
	});

	it('returns empty array when no alerts match any group', () => {
		filterQuery.set('NonExistent');
		const result = get(filteredGrouped);
		expect(result).toHaveLength(0);
	});

	it('combines instanceFilter and text query across groups', () => {
		instanceFilter.set('prod-eu');
		filterQuery.set('CPUHigh');
		const result = get(filteredGrouped);
		expect(result).toHaveLength(1);
		expect(result[0].alerts).toHaveLength(1);
		expect(result[0].alerts[0].fingerprint).toBe('fp1');
	});

	it('updates group count to reflect filtered alerts', () => {
		filterQuery.set('MemHigh');
		const result = get(filteredGrouped);
		expect(result).toHaveLength(1);
		// count should reflect only the matching alert.
		expect(result[0].count).toBe(1);
	});

	it('passes through all groups structurally unchanged when filters are empty', () => {
		filterQuery.set('');
		instanceFilter.set('');
		const result = get(filteredGrouped);
		expect(result).toEqual([criticalGroup, warningGroup]);
		expect(result).toHaveLength(2);
	});

	it('returns empty array when alertsGrouped is empty', () => {
		alertsGrouped.set([]);
		const result = get(filteredGrouped);
		expect(result).toHaveLength(0);
	});

	it('filters by matcher syntax within groups', () => {
		filterQuery.set('severity=critical');
		const result = get(filteredGrouped);
		// a3 has severity=warning, so the warning group is removed.
		expect(result).toHaveLength(1);
		expect(result[0].alerts.every((a) => a.labels['severity'] === 'critical')).toBe(true);
	});
});

// ─── groupedAlerts — client-side grouping (derived from filteredAlerts) ───────

describe('groupedAlerts — client-side derived grouping (extended)', () => {
	const critical1 = makeAlert({ fingerprint: 'fc1', labels: { alertname: 'CPUHigh', severity: 'critical', team: 'infra' } });
	const critical2 = makeAlert({ fingerprint: 'fc2', labels: { alertname: 'MemHigh', severity: 'critical', team: 'app' } });
	const warning1  = makeAlert({ fingerprint: 'fw1', labels: { alertname: 'DiskLow', severity: 'warning', team: 'infra' } });
	const noSev     = makeAlert({ fingerprint: 'fn1', labels: { alertname: 'Heartbeat' } }); // no severity label

	beforeEach(() => {
		alerts.set([critical1, critical2, warning1, noSev]);
		filterQuery.set('');
		instanceFilter.set('');
		groupByLabel.set('severity');
	});

	it('groups by severity and counts correctly', () => {
		const groups = get(groupedAlerts);
		expect(groups.get('critical')).toHaveLength(2);
		expect(groups.get('warning')).toHaveLength(1);
	});

	it('falls back to (none) for missing severity label', () => {
		const groups = get(groupedAlerts);
		expect(groups.get('(none)')).toHaveLength(1);
		expect(groups.get('(none)')![0].fingerprint).toBe('fn1');
	});

	it('groups by team label', () => {
		groupByLabel.set('team');
		const groups = get(groupedAlerts);
		expect(groups.get('infra')).toHaveLength(2);
		expect(groups.get('app')).toHaveLength(1);
		// Heartbeat has no team label → (none)
		expect(groups.get('(none)')).toHaveLength(1);
	});

	it('reflects filterQuery in grouped output', () => {
		filterQuery.set('CPUHigh');
		const groups = get(groupedAlerts);
		expect(groups.get('critical')).toHaveLength(1);
		expect(groups.get('critical')![0].fingerprint).toBe('fc1');
		expect(groups.get('warning')).toBeUndefined();
	});

	it('reflects instanceFilter in grouped output', () => {
		const euAlert = makeAlert({ fingerprint: 'feu', alertmanager: 'eu', labels: { alertname: 'EUAlert', severity: 'info' } });
		const usAlert = makeAlert({ fingerprint: 'fus', alertmanager: 'us', labels: { alertname: 'USAlert', severity: 'info' } });
		alerts.set([euAlert, usAlert]);
		instanceFilter.set('eu');
		const groups = get(groupedAlerts);
		expect(groups.get('info')).toHaveLength(1);
		expect(groups.get('info')![0].fingerprint).toBe('feu');
	});

	it('produces an empty map when all alerts are filtered out', () => {
		filterQuery.set('NoMatch');
		const groups = get(groupedAlerts);
		expect(groups.size).toBe(0);
	});
});

// ─── validateFilterQuery — matcher syntax validation ─────────────────────────

describe('validateFilterQuery — syntax validation', () => {
	it('returns null for empty query', () => {
		expect(validateFilterQuery('')).toBeNull();
	});

	it('returns null for whitespace-only query', () => {
		expect(validateFilterQuery('   ')).toBeNull();
	});

	it('returns null for plain text (substring search)', () => {
		expect(validateFilterQuery('CPUHigh')).toBeNull();
	});

	it('returns null for valid equality matcher', () => {
		expect(validateFilterQuery('severity=critical')).toBeNull();
	});

	it('returns null for valid not-equal matcher', () => {
		expect(validateFilterQuery('env!=staging')).toBeNull();
	});

	it('returns null for valid regex matcher =~', () => {
		expect(validateFilterQuery('env=~prod.*')).toBeNull();
	});

	it('returns null for valid negated regex matcher !~', () => {
		expect(validateFilterQuery('env!~staging.*')).toBeNull();
	});

	it('returns null for multiple valid matchers', () => {
		expect(validateFilterQuery('severity=critical env=~prod.*')).toBeNull();
	});

	it('returns error string for invalid regex in =~ matcher', () => {
		const result = validateFilterQuery('env=~[invalid');
		expect(result).not.toBeNull();
		expect(result).toContain('Invalid regex');
	});

	it('returns error string for invalid regex in !~ matcher', () => {
		const result = validateFilterQuery('env!~[invalid');
		expect(result).not.toBeNull();
		expect(result).toContain('Invalid regex');
	});
});

// ─── filteredAlerts — !~ operator (negated regex) ─────────────────────────────

describe('filteredAlerts — negated regex operator !~', () => {
	const a1 = makeAlert({ fingerprint: 'fp1', labels: { alertname: 'CPUHigh', severity: 'critical', env: 'prod' } });
	const a2 = makeAlert({ fingerprint: 'fp2', labels: { alertname: 'MemHigh', severity: 'warning', env: 'staging' } });
	const a3 = makeAlert({ fingerprint: 'fp3', labels: { alertname: 'DiskFull', severity: 'critical', env: 'prod-us' } });

	beforeEach(() => {
		alerts.set([a1, a2, a3]);
		filterQuery.set('');
		instanceFilter.set('');
	});

	it('filters out alerts matching the negated regex', () => {
		filterQuery.set('env!~prod.*');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(1);
		expect(result[0].fingerprint).toBe('fp2');
	});

	it('returns all alerts when no alerts match the negated regex', () => {
		filterQuery.set('env!~staging.*');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(2);
		expect(result.map((a) => a.fingerprint)).not.toContain('fp2');
	});

	it('combines !~ with = matcher', () => {
		filterQuery.set('severity=critical env!~prod-us');
		const result = get(filteredAlerts);
		expect(result).toHaveLength(1);
		expect(result[0].fingerprint).toBe('fp1');
	});
});

// ─── availableLabels derived store ───────────────────────────────────────────

describe('availableLabels — dynamic label keys', () => {
	beforeEach(() => {
		alerts.set([]);
		filterQuery.set('');
	});

	it('returns empty array when no alerts loaded', () => {
		expect(get(availableLabels)).toEqual([]);
	});

	it('returns sorted unique label keys from all alerts', () => {
		const a1 = makeAlert({ fingerprint: 'fp1', labels: { alertname: 'A', severity: 'critical', env: 'prod' } });
		const a2 = makeAlert({ fingerprint: 'fp2', labels: { alertname: 'B', team: 'infra' } });
		alerts.set([a1, a2]);
		const keys = get(availableLabels);
		expect(keys).toContain('alertname');
		expect(keys).toContain('severity');
		expect(keys).toContain('env');
		expect(keys).toContain('team');
		// Should be sorted
		expect(keys).toEqual([...keys].sort());
	});

	it('deduplicates label keys across alerts', () => {
		const a1 = makeAlert({ fingerprint: 'fp1', labels: { severity: 'critical' } });
		const a2 = makeAlert({ fingerprint: 'fp2', labels: { severity: 'warning' } });
		alerts.set([a1, a2]);
		const keys = get(availableLabels);
		const severityCount = keys.filter((k) => k === 'severity').length;
		expect(severityCount).toBe(1);
	});

	it('updates reactively when alerts change', () => {
		alerts.set([makeAlert({ fingerprint: 'fp1', labels: { cluster: 'eu-west-1' } })]);
		expect(get(availableLabels)).toContain('cluster');
		alerts.set([]);
		expect(get(availableLabels)).toHaveLength(0);
	});
});
