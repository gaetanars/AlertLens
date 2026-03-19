// Package incident implements an in-memory immutable ledger for incident
// tracking, as described in ADR-008.
package incident

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ─── Sentinel errors ──────────────────────────────────────────────────────────

var (
	// ErrNotFound is returned when the requested incident does not exist.
	ErrNotFound = errors.New("incident not found")
	// ErrInvalidTransition is returned when the requested state change violates
	// the state machine rules.
	ErrInvalidTransition = errors.New("invalid state transition")
	// ErrInvalidEventKind is returned when the event kind is unknown.
	ErrInvalidEventKind = errors.New("invalid event kind")
)

// ─── Store ────────────────────────────────────────────────────────────────────

// Store is the in-memory incident ledger. All mutations are protected by an
// RWMutex so the store is safe for concurrent use. On process restart all data
// is lost — persistence is explicitly out of scope for Phase 1 (see ADR-003).
type Store struct {
	mu        sync.RWMutex
	incidents map[string]*Incident
	// order preserves insertion order for stable list responses.
	order []string
}

// NewStore creates an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{
		incidents: make(map[string]*Incident),
	}
}

// ─── CRUD ─────────────────────────────────────────────────────────────────────

// Create opens a new incident and appends the first CREATED event to the ledger.
// Returns the fully-populated Incident.
func (s *Store) Create(req CreateIncidentRequest) (*Incident, error) {
	if req.Title == "" {
		return nil, errors.New("incident title is required")
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "system"
	}

	id, err := newID()
	if err != nil {
		return nil, fmt.Errorf("generate incident id: %w", err)
	}

	now := time.Now().UTC()
	inc := &Incident{
		ID:                   id,
		Title:                req.Title,
		Severity:             req.Severity,
		AlertFingerprint:     req.AlertFingerprint,
		AlertmanagerInstance: req.AlertmanagerInstance,
		Labels:               req.Labels,
		Status:               StatusOpen,
		CreatedAt:            now,
		UpdatedAt:            now,
		Events: []IncidentEvent{
			{
				Seq:        1,
				Kind:       EventKindCreated,
				Status:     StatusOpen,
				Actor:      req.CreatedBy,
				Message:    req.InitialMessage,
				OccurredAt: now,
			},
		},
	}

	s.mu.Lock()
	s.incidents[id] = inc
	s.order = append(s.order, id)
	s.mu.Unlock()

	return inc, nil
}

// Get returns the incident with the given ID, or ErrNotFound.
func (s *Store) Get(id string) (*Incident, error) {
	s.mu.RLock()
	inc, ok := s.incidents[id]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}
	// Return a deep copy to avoid races on the caller side.
	return copyIncident(inc), nil
}

// List returns a paginated snapshot of all incidents, newest-first by UpdatedAt.
// filter.Status filters by status when non-empty. filter.AlertFingerprint
// restricts to incidents linked to a specific alert.
func (s *Store) List(f ListFilter) ListIncidentsResponse {
	s.mu.RLock()
	all := make([]*Incident, 0, len(s.order))
	for _, id := range s.order {
		inc := s.incidents[id]
		if f.matches(inc) {
			all = append(all, inc)
		}
	}
	s.mu.RUnlock()

	// Sort newest UpdatedAt first.
	sort.Slice(all, func(i, j int) bool {
		return all[i].UpdatedAt.After(all[j].UpdatedAt)
	})

	total := len(all)
	if f.Offset > total {
		f.Offset = total
	}
	all = all[f.Offset:]
	if f.Limit > 0 && len(all) > f.Limit {
		all = all[:f.Limit]
	}

	items := make([]IncidentListItem, len(all))
	for i, inc := range all {
		items[i] = toListItem(inc)
	}

	limit := f.Limit
	if limit == 0 {
		limit = 100
	}
	return ListIncidentsResponse{
		Incidents: items,
		Total:     total,
		Limit:     limit,
		Offset:    f.Offset,
	}
}

// AddEvent appends a new lifecycle event to the incident ledger and updates the
// incident's Status / UpdatedAt / ResolvedAt accordingly.
// Returns the updated Incident or an error if the transition is invalid.
func (s *Store) AddEvent(incidentID string, req AddEventRequest) (*Incident, error) {
	if err := validateEventKind(req.Kind); err != nil {
		return nil, err
	}
	if req.Actor == "" {
		req.Actor = "system"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	inc, ok := s.incidents[incidentID]
	if !ok {
		return nil, ErrNotFound
	}

	// Derive the new status from the event kind.
	newStatus, err := statusFromKind(inc.Status, req.Kind)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	evt := IncidentEvent{
		Seq:        len(inc.Events) + 1,
		Kind:       req.Kind,
		Status:     newStatus,
		Actor:      req.Actor,
		Message:    req.Message,
		OccurredAt: now,
	}

	// COMMENT events do not change status.
	if req.Kind == EventKindComment {
		evt.Status = ""
	}

	inc.Events = append(inc.Events, evt)
	inc.UpdatedAt = now

	if newStatus != "" {
		inc.Status = newStatus
	}
	if newStatus == StatusResolved {
		t := now
		inc.ResolvedAt = &t
	} else if req.Kind == EventKindReopened {
		inc.ResolvedAt = nil
		inc.Status = StatusOpen
	}

	return copyIncident(inc), nil
}

// ─── List filters ─────────────────────────────────────────────────────────────

// ListFilter contains optional filter criteria for List.
type ListFilter struct {
	// Status filters incidents whose current status matches. Empty = all.
	Status Status
	// AlertFingerprint restricts to incidents linked to a specific alert.
	AlertFingerprint string
	// Limit caps the number of results (0 = 100).
	Limit int
	// Offset skips the first N results.
	Offset int
}

func (f ListFilter) matches(inc *Incident) bool {
	if f.Status != "" && inc.Status != f.Status {
		return false
	}
	if f.AlertFingerprint != "" && inc.AlertFingerprint != f.AlertFingerprint {
		return false
	}
	return true
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// statusFromKind resolves the next Status given the current status and the
// event kind, enforcing the state machine.
func statusFromKind(current Status, kind EventKind) (Status, error) {
	switch kind {
	case EventKindComment:
		return "", nil // no state change
	case EventKindAck:
		if !CanTransition(current, StatusAck) {
			return "", fmt.Errorf("%w: %s → ACK", ErrInvalidTransition, current)
		}
		return StatusAck, nil
	case EventKindInvestigating:
		if !CanTransition(current, StatusInvestigating) {
			return "", fmt.Errorf("%w: %s → INVESTIGATING", ErrInvalidTransition, current)
		}
		return StatusInvestigating, nil
	case EventKindResolved:
		if !CanTransition(current, StatusResolved) {
			return "", fmt.Errorf("%w: %s → RESOLVED", ErrInvalidTransition, current)
		}
		return StatusResolved, nil
	case EventKindReopened:
		if !CanTransition(current, StatusOpen) {
			return "", fmt.Errorf("%w: %s → OPEN", ErrInvalidTransition, current)
		}
		return StatusOpen, nil
	default:
		return "", ErrInvalidEventKind
	}
}

// validateEventKind returns an error if kind is not one of the user-submittable
// event kinds. EventKindCreated is excluded — it's set internally by Create.
func validateEventKind(kind EventKind) error {
	switch kind {
	case EventKindAck, EventKindInvestigating, EventKindResolved, EventKindReopened, EventKindComment:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidEventKind, kind)
	}
}

// newID generates a cryptographically random 16-byte hex string.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// copyIncident performs a shallow-but-safe copy so external callers can't
// mutate stored data through the returned pointer.
func copyIncident(src *Incident) *Incident {
	cp := *src

	// Deep-copy slices and maps.
	if src.Labels != nil {
		cp.Labels = make(map[string]string, len(src.Labels))
		for k, v := range src.Labels {
			cp.Labels[k] = v
		}
	}
	if src.Events != nil {
		cp.Events = make([]IncidentEvent, len(src.Events))
		copy(cp.Events, src.Events)
	}
	if src.ResolvedAt != nil {
		t := *src.ResolvedAt
		cp.ResolvedAt = &t
	}
	return &cp
}

// toListItem converts a full Incident to the lightweight list payload.
func toListItem(inc *Incident) IncidentListItem {
	item := IncidentListItem{
		ID:                   inc.ID,
		Title:                inc.Title,
		Severity:             inc.Severity,
		AlertFingerprint:     inc.AlertFingerprint,
		AlertmanagerInstance: inc.AlertmanagerInstance,
		Labels:               inc.Labels,
		Status:               inc.Status,
		CreatedAt:            inc.CreatedAt,
		UpdatedAt:            inc.UpdatedAt,
		ResolvedAt:           inc.ResolvedAt,
		EventCount:           len(inc.Events),
	}
	return item
}
