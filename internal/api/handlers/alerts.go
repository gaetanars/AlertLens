package handlers

import (
	"net/http"
	"strconv"

	"github.com/alertlens/alertlens/internal/alertmanager"
)

// AlertsHandler handles alert-related requests.
type AlertsHandler struct {
	pool *alertmanager.Pool
}

// NewAlertsHandler creates an AlertsHandler.
func NewAlertsHandler(pool *alertmanager.Pool) *AlertsHandler {
	return &AlertsHandler{pool: pool}
}

// List handles GET /api/alerts.
//
// Query params:
//   - filter  (multi-value): Alertmanager matcher strings
//   - instance: restrict to one AM instance
//   - silenced: bool (default false)
//   - inhibited: bool (default false)
//   - active: bool (default true)
func (h *AlertsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := alertmanager.AlertsQueryParams{
		Filter:    q["filter"],
		Instance:  q.Get("instance"),
		Silenced:  parseBool(q.Get("silenced"), false),
		Inhibited: parseBool(q.Get("inhibited"), false),
		Active:    parseBool(q.Get("active"), true),
	}

	alerts, err := h.pool.GetAggregatedAlerts(r.Context(), params)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}
	if alerts == nil {
		alerts = []alertmanager.EnrichedAlert{}
	}
	writeJSON(w, alerts)
}

func parseBool(s string, def bool) bool {
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}
