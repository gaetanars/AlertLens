/**
 * bulk.ts — client for POST /api/v1/bulk (ADR-007)
 *
 * Sends selected alerts to the backend for smart-merge silencing or acking.
 * The backend computes the optimal silence strategy (merged vs individual) so
 * the caller does not need to pre-compute matchers.
 */

import { api } from './client';
import type { Alert } from './types';

// ─── Request / response types ─────────────────────────────────────────────────

/** A minimal alert reference sent to the bulk endpoint. */
export interface BulkAlertRef {
	fingerprint: string;
	alertmanager: string;
	labels: Record<string, string>;
}

/** Payload for POST /api/v1/bulk */
export interface BulkActionRequest {
	/** "silence" (Alertmanager silence) or "ack" (visual ack). */
	action: 'silence' | 'ack';
	/** Alert references to silence / ack. */
	alerts: BulkAlertRef[];
	/** ISO datetime for silence expiry. Omit to default to now+1h on the server. */
	ends_at?: string;
	/** Username recorded on the silence. */
	created_by?: string;
	/** Free-text reason. */
	comment?: string;
}

/** Response from POST /api/v1/bulk */
export interface BulkActionResponse {
	/** Alertmanager silence IDs created. */
	silence_ids: string[];
	/**
	 * Strategy used:
	 * - `"merged"` — one silence per alertmanager instance (common matchers found)
	 * - `"individual"` — one silence per alert (no shared labels)
	 */
	strategy: 'merged' | 'individual';
	/** Number of silences created (equals silence_ids.length). */
	count: number;
}

// ─── API helper ───────────────────────────────────────────────────────────────

/**
 * Send a bulk silence / ack request.
 * Uses POST /api/v1/bulk with smart-merge matching on the backend (ADR-007).
 */
export function bulkAction(req: BulkActionRequest): Promise<BulkActionResponse> {
	return api.post<BulkActionResponse>('/v1/bulk', req);
}

// ─── Convenience wrappers ─────────────────────────────────────────────────────

/** Convert a list of Alert objects into BulkAlertRef objects. */
export function alertsToRefs(alerts: Alert[]): BulkAlertRef[] {
	return alerts.map((a) => ({
		fingerprint: a.fingerprint,
		alertmanager: a.alertmanager,
		labels: a.labels
	}));
}

/**
 * Bulk-silence selected alerts using smart-merge.
 * `endsAt` is optional — defaults to now+1h on the backend.
 */
export function bulkSilence(
	alerts: Alert[],
	opts: { endsAt?: Date; createdBy?: string; comment?: string } = {}
): Promise<BulkActionResponse> {
	const req: BulkActionRequest = {
		action: 'silence',
		alerts: alertsToRefs(alerts),
		...(opts.endsAt && { ends_at: opts.endsAt.toISOString() }),
		...(opts.createdBy && { created_by: opts.createdBy }),
		...(opts.comment && { comment: opts.comment })
	};
	return bulkAction(req);
}

/**
 * Bulk-ack (visual ack) selected alerts using smart-merge.
 */
export function bulkAck(
	alerts: Alert[],
	opts: { endsAt?: Date; createdBy?: string; comment?: string } = {}
): Promise<BulkActionResponse> {
	const req: BulkActionRequest = {
		action: 'ack',
		alerts: alertsToRefs(alerts),
		...(opts.endsAt && { ends_at: opts.endsAt.toISOString() }),
		...(opts.createdBy && { created_by: opts.createdBy }),
		...(opts.comment && { comment: opts.comment })
	};
	return bulkAction(req);
}
