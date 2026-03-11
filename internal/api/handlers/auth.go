package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/alertlens/alertlens/internal/auth"
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
	role := ""
	if token != "" {
		if _, r, err := h.svc.Validate(token); err == nil {
			authenticated = true
			role = string(r)
		}
	}
	writeJSON(w, map[string]any{
		"admin_enabled": h.svc.AdminEnabled(),
		"authenticated": authenticated,
		"role":          role,
	})
}

// Login handles POST /api/auth/login.
//
// Request body:
//
//	{ "password": "...", "totp_code": "123456" }
//
// totp_code is required when the user's account has MFA enabled.
// If the password is correct but MFA is required and totp_code is absent,
// the response is 401 with { "error": "MFA challenge required", "mfa_required": true }.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
		TOTPCode string `json:"totp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token, exp, err := h.svc.Login(body.Password, body.TOTPCode)
	if err != nil {
		if errors.Is(err, auth.ErrMFARequired) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			if err := json.NewEncoder(w).Encode(map[string]any{
				"error":        err.Error(),
				"mfa_required": true,
			}); err != nil {
				log.Printf("failed to encode MFA response: %v", err)
			}
			return
		}
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
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

// writeJSONStatus writes v as JSON with an explicit HTTP status code.
// Headers must be set before WriteHeader, so always use this instead of
// calling w.WriteHeader followed by writeJSON.
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		log.Printf("failed to encode error response: %v", err)
	}
}

