package handlers

import (
	"net/http"

	"github.com/alertlens/alertlens/internal/alertmanager"
)

// AlertmanagersHandler handles requests about AM instances.
type AlertmanagersHandler struct {
	pool *alertmanager.Pool
}

// NewAlertmanagersHandler creates an AlertmanagersHandler.
func NewAlertmanagersHandler(pool *alertmanager.Pool) *AlertmanagersHandler {
	return &AlertmanagersHandler{pool: pool}
}

// List handles GET /api/alertmanagers.
func (h *AlertmanagersHandler) List(w http.ResponseWriter, r *http.Request) {
	statuses := h.pool.GetInstanceStatuses(r.Context())
	writeJSON(w, statuses)
}
