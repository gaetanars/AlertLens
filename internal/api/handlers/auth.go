package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gaetanars/alertlens/internal/auth"
)

// AuthHandler holds the auth service for HTTP handler use.
type AuthHandler struct {
	svc *auth.Service
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// Status handles GET /api/auth/status.
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	token := auth.ExtractBearerToken(r)
	authenticated := false
	if token != "" {
		if _, err := h.svc.Validate(token); err == nil {
			authenticated = true
		}
	}
	writeJSON(w, map[string]any{
		"admin_enabled": h.svc.AdminEnabled(),
		"authenticated": authenticated,
	})
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, exp, err := h.svc.Login(body.Password)
	if err != nil {
		writeError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	writeJSON(w, map[string]any{
		"token":      token,
		"expires_at": exp.Format(time.RFC3339),
	})
}

// Logout handles POST /api/auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := auth.ExtractBearerToken(r)
	if token != "" {
		h.svc.Revoke(token)
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Shared helpers ──────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// writeJSONStatus writes v as JSON with an explicit HTTP status code.
// Headers must be set before WriteHeader, so always use this instead of
// calling w.WriteHeader followed by writeJSON.
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func writeError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

