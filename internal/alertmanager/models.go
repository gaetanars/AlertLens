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
type EnrichedAlert struct {
	Alert
	Alertmanager string `json:"alertmanager"`
	Ack          *Ack   `json:"ack,omitempty"`
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
	Filter   []string // Alertmanager matcher strings
	Instance string   // filter to a single instance name
	Silenced bool
	Inhibited bool
	Active   bool
}

// SilenceQueryParams contains query parameters for fetching silences.
type SilenceQueryParams struct {
	Instance string
	Type     string // "silence" | "ack" | "" (all)
}
