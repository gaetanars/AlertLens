package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alertlens/alertlens/internal/alertmanager"
)

// BulkHandler handles bulk operations on alerts (POST /api/v1/bulk).
type BulkHandler struct {
	pool *alertmanager.Pool
}

// NewBulkHandler creates a BulkHandler.
func NewBulkHandler(pool *alertmanager.Pool) *BulkHandler {
	return &BulkHandler{pool: pool}
}

// BulkAlertRef is a minimal alert reference sent by the client.
// Only the labels and the source instance are required to compute matchers.
type BulkAlertRef struct {
	Fingerprint  string            `json:"fingerprint"`
	Alertmanager string            `json:"alertmanager"`
	Labels       map[string]string `json:"labels"`
}

// BulkActionRequest is the payload for POST /api/v1/bulk.
type BulkActionRequest struct {
	// Action is "silence" (Alertmanager silence) or "ack" (visual ack).
	Action string `json:"action"`
	// Alerts is the list of alert refs to silence / ack.
	Alerts []BulkAlertRef `json:"alerts"`
	// EndsAt is the silence expiry time.  Defaults to now+1h when omitted or zero.
	EndsAt *time.Time `json:"ends_at,omitempty"`
	// CreatedBy is the username recorded on the silence.  Defaults to "alertlens".
	CreatedBy string `json:"created_by"`
	// Comment is the free-text reason for the silence.
	Comment string `json:"comment"`
}

// BulkActionResponse is the result of POST /api/v1/bulk.
type BulkActionResponse struct {
	// SilenceIDs holds every Alertmanager silence ID that was created.
	SilenceIDs []string `json:"silence_ids"`
	// Strategy reports which merge path was taken:
	//   "merged"     — one silence per alertmanager instance (common matchers found)
	//   "individual" — one silence per alert (no common matchers in at least one group)
	Strategy string `json:"strategy"`
	// Count equals len(SilenceIDs).
	Count int `json:"count"`
}

// Create handles POST /api/v1/bulk.
//
// Smart Merge logic (ADR-007):
//  1. Alerts are grouped by their alertmanager instance.
//  2. For each group, the label-value intersection is computed across every alert.
//  3. If a non-empty intersection exists → one merged silence per group.
//  4. Otherwise → one individual silence per alert in that group.
func (h *BulkHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req BulkActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.Alerts) == 0 {
		writeError(w, "alerts list is empty", http.StatusBadRequest)
		return
	}
	if req.Action != "silence" && req.Action != "ack" {
		writeError(w, `action must be "silence" or "ack"`, http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	startsAt := now

	var endsAt time.Time
	if req.EndsAt != nil && !req.EndsAt.IsZero() {
		endsAt = req.EndsAt.UTC()
	} else {
		endsAt = now.Add(time.Hour)
	}
	if !endsAt.After(startsAt) {
		writeError(w, "ends_at must be in the future", http.StatusBadRequest)
		return
	}

	createdBy := req.CreatedBy
	if createdBy == "" {
		createdBy = "alertlens"
	}
	comment := req.Comment
	if comment == "" {
		comment = "Bulk silenced by AlertLens"
	}

	// ── Group alerts by alertmanager instance ────────────────────────────────
	groups := groupAlertsByInstance(req.Alerts)

	allSilenceIDs := make([]string, 0, len(req.Alerts))
	mergedAny := false

	for instance, instanceAlerts := range groups {
		client := h.pool.Client(instance)
		if client == nil {
			writeError(w, "unknown alertmanager instance: "+instance, http.StatusBadRequest)
			return
		}

		// ── Smart Merge: find intersection of common matchers ────────────────
		commonMatchers := smartMerge(instanceAlerts)

		if len(commonMatchers) > 0 {
			// Merged path: one silence covers the entire instance group.
			mergedAny = true
			id, err := h.createOne(r, client, req.Action, alertmanager.SilenceInput{
				Matchers:  commonMatchers,
				StartsAt:  startsAt,
				EndsAt:    endsAt,
				CreatedBy: createdBy,
				Comment:   comment,
			}, createdBy, comment)
			if err != nil {
				writeError(w, err.Error(), http.StatusBadGateway)
				return
			}
			allSilenceIDs = append(allSilenceIDs, id)
		} else {
			// Individual fallback: one silence per alert.
			for _, alert := range instanceAlerts {
				id, err := h.createOne(r, client, req.Action, alertmanager.SilenceInput{
					Matchers:  labelsToMatchers(alert.Labels),
					StartsAt:  startsAt,
					EndsAt:    endsAt,
					CreatedBy: createdBy,
					Comment:   comment,
				}, createdBy, comment)
				if err != nil {
					writeError(w, err.Error(), http.StatusBadGateway)
					return
				}
				allSilenceIDs = append(allSilenceIDs, id)
			}
		}
	}

	strategy := "individual"
	if mergedAny {
		strategy = "merged"
	}

	writeJSONStatus(w, http.StatusCreated, BulkActionResponse{
		SilenceIDs: allSilenceIDs,
		Strategy:   strategy,
		Count:      len(allSilenceIDs),
	})
}

// createOne dispatches to client.CreateSilence or client.CreateAck based on action.
func (h *BulkHandler) createOne(
	r *http.Request,
	client *alertmanager.Client,
	action string,
	input alertmanager.SilenceInput,
	ackBy, ackComment string,
) (string, error) {
	if action == "ack" {
		return client.CreateAck(r.Context(), ackBy, ackComment, input)
	}
	return client.CreateSilence(r.Context(), input)
}

// ── Smart Merge helpers ──────────────────────────────────────────────────────

// groupAlertsByInstance partitions alert refs by their alertmanager field.
func groupAlertsByInstance(alerts []BulkAlertRef) map[string][]BulkAlertRef {
	groups := make(map[string][]BulkAlertRef, 4)
	for _, a := range alerts {
		groups[a.Alertmanager] = append(groups[a.Alertmanager], a)
	}
	return groups
}

// smartMerge returns equality matchers for labels whose key and value are
// identical across every alert in the slice.
//
// Internal/meta labels (alertlens_* and __name__) are stripped.
// Returns nil when the intersection is empty.
func smartMerge(alerts []BulkAlertRef) []alertmanager.Matcher {
	if len(alerts) == 0 {
		return nil
	}
	if len(alerts) == 1 {
		return labelsToMatchers(filterMetaLabels(alerts[0].Labels))
	}

	// Seed with the first alert's labels (minus meta-labels).
	common := filterMetaLabels(alerts[0].Labels)

	// Intersect with every subsequent alert.
	for _, a := range alerts[1:] {
		for k, v := range common {
			if av, ok := a.Labels[k]; !ok || av != v {
				delete(common, k)
			}
		}
		if len(common) == 0 {
			return nil
		}
	}

	return labelsToMatchers(common)
}

// filterMetaLabels returns a copy of labels with internal keys removed.
func filterMetaLabels(labels map[string]string) map[string]string {
	out := make(map[string]string, len(labels))
	for k, v := range labels {
		if k == "__name__" || len(k) > 10 && k[:10] == "alertlens_" {
			continue
		}
		out[k] = v
	}
	return out
}

// labelsToMatchers converts a label map to equality Matchers.
// The resulting slice is sorted by label name for deterministic output.
func labelsToMatchers(labels map[string]string) []alertmanager.Matcher {
	matchers := make([]alertmanager.Matcher, 0, len(labels))
	for k, v := range labels {
		matchers = append(matchers, alertmanager.Matcher{
			Name:    k,
			Value:   v,
			IsRegex: false,
			IsEqual: true,
		})
	}
	return matchers
}
