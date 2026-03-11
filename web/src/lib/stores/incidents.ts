import { writable, derived } from 'svelte/store';
import type { Incident, IncidentListItem, IncidentStatus } from '$lib/api/types';
import {
	fetchIncidents,
	fetchIncident,
	createIncident,
	addIncidentEvent
} from '$lib/api/incidents';
import type { CreateIncidentRequest, AddEventRequest } from '$lib/api/incidents';

// ─── Raw data stores ──────────────────────────────────────────────────────────

/** Paginated list of incident summaries. */
export const incidents = writable<IncidentListItem[]>([]);

/** Total incident count (before pagination). */
export const incidentsTotal = writable(0);

/** Currently-selected incident (full object with timeline). */
export const selectedIncident = writable<Incident | null>(null);

export const incidentsLoading = writable(false);
export const incidentDetailLoading = writable(false);
export const incidentsError = writable<string | null>(null);

// ─── Filter/pagination state ─────────────────────────────────────────────────

export const incidentStatusFilter = writable<IncidentStatus | ''>('');
export const incidentAlertFilter = writable<string>('');
export const incidentsLimit = writable(100);
export const incidentsOffset = writable(0);

// ─── Derived: filtered incidents (client-side text filter) ───────────────────

export const incidentSearchQuery = writable('');

export const filteredIncidents = derived(
	[incidents, incidentSearchQuery],
	([$incidents, $q]) => {
		if (!$q.trim()) return $incidents;
		const lower = $q.toLowerCase();
		return $incidents.filter(
			(inc) =>
				inc.title.toLowerCase().includes(lower) ||
				inc.severity.toLowerCase().includes(lower) ||
				inc.status.toLowerCase().includes(lower) ||
				(inc.alert_fingerprint ?? '').toLowerCase().includes(lower) ||
				Object.values(inc.labels ?? {}).some((v) => v.toLowerCase().includes(lower))
		);
	}
);

// ─── Derived: incidents grouped by status ─────────────────────────────────────

export const incidentsByStatus = derived(incidents, ($incidents) => {
	const grouped: Record<IncidentStatus, IncidentListItem[]> = {
		OPEN: [],
		ACK: [],
		INVESTIGATING: [],
		RESOLVED: []
	};
	for (const inc of $incidents) {
		grouped[inc.status]?.push(inc);
	}
	return grouped;
});

// ─── Derived: active incident count (non-resolved) ───────────────────────────

export const activeIncidentCount = derived(incidents, ($incidents) =>
	$incidents.filter((inc) => inc.status !== 'RESOLVED').length
);

// ─── Actions ─────────────────────────────────────────────────────────────────

/**
 * Load the incident list applying current filter/pagination state.
 */
export async function loadIncidents(): Promise<void> {
	incidentsLoading.set(true);
	incidentsError.set(null);

	let statusFilter: IncidentStatus | '' = '';
	incidentStatusFilter.subscribe((v) => (statusFilter = v))();

	let alertFilter = '';
	incidentAlertFilter.subscribe((v) => (alertFilter = v))();

	let limit = 100;
	incidentsLimit.subscribe((v) => (limit = v))();

	let offset = 0;
	incidentsOffset.subscribe((v) => (offset = v))();

	try {
		const resp = await fetchIncidents({
			status: statusFilter || undefined,
			alertFingerprint: alertFilter || undefined,
			limit,
			offset
		});
		incidents.set(resp.incidents);
		incidentsTotal.set(resp.total);
	} catch (err) {
		incidentsError.set(err instanceof Error ? err.message : 'Failed to load incidents');
	} finally {
		incidentsLoading.set(false);
	}
}

/**
 * Load the full incident (with timeline) and set selectedIncident.
 */
export async function loadIncidentDetail(id: string): Promise<void> {
	incidentDetailLoading.set(true);
	try {
		const inc = await fetchIncident(id);
		selectedIncident.set(inc);
	} catch (err) {
		incidentsError.set(err instanceof Error ? err.message : 'Failed to load incident');
	} finally {
		incidentDetailLoading.set(false);
	}
}

/**
 * Create a new incident and reload the list.
 */
export async function openIncident(req: CreateIncidentRequest): Promise<Incident> {
	const inc = await createIncident(req);
	await loadIncidents();
	return inc;
}

/**
 * Append a lifecycle event and refresh the selectedIncident if it matches.
 */
export async function dispatchEvent(id: string, req: AddEventRequest): Promise<Incident> {
	const updated = await addIncidentEvent(id, req);

	// Update selectedIncident in-place if it's the same incident.
	selectedIncident.update((current) => (current?.id === id ? updated : current));

	// Refresh summary list to reflect new status / updated_at.
	await loadIncidents();

	return updated;
}
