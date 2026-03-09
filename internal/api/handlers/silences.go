package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/alertlens/alertlens/internal/alertmanager"
)

// SilencesHandler handles silence-related requests.
type SilencesHandler struct {
	pool *alertmanager.Pool
}

// NewSilencesHandler creates a SilencesHandler.
func NewSilencesHandler(pool *alertmanager.Pool) *SilencesHandler {
	return &SilencesHandler{pool: pool}
}

// List handles GET /api/silences.
func (h *SilencesHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := alertmanager.SilenceQueryParams{
		Instance: q.Get("instance"),
		Type:     q.Get("type"),
	}
	silences, errs := h.pool.GetAggregatedSilences(r.Context(), params)
	// COR-02/ERR-04: if every instance failed and no silences were returned,
	// return a gateway error; otherwise serve partial results with metadata.
	if len(silences) == 0 && len(errs) > 0 {
		writeError(w, "all alertmanager instances failed to respond", http.StatusBadGateway)
		return
	}
	if silences == nil {
		silences = []alertmanager.EnrichedSilence{}
	}
	writeJSON(w, silences)
}

// Get handles GET /api/silences/{id}.
func (h *SilencesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	instance := r.URL.Query().Get("instance")

	client := resolveClient(h.pool, w, instance)
	if client == nil {
		return
	}
	silence, err := client.GetSilence(r.Context(), id)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, silence)
}

// silenceRequest is the payload for POST /api/silences.
type silenceRequest struct {
	Alertmanager string                    `json:"alertmanager"`
	Matchers     []alertmanager.Matcher    `json:"matchers"`
	StartsAt     time.Time                 `json:"starts_at"`
	EndsAt       time.Time                 `json:"ends_at"`
	CreatedBy    string                    `json:"created_by"`
	Comment      string                    `json:"comment"`
	AckType      string                    `json:"ack_type"`     // "visual" for visual ack
	AckBy        string                    `json:"ack_by"`
	AckComment   string                    `json:"ack_comment"`
}

// Create handles POST /api/silences.
func (h *SilencesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req silenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	client := resolveClient(h.pool, w, req.Alertmanager)
	if client == nil {
		return
	}

	input := alertmanager.SilenceInput{
		Matchers:  req.Matchers,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		CreatedBy: req.CreatedBy,
		Comment:   req.Comment,
	}

	var silenceID string
	var err error

	if req.AckType == "visual" {
		silenceID, err = client.CreateAck(r.Context(), req.AckBy, req.AckComment, input)
	} else {
		silenceID, err = client.CreateSilence(r.Context(), input)
	}
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}

	writeJSONStatus(w, http.StatusCreated, map[string]string{"silence_id": silenceID})
}

// Update handles PUT /api/silences/{id}.
func (h *SilencesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req silenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	client := resolveClient(h.pool, w, req.Alertmanager)
	if client == nil {
		return
	}

	input := alertmanager.SilenceInput{
		Matchers:  req.Matchers,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		CreatedBy: req.CreatedBy,
		Comment:   req.Comment,
	}

	silenceID, err := client.UpdateSilence(r.Context(), id, input)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]string{"silence_id": silenceID})
}

// Delete handles DELETE /api/silences/{id}.
func (h *SilencesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	instance := r.URL.Query().Get("instance")

	client := resolveClient(h.pool, w, instance)
	if client == nil {
		return
	}

	if err := client.ExpireSilence(r.Context(), id); err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

