// Package incident provides the immutable-ledger incident lifecycle engine for
// AlertLens. Each state transition is recorded as an append-only IncidentEvent;
// the current state is always derived from the latest event. No event is ever
// deleted or mutated — this gives a complete audit trail.
package incident

import (
	"time"
)

// ─── State machine ────────────────────────────────────────────────────────────

// Status represents the current lifecycle state of an incident.
// Valid transitions:
//
//	OPEN → ACK | INVESTIGATING | RESOLVED
//	ACK  → INVESTIGATING | RESOLVED | OPEN (reopen)
//	INVESTIGATING → RESOLVED | OPEN (reopen)
//	RESOLVED → OPEN (reopen)
type Status string

const (
	StatusOpen          Status = "OPEN"
	StatusAck           Status = "ACK"
	StatusInvestigating Status = "INVESTIGATING"
	StatusResolved      Status = "RESOLVED"
)

// validTransitions maps each status to the set of statuses it can transition to.
var validTransitions = map[Status]map[Status]bool{
	StatusOpen:          {StatusAck: true, StatusInvestigating: true, StatusResolved: true},
	StatusAck:           {StatusInvestigating: true, StatusResolved: true, StatusOpen: true},
	StatusInvestigating: {StatusResolved: true, StatusOpen: true},
	StatusResolved:      {StatusOpen: true},
}

// CanTransition reports whether a transition from `from` to `to` is permitted
// by the state machine.
func CanTransition(from, to Status) bool {
	return validTransitions[from][to]
}

// ─── Event kinds ──────────────────────────────────────────────────────────────

// EventKind classifies each ledger entry.
type EventKind string

const (
	EventKindCreated      EventKind = "CREATED"
	EventKindAck          EventKind = "ACK"
	EventKindInvestigating EventKind = "INVESTIGATING"
	EventKindResolved     EventKind = "RESOLVED"
	EventKindReopened     EventKind = "REOPENED"
	EventKindComment      EventKind = "COMMENT" // annotation without state change
)

// ─── Core types ───────────────────────────────────────────────────────────────

// IncidentEvent is a single immutable entry in the incident ledger.
// Events are append-only; they are never modified after creation.
type IncidentEvent struct {
	// Seq is the 1-based sequence number within the incident (monotonically increasing).
	Seq int `json:"seq"`
	// Kind classifies the event.
	Kind EventKind `json:"kind"`
	// Status is the incident state after this event (empty for COMMENT events).
	Status Status `json:"status,omitempty"`
	// Actor is the user or system that created the event.
	Actor string `json:"actor"`
	// Message is an optional human-readable note attached to this event.
	Message string `json:"message,omitempty"`
	// OccurredAt is the wall-clock time of the event.
	OccurredAt time.Time `json:"occurred_at"`
}

// Incident represents an operational incident tracked in AlertLens.
// The Status field always reflects the most recent IncidentEvent.
type Incident struct {
	// ID is a UUID v4 (formatted as compact hex without dashes for brevity).
	ID string `json:"id"`
	// Title is a short human-readable description.
	Title string `json:"title"`
	// Severity mirrors the alert severity that triggered the incident.
	Severity string `json:"severity"`
	// AlertFingerprint optionally links this incident to an Alertmanager alert.
	AlertFingerprint string `json:"alert_fingerprint,omitempty"`
	// AlertmanagerInstance is the source AM instance name (when alert is linked).
	AlertmanagerInstance string `json:"alertmanager_instance,omitempty"`
	// Labels carry arbitrary key-value context (copied from alert labels or set manually).
	Labels map[string]string `json:"labels,omitempty"`
	// Status is the current lifecycle status (derived from latest event).
	Status Status `json:"status"`
	// CreatedAt is when the incident was opened.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt reflects the timestamp of the most recent event.
	UpdatedAt time.Time `json:"updated_at"`
	// ResolvedAt is set when/if the incident reaches RESOLVED.
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	// Events is the complete immutable event log (timeline).
	Events []IncidentEvent `json:"events"`
}

// CurrentStatus returns the status of the incident as derived from the last
// event that carries a status. Falls back to StatusOpen if the log is empty.
func (inc *Incident) CurrentStatus() Status {
	for i := len(inc.Events) - 1; i >= 0; i-- {
		if inc.Events[i].Status != "" {
			return inc.Events[i].Status
		}
	}
	return StatusOpen
}

// ─── Request / response payloads ─────────────────────────────────────────────

// CreateIncidentRequest is the payload for POST /api/incidents.
type CreateIncidentRequest struct {
	Title                string            `json:"title"`
	Severity             string            `json:"severity"`
	AlertFingerprint     string            `json:"alert_fingerprint,omitempty"`
	AlertmanagerInstance string            `json:"alertmanager_instance,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	// InitialMessage is the first annotation attached to the CREATED event.
	InitialMessage string `json:"initial_message,omitempty"`
	// CreatedBy is the actor that opened the incident.
	CreatedBy string `json:"created_by"`
}

// AddEventRequest is the payload for POST /api/incidents/{id}/events.
type AddEventRequest struct {
	// Kind must be one of ACK, INVESTIGATING, RESOLVED, REOPENED, or COMMENT.
	Kind    EventKind `json:"kind"`
	Actor   string    `json:"actor"`
	Message string    `json:"message,omitempty"`
}

// IncidentListItem is a lightweight incident summary for list responses
// (excludes the full event log to keep payloads small).
type IncidentListItem struct {
	ID                   string            `json:"id"`
	Title                string            `json:"title"`
	Severity             string            `json:"severity"`
	AlertFingerprint     string            `json:"alert_fingerprint,omitempty"`
	AlertmanagerInstance string            `json:"alertmanager_instance,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Status               Status            `json:"status"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	ResolvedAt           *time.Time        `json:"resolved_at,omitempty"`
	// EventCount is the total number of events in the incident ledger.
	EventCount int `json:"event_count"`
}

// ListIncidentsResponse wraps a paginated incident list.
type ListIncidentsResponse struct {
	Incidents []IncidentListItem `json:"incidents"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}
