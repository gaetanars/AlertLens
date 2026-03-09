package alertmanager

import "time"

// ─── Alertmanager API v2 types ───────────────────────────────────────────────

// Alert represents a single alert as returned by GET /api/v2/alerts.
type Alert struct {
	Fingerprint  string            `json:"fingerprint"`
	Receivers    []Receiver        `json:"receivers"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Status       AlertStatus       `json:"status"`
}

type AlertStatus struct {
	State       string   `json:"state"` // "active" | "suppressed" | "unprocessed"
	SilencedBy  []string `json:"silencedBy"`
	InhibitedBy []string `json:"inhibitedBy"`
}

// EnrichedAlert is an Alert enriched with its Alertmanager instance name and ack info.
// InstanceID is an alias for Alertmanager for API clarity.
type EnrichedAlert struct {
	Alert
	// Alertmanager is the name of the instance this alert was fetched from.
	Alertmanager string `json:"alertmanager"`
	// InstanceID mirrors Alertmanager for forward-compatibility with multi-instance API.
	InstanceID string `json:"instance_id,omitempty"`
	Ack        *Ack   `json:"ack,omitempty"`
}

// Ack holds visual-ack metadata reconstructed from silence labels.
type Ack struct {
	Active    bool   `json:"active"`
	By        string `json:"by"`
	Comment   string `json:"comment"`
	SilenceID string `json:"silence_id"`
}

// Silence represents a silence as returned by GET /api/v2/silences.
type Silence struct {
	ID        string    `json:"id"`
	Matchers  []Matcher `json:"matchers"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	CreatedBy string    `json:"createdBy"`
	Comment   string    `json:"comment"`
	Status    SilenceStatus `json:"status"`
}

type SilenceStatus struct {
	State string `json:"state"` // "active" | "pending" | "expired"
}

// Matcher represents a label matcher used in silences and routes.
type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
	IsEqual bool   `json:"isEqual"`
}

// SilenceInput is the payload for POST/PUT /api/v2/silences.
type SilenceInput struct {
	ID        string    `json:"id,omitempty"`
	Matchers  []Matcher `json:"matchers"`
	StartsAt  time.Time `json:"startsAt"`
	EndsAt    time.Time `json:"endsAt"`
	CreatedBy string    `json:"createdBy"`
	Comment   string    `json:"comment"`
}

// Receiver represents an Alertmanager receiver reference.
type Receiver struct {
	Name string `json:"name"`
}

// AMStatus is the response from GET /api/v2/status.
type AMStatus struct {
	Cluster   ClusterStatus `json:"cluster"`
	Config    AMConfig      `json:"config"`
	VersionInfo VersionInfo `json:"versionInfo"`
}

type ClusterStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "ready" | "settling" | "disabled"
	Peers  []Peer `json:"peers"`
}

type Peer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type AMConfig struct {
	Original string `json:"original"`
}

type VersionInfo struct {
	Branch    string `json:"branch"`
	BuildDate string `json:"buildDate"`
	BuildUser string `json:"buildUser"`
	GoVersion string `json:"goVersion"`
	Revision  string `json:"revision"`
	Version   string `json:"version"`
}

// ─── AlertLens API response types ───────────────────────────────────────────

// InstanceStatus is returned by GET /api/alertmanagers.
type InstanceStatus struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Healthy   bool   `json:"healthy"`
	Version   string `json:"version"`
	HasTenant bool   `json:"has_tenant"`
	Error     string `json:"error,omitempty"`
}

// AlertsQueryParams contains query parameters for fetching alerts.
type AlertsQueryParams struct {
	Filter    []string // Alertmanager matcher strings
	Instance  string   // filter to a single instance name
	Silenced  bool
	Inhibited bool
	Active    bool
}

// AlertsViewParams extends AlertsQueryParams with view-layer options that are
// applied by the API handler (not forwarded to Alertmanager).
type AlertsViewParams struct {
	AlertsQueryParams
	// GroupBy holds label keys to group alerts by in the response.
	// e.g. ["severity"], ["status"], ["alertmanager"], ["team","env"]
	GroupBy []string
	// Severity filters results to alerts whose "severity" label matches any of
	// the provided values. Empty = no filter.
	Severity []string
	// Status filters by alert state: "active" | "suppressed" | "unprocessed"
	// Empty = no filter.
	Status []string
	// Limit caps the number of alerts returned (default 500, max 5000).
	Limit int
	// Offset skips the first N alerts (for pagination).
	Offset int
}

// AlertGroup represents a set of alerts sharing common label values.
type AlertGroup struct {
	// Labels are the grouping key-value pairs.
	Labels map[string]string `json:"labels"`
	// Alerts in this group.
	Alerts []EnrichedAlert `json:"alerts"`
	// Count is len(Alerts) for convenience.
	Count int `json:"count"`
}

// AlertsResponse is the envelope returned by GET /api/v2/alerts.
type AlertsResponse struct {
	// Groups contains alerts arranged by the requested GroupBy keys.
	// When GroupBy is empty there is a single group with empty Labels.
	Groups []AlertGroup `json:"groups"`
	// Total is the total number of alerts before pagination (after filtering).
	Total int `json:"total"`
	// Limit and Offset echo back the pagination params used.
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	// PartialFailures lists per-instance errors when one or more instances
	// failed to respond. An empty slice means all instances succeeded.
	// Non-empty but with data present = degraded mode (partial results).
	PartialFailures []InstanceError `json:"partial_failures,omitempty"`
}

// SilenceQueryParams contains query parameters for fetching silences.
type SilenceQueryParams struct {
	Instance string
	Type     string // "silence" | "ack" | "" (all)
}
