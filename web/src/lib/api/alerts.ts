import { api } from './client';
import type { Alert, InstanceStatus } from './types';

export interface AlertsParams {
	filter?: string[];
	instance?: string;
	silenced?: boolean;
	inhibited?: boolean;
	active?: boolean;
}

export function fetchAlerts(params: AlertsParams = {}): Promise<Alert[]> {
	const q = new URLSearchParams();
	params.filter?.forEach((f) => q.append('filter', f));
	if (params.instance) q.set('instance', params.instance);
	if (params.silenced !== undefined) q.set('silenced', String(params.silenced));
	if (params.inhibited !== undefined) q.set('inhibited', String(params.inhibited));
	if (params.active !== undefined) q.set('active', String(params.active));
	const qs = q.toString();
	return api.get<Alert[]>(`/alerts${qs ? '?' + qs : ''}`);
}

export function fetchAlertmanagers(): Promise<InstanceStatus[]> {
	return api.get<InstanceStatus[]>('/alertmanagers');
}
