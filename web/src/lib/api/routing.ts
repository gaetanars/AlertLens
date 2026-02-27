import { api } from './client';
import type { RouteNode } from './types';

export function fetchRouting(instance?: string): Promise<{ alertmanager: string; route: RouteNode }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/routing${q}`);
}

export function matchRouting(
	alertmanager: string,
	labels: Record<string, string>
): Promise<{ matched_routes: RouteNode[] }> {
	return api.post('/routing/match', { alertmanager, labels });
}
