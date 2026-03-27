// ─── Shared types matching the Go backend models ─────────────────────────────

export interface Matcher {
	name: string;
	value: string;
	isRegex: boolean;
	isEqual: boolean;
}

export interface Ack {
	active: boolean;
	by: string;
	comment: string;
	silence_id: string;
}

export interface AlertStatus {
	state: 'active' | 'suppressed' | 'unprocessed';
	silencedBy: string[];
	inhibitedBy: string[];
}

export interface Alert {
	fingerprint: string;
	/** Name of the Alertmanager instance this alert originated from. */
	alertmanager: string;
	/**
	 * InstanceID is an alias for alertmanager.
	 * Present in API responses from Feature #2 onward.
	 */
	instance_id?: string;
	labels: Record<string, string>;
	annotations: Record<string, string>;
	state: string;
	startsAt: string;
	endsAt: string;
	updatedAt?: string;
	generatorURL: string;
	receivers: { name: string }[];
	status: AlertStatus;
	ack?: Ack;
}

export interface SilenceStatus {
	state: 'active' | 'pending' | 'expired';
}

export interface Silence {
	id: string;
	alertmanager: string;
	matchers: Matcher[];
	startsAt: string;
	endsAt: string;
	createdBy: string;
	comment: string;
	status: SilenceStatus;
}

// ─── Alert views API ──────────────────────────────────────────────────────────

/** A group of alerts sharing the same groupBy label values. */
export interface AlertGroup {
	/** Grouping key-value pairs (empty when no group_by was specified). */
	labels: Record<string, string>;
	alerts: Alert[];
	count: number;
}

/** Per-instance error when one Alertmanager instance failed to respond. */
export interface InstanceError {
	instance: string;
	error: string;
}

/** Top-level response from GET /api/alerts with grouping & pagination. */
export interface AlertsResponse {
	groups: AlertGroup[];
	/** Total alerts before pagination. */
	total: number;
	limit: number;
	offset: number;
	/**
	 * partial_failures is non-empty when one or more instances failed.
	 * The response still contains alerts from healthy instances (degraded mode).
	 */
	partial_failures?: InstanceError[];
}

export interface InstanceStatus {
	name: string;
	url: string;
	healthy: boolean;
	version: string;
	has_tenant: boolean;
	error?: string;
}

export interface RouteNode {
	receiver: string;
	matchers: Matcher[];
	group_by: string[];
	continue: boolean;
	group_wait?: string;
	group_interval?: string;
	repeat_interval?: string;
	mute_time_intervals?: string[];
	active_time_intervals?: string[];
	routes: RouteNode[];
	/** Number of active alerts that match this node's matchers (populated when annotate_alerts=true). */
	alert_count?: number;
	/** Per-severity alert counts for this node (populated when annotate_alerts=true). */
	severity_counts?: Record<string, number>;
}

// ─── Hub topology types ───────────────────────────────────────────────────────

/** Per-instance statistics returned by GET /api/hub/topology. */
export interface SpokeStats {
	name: string;
	url: string;
	healthy: boolean;
	version: string;
	error?: string;
	alert_count: number;
	active_count: number;
	suppressed_count: number;
	severity_counts: Record<string, number>;
}

/** Hub-level aggregate stats. */
export interface HubStats {
	name: string;
	total_instances: number;
	healthy_instances: number;
	total_alerts: number;
	critical_alerts: number;
}

/** Response envelope for GET /api/hub/topology. */
export interface HubTopology {
	hub: HubStats;
	spokes: SpokeStats[];
}

// ─── Incident tracking types ─────────────────────────────────────────────────

/**
 * Lifecycle status of an incident.
 * Valid transitions enforced by the backend state machine:
 *   OPEN → ACK | INVESTIGATING | RESOLVED
 *   ACK  → INVESTIGATING | RESOLVED | OPEN (reopen)
 *   INVESTIGATING → RESOLVED | OPEN (reopen)
 *   RESOLVED → OPEN (reopen)
 */
export type IncidentStatus = 'OPEN' | 'ACK' | 'INVESTIGATING' | 'RESOLVED';

/** Classifies each entry in the immutable event ledger. */
export type IncidentEventKind =
	| 'CREATED'
	| 'ACK'
	| 'INVESTIGATING'
	| 'RESOLVED'
	| 'REOPENED'
	| 'COMMENT';

/** Single immutable entry in the incident event ledger. */
export interface IncidentEvent {
	/** 1-based sequence number within the incident. */
	seq: number;
	kind: IncidentEventKind;
	/** Status after this event (empty string for COMMENT events). */
	status: IncidentStatus | '';
	actor: string;
	message?: string;
	occurred_at: string; // ISO 8601
}

/** Full incident with complete event log. Returned by GET /api/incidents/{id}. */
export interface Incident {
	id: string;
	title: string;
	severity: string;
	alert_fingerprint?: string;
	alertmanager_instance?: string;
	labels?: Record<string, string>;
	status: IncidentStatus;
	created_at: string;  // ISO 8601
	updated_at: string;  // ISO 8601
	resolved_at?: string; // ISO 8601
	/** Complete immutable event log (timeline). */
	events: IncidentEvent[];
}

/** Lightweight summary used in list responses (no event log). */
export interface IncidentListItem {
	id: string;
	title: string;
	severity: string;
	alert_fingerprint?: string;
	alertmanager_instance?: string;
	labels?: Record<string, string>;
	status: IncidentStatus;
	created_at: string;
	updated_at: string;
	resolved_at?: string;
	event_count: number;
}

/** Paginated list response from GET /api/incidents. */
export interface ListIncidentsResponse {
	incidents: IncidentListItem[];
	total: number;
	limit: number;
	offset: number;
}

/** Timeline-only response from GET /api/incidents/{id}/timeline. */
export interface IncidentTimelineResponse {
	incident_id: string;
	status: IncidentStatus;
	events: IncidentEvent[];
}

/** Payload for POST /api/incidents. */
export interface CreateIncidentRequest {
	title: string;
	severity: string;
	alert_fingerprint?: string;
	alertmanager_instance?: string;
	labels?: Record<string, string>;
	initial_message?: string;
	created_by: string;
}

/** Payload for POST /api/incidents/{id}/events. */
export interface AddEventRequest {
	kind: Exclude<IncidentEventKind, 'CREATED'>;
	actor: string;
	message?: string;
}

export interface CreateSilenceRequest {
	alertmanager: string;
	matchers: Matcher[];
	starts_at: string;
	ends_at: string;
	created_by: string;
	comment: string;
	ack_type?: 'visual';
	ack_by?: string;
	ack_comment?: string;
}

export type UserRole = 'viewer' | 'silencer' | 'config-editor' | 'admin' | '';

export interface AuthStatus {
	admin_enabled: boolean;
	authenticated: boolean;
	/** Role granted to the current token. Empty string when not authenticated. */
	role: UserRole;
}

export interface ValidationResult {
	valid: boolean;
	warnings: string[];
	errors?: string[];
}

export interface ConfigResponse {
	alertmanager: string;
	raw_yaml: string;
}

// ─── Config Builder types ─────────────────────────────────────────────────────

/** Mirrors configbuilder.RouteSpec in the Go backend. */
export interface RouteSpec {
	receiver?: string;
	group_by?: string[];
	matchers?: string[];
	continue?: boolean;
	group_wait?: string;
	group_interval?: string;
	repeat_interval?: string;
	mute_time_intervals?: string[];
	active_time_intervals?: string[];
	routes: RouteSpec[];
}

export interface WebhookConfigDef {
	url: string;
	send_resolved?: boolean;
	max_alerts?: number;
}

export interface SlackConfigDef {
	channel: string;
	api_url?: string;
	username?: string;
	text?: string;
	title?: string;
	send_resolved?: boolean;
}

export interface EmailConfigDef {
	to: string;
	from?: string;
	smarthost?: string;
	auth_username?: string;
	auth_password?: string;
	send_resolved?: boolean;
}

export interface PagerdutyConfigDef {
	routing_key?: string;
	service_key?: string;
	description?: string;
	send_resolved?: boolean;
}

export interface OpsgenieConfigDef {
	api_key?: string;
	message?: string;
	priority?: string;
	send_resolved?: boolean;
}

export interface ReceiverDef {
	name: string;
	webhook_configs?: WebhookConfigDef[];
	slack_configs?: SlackConfigDef[];
	email_configs?: EmailConfigDef[];
	pagerduty_configs?: PagerdutyConfigDef[];
	opsgenie_configs?: OpsgenieConfigDef[];
	raw_yaml?: string;
}

export interface TimeRangeDef {
	start_time: string;
	end_time: string;
}

export interface TimeIntervalDef {
	times?: TimeRangeDef[];
	weekdays?: string[];
	days_of_month?: string[];
	months?: string[];
	years?: string[];
	location?: string;
}

export interface TimeIntervalEntry {
	name: string;
	time_intervals: TimeIntervalDef[];
}

export interface BuilderReceiverRouteRef {
	matchers: string[];
	depth: number;
}

export interface BuilderReceiverRoutesResponse {
	receiver: string;
	referenced_by: BuilderReceiverRouteRef[];
}

export interface SaveConfigRequest {
	alertmanager: string;
	raw_yaml: string;
	save_mode: 'disk' | 'github' | 'gitlab';
	disk_options?: { file_path: string };
	git_options?: {
		repo: string;
		branch: string;
		file_path: string;
		commit_message?: string;
		author_name?: string;
		author_email?: string;
	};
	webhook_url?: string;
}
