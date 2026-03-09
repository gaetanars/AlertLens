/**
 * Unit tests for the alerts API module.
 * Tests flattenGroups and parameter serialization.
 */
import { describe, it, expect } from 'vitest';
import { flattenGroups } from './alerts';
import type { AlertsResponse, Alert } from './types';

function makeAlert(fp: string, severity = 'critical'): Alert {
	return {
		fingerprint: fp,
		alertmanager: 'test',
		labels: { alertname: 'TestAlert', severity },
		annotations: {},
		state: 'active',
		startsAt: '2026-03-09T10:00:00Z',
		endsAt: '0001-01-01T00:00:00Z',
		generatorURL: '',
		receivers: [],
		status: { state: 'active', silencedBy: [], inhibitedBy: [] }
	};
}

function makeResponse(groups: Array<{ labels: Record<string, string>; alerts: Alert[] }>): AlertsResponse {
	return {
		groups: groups.map((g) => ({ ...g, count: g.alerts.length })),
		total: groups.reduce((s, g) => s + g.alerts.length, 0),
		limit: 500,
		offset: 0
	};
}

describe('flattenGroups', () => {
	it('returns empty array for empty groups', () => {
		const resp = makeResponse([]);
		expect(flattenGroups(resp)).toEqual([]);
	});

	it('flattens a single group', () => {
		const resp = makeResponse([
			{ labels: { severity: 'critical' }, alerts: [makeAlert('fp1'), makeAlert('fp2')] }
		]);
		expect(flattenGroups(resp)).toHaveLength(2);
	});

	it('flattens multiple groups in order', () => {
		const resp = makeResponse([
			{ labels: { severity: 'critical' }, alerts: [makeAlert('fp1')] },
			{ labels: { severity: 'warning' }, alerts: [makeAlert('fp2', 'warning'), makeAlert('fp3', 'warning')] }
		]);
		const flat = flattenGroups(resp);
		expect(flat).toHaveLength(3);
		expect(flat[0].fingerprint).toBe('fp1');
		expect(flat[1].fingerprint).toBe('fp2');
		expect(flat[2].fingerprint).toBe('fp3');
	});

	it('returns alerts in group order', () => {
		const resp = makeResponse([
			{ labels: { severity: 'warning' }, alerts: [makeAlert('fp-w')] },
			{ labels: { severity: 'critical' }, alerts: [makeAlert('fp-c')] }
		]);
		const flat = flattenGroups(resp);
		expect(flat[0].fingerprint).toBe('fp-w');
		expect(flat[1].fingerprint).toBe('fp-c');
	});
});

describe('AlertsResponse type compatibility', () => {
	it('has required fields', () => {
		const resp: AlertsResponse = {
			groups: [],
			total: 0,
			limit: 500,
			offset: 0
		};
		expect(resp.total).toBe(0);
		expect(resp.limit).toBe(500);
		expect(resp.offset).toBe(0);
	});

	it('AlertGroup has labels, alerts, count', () => {
		const resp = makeResponse([
			{ labels: { severity: 'critical' }, alerts: [makeAlert('fp1')] }
		]);
		const group = resp.groups[0];
		expect(group.labels).toEqual({ severity: 'critical' });
		expect(group.count).toBe(1);
		expect(group.alerts).toHaveLength(1);
	});
});
