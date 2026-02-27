import { api } from './client';
import type { Silence, CreateSilenceRequest } from './types';

export interface SilencesParams {
	instance?: string;
	type?: 'silence' | 'ack';
}

export function fetchSilences(params: SilencesParams = {}): Promise<Silence[]> {
	const q = new URLSearchParams();
	if (params.instance) q.set('instance', params.instance);
	if (params.type) q.set('type', params.type);
	const qs = q.toString();
	return api.get<Silence[]>(`/silences${qs ? '?' + qs : ''}`);
}

export function createSilence(req: CreateSilenceRequest): Promise<{ silence_id: string }> {
	return api.post('/silences', req);
}

export function updateSilence(
	id: string,
	req: CreateSilenceRequest
): Promise<{ silence_id: string }> {
	return api.put(`/silences/${id}`, req);
}

export function expireSilence(id: string, instance: string): Promise<void> {
	return api.delete(`/silences/${id}?instance=${encodeURIComponent(instance)}`);
}
