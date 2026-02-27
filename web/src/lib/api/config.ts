import { api } from './client';
import type { ConfigResponse, ValidationResult, SaveConfigRequest } from './types';

export function fetchConfig(instance?: string): Promise<ConfigResponse> {
	const q = instance ? `?instance=${encodeURIComponent(instance)}` : '';
	return api.get<ConfigResponse>(`/config${q}`);
}

export function validateConfig(rawYaml: string): Promise<ValidationResult> {
	return api.post<ValidationResult>('/config/validate', { raw_yaml: rawYaml });
}

export function diffConfig(
	alertmanager: string,
	proposedYaml: string
): Promise<{ diff: string; has_changes: boolean }> {
	return api.post('/config/diff', {
		alertmanager,
		proposed_yaml: proposedYaml
	});
}

export function saveConfig(req: SaveConfigRequest): Promise<{
	saved: boolean;
	mode: string;
	commit_sha?: string;
	html_url?: string;
	warning?: string;
}> {
	return api.post('/config/save', req);
}
