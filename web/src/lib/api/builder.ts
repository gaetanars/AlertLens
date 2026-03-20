import { api } from './client';
import type { RouteSpec, BuilderReceiverDef, ValidationResult } from './types';

export interface ExportConfigRequest {
	instance?: string;
	route?: RouteSpec;
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

export function listBuilderReceivers(
	instance?: string
): Promise<{ receivers: BuilderReceiverDef[] }> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get(`/builder/receivers${q}`);
}
