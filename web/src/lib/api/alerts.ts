import { api } from './client';
import type { Alert, AlertsResponse, InstanceStatus } from './types';

// ─── Query parameter types ────────────────────────────────────────────────────

export interface AlertsParams {
	/** Alertmanager matcher strings, e.g. ['severity="critical"'] */
	filter?: string[];
	/** Filter to a single alertmanager instance name */
	instance?: string;
	silenced?: boolean;
	inhibited?: boolean;
	active?: boolean;
	/** Filter by severity label (view-layer, not forwarded to AM) */
	severity?: string[];
	/** Filter by alert state: active | suppressed | unprocessed */
	status?: string[];
	/** Group results by label key(s) */
	groupBy?: string[];
	/** Max alerts to return (default 500) */
	limit?: number;
	/** Pagination offset */
	offset?: number;
}

// ─── API functions ────────────────────────────────────────────────────────────

/**
 * Fetch alerts with optional filtering, grouping, and pagination.
 * Returns the structured AlertsResponse (with groups).
 */
export function fetchAlerts(params: AlertsParams = {}): Promise<AlertsResponse> {
	const q = new URLSearchParams();
	params.filter?.forEach((f) => q.append('filter', f));
	if (params.instance) q.set('instance', params.instance);
	if (params.silenced !== undefined) q.set('silenced', String(params.silenced));
	if (params.inhibited !== undefined) q.set('inhibited', String(params.inhibited));
	if (params.active !== undefined) q.set('active', String(params.active));
	params.severity?.forEach((s) => q.append('severity', s));
	params.status?.forEach((s) => q.append('status', s));
	params.groupBy?.forEach((g) => q.append('group_by', g));
	if (params.limit !== undefined) q.set('limit', String(params.limit));
	if (params.offset !== undefined) q.set('offset', String(params.offset));
	const qs = q.toString();
	return api.get<AlertsResponse>(`/alerts${qs ? '?' + qs : ''}`);
}

/**
 * Fetch a flat list of alerts (unwraps groups from AlertsResponse).
 * Convenience helper for components that don't need grouping.
 */
export async function fetchFlatAlerts(params: AlertsParams = {}): Promise<Alert[]> {
	const resp = await fetchAlerts(params);
	return flattenGroups(resp);
}

/**
 * Flatten an AlertsResponse into a flat Alert array.
 */
export function flattenGroups(resp: AlertsResponse): Alert[] {
	return resp.groups.flatMap((g) => g.alerts);
}

export function fetchAlertmanagers(): Promise<InstanceStatus[]> {
	return api.get<InstanceStatus[]>('/alertmanagers');
}
