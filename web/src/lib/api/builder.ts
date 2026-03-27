import { api } from './client';
import type {
	RouteSpec,
	ReceiverDef,
	TimeIntervalEntry,
	BuilderReceiverRoutesResponse,
	ValidationResult
} from './types';

export interface ExportConfigRequest {
	instance?: string;
	route?: RouteSpec;
	receivers?: ReceiverDef[];
	time_intervals?: TimeIntervalEntry[];
}

export interface BuilderRouteResponse {
	route: RouteSpec;
	raw_yaml: string;
	validation: ValidationResult;
}

export function getRoute(instance?: string): Promise<{ route: RouteSpec }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/route${q}`);
}

export function setRoute(
	instance: string,
	route: RouteSpec
): Promise<BuilderRouteResponse> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.put(`/builder/route${q}`, route);
}

export function exportConfig(
	req: ExportConfigRequest
): Promise<{ raw_yaml: string; validation: ValidationResult }> {
	return api.post('/builder/export', req);
}

export function listReceivers(instance?: string): Promise<{ receivers: ReceiverDef[] }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/receivers${q}`);
}

export function getReceiver(name: string, instance?: string): Promise<ReceiverDef> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/receivers/${encodeURIComponent(name)}${q}`);
}

export function upsertReceiver(
	name: string,
	rec: ReceiverDef,
	instance?: string
): Promise<{ receiver: ReceiverDef; raw_yaml: string; validation: ValidationResult }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.put(`/builder/receivers/${encodeURIComponent(name)}${q}`, rec);
}

export function deleteReceiver(
	name: string,
	instance?: string
): Promise<{ deleted: string; raw_yaml: string; validation: ValidationResult }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.delete(`/builder/receivers/${encodeURIComponent(name)}${q}`);
}

export function validateReceiver(rec: ReceiverDef): Promise<ValidationResult> {
	return api.post('/builder/receivers/validate', rec);
}

export function getReceiverRoutes(
	name: string,
	instance?: string
): Promise<BuilderReceiverRoutesResponse> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/receivers/${encodeURIComponent(name)}/routes${q}`);
}

export function listTimeIntervals(
	instance?: string
): Promise<{ time_intervals: TimeIntervalEntry[] }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/time-intervals${q}`);
}

export function getTimeInterval(name: string, instance?: string): Promise<TimeIntervalEntry> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/time-intervals/${encodeURIComponent(name)}${q}`);
}

export function upsertTimeInterval(
	name: string,
	entry: TimeIntervalEntry,
	instance?: string
): Promise<{ time_interval: TimeIntervalEntry; raw_yaml: string; validation: ValidationResult }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.put(`/builder/time-intervals/${encodeURIComponent(name)}${q}`, entry);
}

export function deleteTimeInterval(
	name: string,
	instance?: string
): Promise<{ deleted: string; raw_yaml: string; validation: ValidationResult }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.delete(`/builder/time-intervals/${encodeURIComponent(name)}${q}`);
}

export function validateTimeInterval(entry: TimeIntervalEntry): Promise<ValidationResult> {
	return api.post('/builder/time-intervals/validate', entry);
}
