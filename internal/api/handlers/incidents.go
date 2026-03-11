package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/alertlens/alertlens/internal/incident"
)

// IncidentsHandler exposes the incident lifecycle API.
// Routes:
//
//	GET  /api/incidents                  → list (paginated, filterable)
//	POST /api/incidents                  → create
//	GET  /api/incidents/{id}             → get full incident (with timeline)
//	GET  /api/incidents/{id}/timeline    → event log only
//	POST /api/incidents/{id}/events      → add lifecycle event
type IncidentsHandler struct {
	store *incident.Store
}

// NewIncidentsHandler creates an IncidentsHandler backed by the given Store.
func NewIncidentsHandler(store *incident.Store) *IncidentsHandler {
	return &IncidentsHandler{store: store}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List handles GET /api/incidents.
//
// Query params:
//   - status   : filter by incident status (OPEN|ACK|INVESTIGATING|RESOLVED)
//   - alert_fingerprint : filter to incidents linked to a specific alert
//   - limit    : page size (default 100, max 500)
//   - offset   : pagination offset (default 0)
func (h *IncidentsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	status := incident.Status(q.Get("status"))
	fingerprint := q.Get("alert_fingerprint")

	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 || limit > 500 {
		limit = 100
	}
	offset, err := strconv.Atoi(q.Get("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	resp := h.store.List(incident.ListFilter{
		Status:           status,
		AlertFingerprint: fingerprint,
		Limit:            limit,
		Offset:           offset,
	})
	writeJSON(w, resp)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create handles POST /api/incidents.
func (h *IncidentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req incident.CreateIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	inc, err := h.store.Create(req)
	if err != nil {
		writeError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, inc)
}

// ─── Get ──────────────────────────────────────────────────────────────────────

// Get handles GET /api/incidents/{id}.
// Returns the full Incident including the event log (timeline).
func (h *IncidentsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inc, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, incident.ErrNotFound) {
			writeError(w, "incident not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, inc)
}

// ─── Timeline ────────────────────────────────────────────────────────────────

// Timeline handles GET /api/incidents/{id}/timeline.
// Returns only the events array — useful for polling updates without re-fetching
// the entire incident payload.
func (h *IncidentsHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inc, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, incident.ErrNotFound) {
			writeError(w, "incident not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type timelineResponse struct {
		IncidentID string                   `json:"incident_id"`
		Status     incident.Status          `json:"status"`
		Events     []incident.IncidentEvent `json:"events"`
	}
	writeJSON(w, timelineResponse{
		IncidentID: inc.ID,
		Status:     inc.Status,
		Events:     inc.Events,
	})
}

// ─── AddEvent ────────────────────────────────────────────────────────────────

// AddEvent handles POST /api/incidents/{id}/events.
// Accepts JSON body: { "kind": "ACK|INVESTIGATING|RESOLVED|REOPENED|COMMENT",
//
//	"actor": "...", "message": "..." }
//
// Returns the updated full Incident on success.
func (h *IncidentsHandler) AddEvent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req incident.AddEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	inc, err := h.store.AddEvent(id, req)
	if err != nil {
		switch {
		case errors.Is(err, incident.ErrNotFound):
			writeError(w, "incident not found", http.StatusNotFound)
		case errors.Is(err, incident.ErrInvalidTransition):
			writeError(w, err.Error(), http.StatusConflict)
		case errors.Is(err, incident.ErrInvalidEventKind):
			writeError(w, err.Error(), http.StatusBadRequest)
		default:
			writeError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, inc)
}
