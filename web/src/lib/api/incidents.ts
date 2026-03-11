import { api } from './client';
import type {
	Incident,
	IncidentListItem,
	IncidentEvent,
	IncidentStatus,
	IncidentEventKind,
	ListIncidentsResponse,
	CreateIncidentRequest,
	AddEventRequest,
	IncidentTimelineResponse
} from './types';

export type {
	Incident,
	IncidentListItem,
	IncidentEvent,
	IncidentStatus,
	IncidentEventKind,
	ListIncidentsResponse,
	CreateIncidentRequest,
	AddEventRequest,
	IncidentTimelineResponse
};

// ─── Query parameter types ────────────────────────────────────────────────────

export interface IncidentsParams {
	/** Filter by lifecycle status */
	status?: IncidentStatus;
	/** Filter to incidents linked to a specific alert fingerprint */
	alertFingerprint?: string;
	/** Max results to return (default 100, max 500) */
	limit?: number;
	/** Pagination offset */
	offset?: number;
}

// ─── API functions ────────────────────────────────────────────────────────────

/**
 * Fetch a paginated list of incidents.
 */
export function fetchIncidents(params: IncidentsParams = {}): Promise<ListIncidentsResponse> {
	const q = new URLSearchParams();
	if (params.status) q.set('status', params.status);
	if (params.alertFingerprint) q.set('alert_fingerprint', params.alertFingerprint);
	if (params.limit !== undefined) q.set('limit', String(params.limit));
	if (params.offset !== undefined) q.set('offset', String(params.offset));
	const qs = q.toString();
	return api.get<ListIncidentsResponse>(`/incidents${qs ? '?' + qs : ''}`);
}

/**
 * Fetch a single incident (includes full event timeline).
 */
export function fetchIncident(id: string): Promise<Incident> {
	return api.get<Incident>(`/incidents/${encodeURIComponent(id)}`);
}

/**
 * Fetch only the event timeline for an incident (lightweight polling endpoint).
 */
export function fetchIncidentTimeline(id: string): Promise<IncidentTimelineResponse> {
	return api.get<IncidentTimelineResponse>(
		`/incidents/${encodeURIComponent(id)}/timeline`
	);
}

/**
 * Create a new incident.
 */
export function createIncident(req: CreateIncidentRequest): Promise<Incident> {
	return api.post<Incident>('/incidents', req);
}

/**
 * Append a lifecycle event (ACK, INVESTIGATING, RESOLVED, REOPENED, COMMENT).
 * Returns the updated incident.
 */
export function addIncidentEvent(id: string, req: AddEventRequest): Promise<Incident> {
	return api.post<Incident>(`/incidents/${encodeURIComponent(id)}/events`, req);
}

// ─── Convenience lifecycle helpers ───────────────────────────────────────────

export const ackIncident = (id: string, actor: string, message?: string) =>
	addIncidentEvent(id, { kind: 'ACK', actor, message });

export const startInvestigating = (id: string, actor: string, message?: string) =>
	addIncidentEvent(id, { kind: 'INVESTIGATING', actor, message });

export const resolveIncident = (id: string, actor: string, message?: string) =>
	addIncidentEvent(id, { kind: 'RESOLVED', actor, message });

export const reopenIncident = (id: string, actor: string, message?: string) =>
	addIncidentEvent(id, { kind: 'REOPENED', actor, message });

export const commentOnIncident = (id: string, actor: string, message: string) =>
	addIncidentEvent(id, { kind: 'COMMENT', actor, message });
