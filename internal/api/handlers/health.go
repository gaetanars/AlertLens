package handlers

import "net/http"

// HealthHandler handles the /api/health endpoint.
// The version string is injected at construction time, avoiding global mutable state.
type HealthHandler struct {
	version string
}

// NewHealthHandler creates a HealthHandler with the given build version.
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

// Health handles GET /api/health.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"status":  "ok",
		"version": h.version,
	})
}
