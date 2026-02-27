import { writable } from 'svelte/store';
import type { Silence } from '$lib/api/types';
import { fetchSilences } from '$lib/api/silences';

export const silences = writable<Silence[]>([]);
export const silencesLoading = writable(false);
export const silencesError = writable<string | null>(null);

export async function loadSilences(instance?: string) {
	silencesLoading.set(true);
	silencesError.set(null);
	try {
		const data = await fetchSilences({ instance });
		silences.set(data ?? []);
	} catch (e) {
		silencesError.set(e instanceof Error ? e.message : 'Failed to load silences');
	} finally {
		silencesLoading.set(false);
	}
}
