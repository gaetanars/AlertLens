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
	alertmanager: string;
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

/** Top-level response from GET /api/alerts with grouping & pagination. */
export interface AlertsResponse {
	groups: AlertGroup[];
	/** Total alerts before pagination. */
	total: number;
	limit: number;
	offset: number;
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
