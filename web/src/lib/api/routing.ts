import { api } from './client';
import type { HubTopology, RouteNode } from './types';

export function fetchRouting(
	instance?: string,
	annotateAlerts?: boolean
): Promise<{ alertmanager: string; route: RouteNode }> {
	const params = new URLSearchParams();
	if (instance) params.set('instance', instance);
	if (annotateAlerts) params.set('annotate_alerts', 'true');
	const q = params.size > 0 ? '?' + params.toString() : '';
	return api.get(`/routing${q}`);
}

export function matchRouting(
	alertmanager: string,
	labels: Record<string, string>
): Promise<{ matched_routes: RouteNode[] }> {
	return api.post('/routing/match', { alertmanager, labels });
}

/** Fetch the hub-and-spoke topology with per-instance stats. */
export function fetchHubTopology(): Promise<HubTopology> {
	return api.get('/hub/topology');
}
