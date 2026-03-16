package handlers

import (
	"net/http"
	"strconv"
	"strings"

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
// Query params (forwarded to Alertmanager):
//   - filter      (multi-value): Alertmanager matcher strings, e.g. filter=severity="critical"
//   - instance    : restrict to one AM instance name
//   - silenced    : bool (default false)
//   - inhibited   : bool (default false)
//   - active      : bool (default true)
//
// View-layer params (applied by the handler):
//   - severity    (multi-value): filter by severity label, e.g. severity=critical&severity=warning
//   - status      (multi-value): filter by alert state (active|suppressed|unprocessed)
//   - group_by    (multi-value): group results by label key(s), e.g. group_by=severity
//   - limit       : max alerts per page (default 500, max 5000)
//   - offset      : pagination offset (default 0)
func (h *AlertsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Build base query params (forwarded to Alertmanager API).
	base := alertmanager.AlertsQueryParams{
		Filter:    q["filter"],
		Instance:  q.Get("instance"),
		Silenced:  parseBool(q.Get("silenced"), false),
		Inhibited: parseBool(q.Get("inhibited"), false),
		Active:    parseBool(q.Get("active"), true),
	}

	// Parse view-layer params.
	groupBy := q["group_by"]
	// Convenience: allow comma-separated values in a single param
	groupBy = splitCommaSeparated(groupBy)

	severity := q["severity"]
	severity = splitCommaSeparated(severity)

	status := q["status"]
	status = splitCommaSeparated(status)

	// Safely parse and validate pagination parameters
	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil || limit <= 0 || limit > 5000 {
		limit = 500 // default limit
	}

	offset, err := strconv.Atoi(q.Get("offset"))
	if err != nil || offset < 0 {
		offset = 0 // default offset
	}

	params := alertmanager.AlertsViewParams{
		AlertsQueryParams: base,
		GroupBy:           groupBy,
		Severity:          severity,
		Status:            status,
		Limit:             limit,
		Offset:            offset,
	}

	resp, err := h.pool.GetAlertsView(r.Context(), params)
	if err != nil {
		writeAMError(w, err)
		return
	}
	writeJSON(w, resp)
}

// splitCommaSeparated expands any comma-separated values within the slice and
// trims whitespace, so both ?group_by=severity,status and
// ?group_by=severity&group_by=status work.
func splitCommaSeparated(vals []string) []string {
	if len(vals) == 0 {
		return vals
	}
	var out []string
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
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
