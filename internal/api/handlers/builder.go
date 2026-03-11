package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"github.com/alertlens/alertlens/internal/configbuilder"
	"github.com/go-chi/chi/v5"
)

// BuilderHandler handles Config Builder CRUD requests for time intervals,
// receivers, and routing.  All operations fetch the live config from
// Alertmanager, apply the requested mutation, validate the result with the
// official Alertmanager library, and return the new YAML.
//
// The caller should then POST /api/config/save if they want to persist the
// returned raw_yaml back to Alertmanager.
type BuilderHandler struct {
	pool *alertmanager.Pool
}

// NewBuilderHandler creates a BuilderHandler.
func NewBuilderHandler(pool *alertmanager.Pool) *BuilderHandler {
	return &BuilderHandler{pool: pool}
}

// ─── Time Intervals ──────────────────────────────────────────────────────────

// ListTimeIntervals handles GET /api/builder/time-intervals?instance=<name>.
func (h *BuilderHandler) ListTimeIntervals(w http.ResponseWriter, r *http.Request) {
	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}
	entries, err := b.ListTimeIntervals()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"time_intervals": entries})
}

// GetTimeInterval handles GET /api/builder/time-intervals/{name}?instance=<name>.
func (h *BuilderHandler) GetTimeInterval(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}
	entries, err := b.ListTimeIntervals()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, e := range entries {
		if e.Name == name {
			writeJSON(w, e)
			return
		}
	}
	writeError(w, "time interval not found: "+name, http.StatusNotFound)
}

// UpsertTimeInterval handles PUT /api/builder/time-intervals/{name}.
// Body: TimeIntervalEntry JSON.  Returns the updated raw_yaml + validation.
func (h *BuilderHandler) UpsertTimeInterval(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var entry configbuilder.TimeIntervalEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	entry.Name = name // URL name is canonical; prevents accidental renames

	// Validate the interval in isolation before touching the live config.
	if result := configbuilder.ValidateTimeInterval(entry); !result.Valid {
		writeJSONStatus(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "time interval validation failed",
			"errors": result.Errors,
		})
		return
	}

	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}

	if err := b.UpsertTimeInterval(entry); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawYAML, err := b.BuildRaw()
	if err != nil {
		writeError(w, "serializing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"time_interval": entry,
		"raw_yaml":      string(rawYAML),
		"validation":    configbuilder.Validate(rawYAML),
	})
}

// DeleteTimeInterval handles DELETE /api/builder/time-intervals/{name}.
func (h *BuilderHandler) DeleteTimeInterval(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}

	found, err := b.DeleteTimeInterval(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		writeError(w, "time interval not found: "+name, http.StatusNotFound)
		return
	}

	rawYAML, err := b.BuildRaw()
	if err != nil {
		writeError(w, "serializing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"deleted":    name,
		"raw_yaml":   string(rawYAML),
		"validation": configbuilder.Validate(rawYAML),
	})
}

// ValidateTimeInterval handles POST /api/builder/time-intervals/validate.
// Body: TimeIntervalEntry JSON.  Does not modify any config.
func (h *BuilderHandler) ValidateTimeInterval(w http.ResponseWriter, r *http.Request) {
	var entry configbuilder.TimeIntervalEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	result := configbuilder.ValidateTimeInterval(entry)
	if !result.Valid {
		writeJSONStatus(w, http.StatusUnprocessableEntity, result)
		return
	}
	writeJSON(w, result)
}

// ─── Receivers ───────────────────────────────────────────────────────────────

// ListReceivers handles GET /api/builder/receivers?instance=<name>.
func (h *BuilderHandler) ListReceivers(w http.ResponseWriter, r *http.Request) {
	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}
	recs, err := b.ListReceivers()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"receivers": recs})
}

// GetReceiver handles GET /api/builder/receivers/{name}?instance=<name>.
func (h *BuilderHandler) GetReceiver(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}
	recs, err := b.ListReceivers()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, rec := range recs {
		if rec.Name == name {
			writeJSON(w, rec)
			return
		}
	}
	writeError(w, "receiver not found: "+name, http.StatusNotFound)
}

// UpsertReceiver handles PUT /api/builder/receivers/{name}.
// Body: ReceiverDef JSON.  Returns the updated raw_yaml + validation.
func (h *BuilderHandler) UpsertReceiver(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	var rec configbuilder.ReceiverDef
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	rec.Name = name // URL name is canonical

	// Validate in isolation first.
	if result := configbuilder.ValidateReceiver(rec); !result.Valid {
		writeJSONStatus(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "receiver validation failed",
			"errors": result.Errors,
		})
		return
	}

	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}

	if err := b.UpsertReceiver(rec); err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawYAML, err := b.BuildRaw()
	if err != nil {
		writeError(w, "serializing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"receiver":   rec,
		"raw_yaml":   string(rawYAML),
		"validation": configbuilder.Validate(rawYAML),
	})
}

// DeleteReceiver handles DELETE /api/builder/receivers/{name}.
func (h *BuilderHandler) DeleteReceiver(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}

	found, err := b.DeleteReceiver(name)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !found {
		writeError(w, "receiver not found: "+name, http.StatusNotFound)
		return
	}

	rawYAML, err := b.BuildRaw()
	if err != nil {
		writeError(w, "serializing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"deleted":    name,
		"raw_yaml":   string(rawYAML),
		"validation": configbuilder.Validate(rawYAML),
	})
}

// ValidateReceiver handles POST /api/builder/receivers/validate.
// Body: ReceiverDef JSON.  Does not modify any config.
func (h *BuilderHandler) ValidateReceiver(w http.ResponseWriter, r *http.Request) {
	var rec configbuilder.ReceiverDef
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	result := configbuilder.ValidateReceiver(rec)
	if !result.Valid {
		writeJSONStatus(w, http.StatusUnprocessableEntity, result)
		return
	}
	writeJSON(w, result)
}

// ─── Route ───────────────────────────────────────────────────────────────────

// GetRoute handles GET /api/builder/route?instance=<name>.
func (h *BuilderHandler) GetRoute(w http.ResponseWriter, r *http.Request) {
	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}
	route, err := b.GetRoute()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"route": route})
}

// SetRoute handles PUT /api/builder/route.
// Body: RouteSpec JSON.  Returns the updated raw_yaml + validation.
func (h *BuilderHandler) SetRoute(w http.ResponseWriter, r *http.Request) {
	var route configbuilder.RouteSpec
	if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	b := h.builderFromRequest(w, r)
	if b == nil {
		return
	}

	b.SetRoute(route)

	rawYAML, err := b.BuildRaw()
	if err != nil {
		writeError(w, "serializing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"route":      route,
		"raw_yaml":   string(rawYAML),
		"validation": configbuilder.Validate(rawYAML),
	})
}

// ─── Full config export ───────────────────────────────────────────────────────

// ExportConfig handles POST /api/builder/export.
//
// Body:
//
//	{
//	  "instance":       "<alertmanager-name>",  // optional: start from live config
//	  "route":          { ... },                // optional: override root route
//	  "receivers":      [ ... ],                // optional: upsert receivers
//	  "time_intervals": [ ... ]                 // optional: upsert time intervals
//	}
//
// Assembles a complete Alertmanager config from the provided pieces, validates
// it, and returns the YAML.  Use POST /api/config/save to persist.
func (h *BuilderHandler) ExportConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Instance      string                            `json:"instance"`
		Route         *configbuilder.RouteSpec          `json:"route,omitempty"`
		Receivers     []configbuilder.ReceiverDef       `json:"receivers,omitempty"`
		TimeIntervals []configbuilder.TimeIntervalEntry `json:"time_intervals,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Start from the live config if an instance is given, otherwise empty.
	var seedYAML []byte
	if req.Instance != "" {
		client := resolveClient(h.pool, w, req.Instance)
		if client == nil {
			return
		}
		status, err := client.GetStatus(r.Context())
		if err != nil {
			writeError(w, err.Error(), http.StatusBadGateway)
			return
		}
		seedYAML = []byte(status.Config.Original)
	}

	b, err := configbuilder.NewConfigBuilder(seedYAML)
	if err != nil {
		writeError(w, "parsing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Route != nil {
		b.SetRoute(*req.Route)
	}
	for _, rec := range req.Receivers {
		if err := b.UpsertReceiver(rec); err != nil {
			writeError(w, "receiver error: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	for _, ti := range req.TimeIntervals {
		if err := b.UpsertTimeInterval(ti); err != nil {
			writeError(w, "time interval error: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	out, err := b.BuildRaw()
	if err != nil {
		writeError(w, "serializing config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"raw_yaml":   string(out),
		"validation": configbuilder.Validate(out),
	})
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// builderFromRequest fetches the live config from the Alertmanager instance
// identified by the "instance" query parameter and returns a ConfigBuilder.
// On failure it writes an error response and returns nil.
func (h *BuilderHandler) builderFromRequest(w http.ResponseWriter, r *http.Request) *configbuilder.ConfigBuilder {
	instance := r.URL.Query().Get("instance")
	client := resolveClient(h.pool, w, instance)
	if client == nil {
		return nil
	}
	status, err := client.GetStatus(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return nil
	}
	b, err := configbuilder.NewConfigBuilder([]byte(status.Config.Original))
	if err != nil {
		writeError(w, "parsing alertmanager config: "+err.Error(), http.StatusInternalServerError)
		return nil
	}
	return b
}
