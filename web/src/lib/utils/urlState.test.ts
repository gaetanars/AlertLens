/**
 * Unit tests for urlState helpers.
 *
 * ADR-006: All alert view state is persisted in URL search params so that
 * hard-refresh and link-sharing reproduce the exact same view.
 *
 * These tests cover the acceptance criterion: "User can switch between Kanban
 * and List view; preference is remembered across page reloads" (issue #45).
 */
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { parseAlertURLState, buildAlertURLParams, syncURLState } from './urlState';
import type { AlertURLState } from './urlState';

// ─── parseAlertURLState ───────────────────────────────────────────────────────

describe('parseAlertURLState — default values', () => {
	it('returns kanban view when param is absent', () => {
		const result = parseAlertURLState(new URLSearchParams(''));
		expect(result.view).toBe('kanban');
	});

	it('returns empty query when q is absent', () => {
		expect(parseAlertURLState(new URLSearchParams('')).q).toBe('');
	});

	it('returns empty instance when param is absent', () => {
		expect(parseAlertURLState(new URLSearchParams('')).instance).toBe('');
	});

	it('returns empty severity array when param is absent', () => {
		expect(parseAlertURLState(new URLSearchParams('')).severity).toEqual([]);
	});

	it('returns empty status array when param is absent', () => {
		expect(parseAlertURLState(new URLSearchParams('')).status).toEqual([]);
	});

	it('returns severity groupBy by default', () => {
		expect(parseAlertURLState(new URLSearchParams('')).groupBy).toBe('severity');
	});

	it('returns startsAt sort by default', () => {
		expect(parseAlertURLState(new URLSearchParams('')).sort).toBe('startsAt');
	});

	it('returns desc sortDir by default', () => {
		expect(parseAlertURLState(new URLSearchParams('')).sortDir).toBe('desc');
	});
});

describe('parseAlertURLState — view mode persistence', () => {
	it('parses view=kanban', () => {
		const result = parseAlertURLState(new URLSearchParams('view=kanban'));
		expect(result.view).toBe('kanban');
	});

	it('parses view=list', () => {
		const result = parseAlertURLState(new URLSearchParams('view=list'));
		expect(result.view).toBe('list');
	});

	it('falls back to kanban for invalid view value', () => {
		const result = parseAlertURLState(new URLSearchParams('view=grid'));
		expect(result.view).toBe('kanban');
	});

	it('falls back to kanban for empty view value', () => {
		const result = parseAlertURLState(new URLSearchParams('view='));
		expect(result.view).toBe('kanban');
	});
});

describe('parseAlertURLState — filter params', () => {
	it('parses q param', () => {
		const result = parseAlertURLState(new URLSearchParams('q=CPUHigh'));
		expect(result.q).toBe('CPUHigh');
	});

	it('parses instance param', () => {
		const result = parseAlertURLState(new URLSearchParams('instance=prod-eu'));
		expect(result.instance).toBe('prod-eu');
	});

	it('parses severity as comma-separated array', () => {
		const result = parseAlertURLState(new URLSearchParams('severity=critical,warning'));
		expect(result.severity).toEqual(['critical', 'warning']);
	});

	it('parses single severity value', () => {
		const result = parseAlertURLState(new URLSearchParams('severity=critical'));
		expect(result.severity).toEqual(['critical']);
	});

	it('parses status as comma-separated array', () => {
		const result = parseAlertURLState(new URLSearchParams('status=active,suppressed'));
		expect(result.status).toEqual(['active', 'suppressed']);
	});

	it('parses groupBy param', () => {
		const result = parseAlertURLState(new URLSearchParams('groupBy=alertname'));
		expect(result.groupBy).toBe('alertname');
	});

	it('parses all params together', () => {
		const params = new URLSearchParams(
			'view=list&q=cpu&instance=prod-eu&severity=critical&status=active&groupBy=alertname&sort=alertname&sortDir=asc'
		);
		const result = parseAlertURLState(params);
		expect(result).toEqual({
			view: 'list',
			q: 'cpu',
			instance: 'prod-eu',
			severity: ['critical'],
			status: ['active'],
			groupBy: 'alertname',
			sort: 'alertname',
			sortDir: 'asc'
		});
	});
});

describe('parseAlertURLState — sort params', () => {
	it('parses sort=alertname', () => {
		expect(parseAlertURLState(new URLSearchParams('sort=alertname')).sort).toBe('alertname');
	});

	it('parses sort=severity', () => {
		expect(parseAlertURLState(new URLSearchParams('sort=severity')).sort).toBe('severity');
	});

	it('parses sort=alertmanager', () => {
		expect(parseAlertURLState(new URLSearchParams('sort=alertmanager')).sort).toBe('alertmanager');
	});

	it('falls back to startsAt for invalid sort value', () => {
		expect(parseAlertURLState(new URLSearchParams('sort=foobar')).sort).toBe('startsAt');
	});

	it('parses sortDir=asc', () => {
		expect(parseAlertURLState(new URLSearchParams('sortDir=asc')).sortDir).toBe('asc');
	});

	it('falls back to desc for invalid sortDir', () => {
		expect(parseAlertURLState(new URLSearchParams('sortDir=random')).sortDir).toBe('desc');
	});
});

// ─── buildAlertURLParams ──────────────────────────────────────────────────────

describe('buildAlertURLParams — omits defaults', () => {
	const defaults: AlertURLState = {
		view: 'kanban',
		q: '',
		instance: '',
		severity: [],
		status: [],
		groupBy: 'severity',
		sort: 'startsAt',
		sortDir: 'desc'
	};

	it('produces empty params for default state', () => {
		const p = buildAlertURLParams(defaults);
		expect(p.toString()).toBe('');
	});

	it('includes view=list when not default', () => {
		const p = buildAlertURLParams({ ...defaults, view: 'list' });
		expect(p.get('view')).toBe('list');
	});

	it('omits view param when kanban (default)', () => {
		const p = buildAlertURLParams({ ...defaults, view: 'kanban' });
		expect(p.has('view')).toBe(false);
	});

	it('includes q when non-empty', () => {
		const p = buildAlertURLParams({ ...defaults, q: 'cpu' });
		expect(p.get('q')).toBe('cpu');
	});

	it('omits q when empty', () => {
		const p = buildAlertURLParams({ ...defaults, q: '' });
		expect(p.has('q')).toBe(false);
	});

	it('includes instance when non-empty', () => {
		const p = buildAlertURLParams({ ...defaults, instance: 'prod-eu' });
		expect(p.get('instance')).toBe('prod-eu');
	});

	it('omits instance when empty', () => {
		const p = buildAlertURLParams({ ...defaults, instance: '' });
		expect(p.has('instance')).toBe(false);
	});

	it('serialises severity array as comma-joined string', () => {
		const p = buildAlertURLParams({ ...defaults, severity: ['critical', 'warning'] });
		expect(p.get('severity')).toBe('critical,warning');
	});

	it('omits severity when empty array', () => {
		const p = buildAlertURLParams({ ...defaults, severity: [] });
		expect(p.has('severity')).toBe(false);
	});

	it('serialises status array as comma-joined string', () => {
		const p = buildAlertURLParams({ ...defaults, status: ['active', 'suppressed'] });
		expect(p.get('status')).toBe('active,suppressed');
	});

	it('includes groupBy when not default', () => {
		const p = buildAlertURLParams({ ...defaults, groupBy: 'alertname' });
		expect(p.get('groupBy')).toBe('alertname');
	});

	it('omits groupBy when severity (default)', () => {
		const p = buildAlertURLParams({ ...defaults, groupBy: 'severity' });
		expect(p.has('groupBy')).toBe(false);
	});

	it('includes sort when not default', () => {
		const p = buildAlertURLParams({ ...defaults, sort: 'alertname' });
		expect(p.get('sort')).toBe('alertname');
	});

	it('omits sort when startsAt (default)', () => {
		const p = buildAlertURLParams({ ...defaults, sort: 'startsAt' });
		expect(p.has('sort')).toBe(false);
	});

	it('includes sortDir when asc (non-default)', () => {
		const p = buildAlertURLParams({ ...defaults, sortDir: 'asc' });
		expect(p.get('sortDir')).toBe('asc');
	});

	it('omits sortDir when desc (default)', () => {
		const p = buildAlertURLParams({ ...defaults, sortDir: 'desc' });
		expect(p.has('sortDir')).toBe(false);
	});
});

describe('buildAlertURLParams — round-trip with parseAlertURLState', () => {
	it('round-trips list view state', () => {
		const original: AlertURLState = {
			view: 'list',
			q: 'cpu',
			instance: 'prod-eu',
			severity: ['critical', 'warning'],
			status: ['active'],
			groupBy: 'alertname',
			sort: 'alertname',
			sortDir: 'asc'
		};
		const params = buildAlertURLParams(original);
		const parsed = parseAlertURLState(params);
		expect(parsed).toEqual(original);
	});

	it('round-trips kanban view state with all defaults', () => {
		const original: AlertURLState = {
			view: 'kanban',
			q: '',
			instance: '',
			severity: [],
			status: [],
			groupBy: 'severity',
			sort: 'startsAt',
			sortDir: 'desc'
		};
		const params = buildAlertURLParams(original);
		const parsed = parseAlertURLState(params);
		expect(parsed).toEqual(original);
	});

	it('round-trips partial non-default state', () => {
		const original: AlertURLState = {
			view: 'kanban',
			q: 'MemHigh',
			instance: '',
			severity: ['critical'],
			status: [],
			groupBy: 'severity',
			sort: 'startsAt',
			sortDir: 'desc'
		};
		const params = buildAlertURLParams(original);
		const parsed = parseAlertURLState(params);
		expect(parsed).toEqual(original);
	});
});

// ─── syncURLState — browser history integration ───────────────────────────────

describe('syncURLState — browser history', () => {
	const defaults: AlertURLState = {
		view: 'kanban',
		q: '',
		instance: '',
		severity: [],
		status: [],
		groupBy: 'severity',
		sort: 'startsAt',
		sortDir: 'desc'
	};

	beforeEach(() => {
		// Simulate a browser environment with history API.
		Object.defineProperty(window, 'location', {
			value: { pathname: '/alerts', search: '' },
			writable: true
		});
		vi.spyOn(window.history, 'replaceState').mockImplementation(() => undefined);
		vi.spyOn(window.history, 'pushState').mockImplementation(() => undefined);
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('calls replaceState when push=false (default)', () => {
		syncURLState(defaults, false);
		expect(window.history.replaceState).toHaveBeenCalledTimes(1);
		expect(window.history.pushState).not.toHaveBeenCalled();
	});

	it('calls pushState when push=true', () => {
		syncURLState({ ...defaults, view: 'list' }, true);
		expect(window.history.pushState).toHaveBeenCalledTimes(1);
		expect(window.history.replaceState).not.toHaveBeenCalled();
	});

	it('uses bare pathname when state is all-defaults (no query string)', () => {
		syncURLState(defaults, false);
		const [, , url] = (window.history.replaceState as ReturnType<typeof vi.spyOn>).mock.calls[0];
		expect(url).toBe('/alerts');
	});

	it('appends query string for non-default view=list', () => {
		syncURLState({ ...defaults, view: 'list' }, false);
		const [, , url] = (window.history.replaceState as ReturnType<typeof vi.spyOn>).mock.calls[0];
		expect(url).toContain('view=list');
	});

	it('appends query string for active filter', () => {
		syncURLState({ ...defaults, q: 'CPUHigh' }, false);
		const [, , url] = (window.history.replaceState as ReturnType<typeof vi.spyOn>).mock.calls[0];
		expect(url).toContain('q=CPUHigh');
	});
});
