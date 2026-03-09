package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alertlens/alertlens/internal/auth"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

func newAuthHandler(password string) *AuthHandler {
	return NewAuthHandler(auth.NewService(password))
}

func decodeJSON(t *testing.T, body *bytes.Buffer) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(body).Decode(&out); err != nil {
		t.Fatalf("decoding response JSON: %v", err)
	}
	return out
}

// ─── Status ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Status_AdminDisabled(t *testing.T) {
	h := newAuthHandler("")
	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec.Body)
	if body["admin_enabled"] != false {
		t.Errorf("expected admin_enabled=false, got %v", body["admin_enabled"])
	}
	if body["authenticated"] != false {
		t.Errorf("expected authenticated=false, got %v", body["authenticated"])
	}
}

func TestAuthHandler_Status_AdminEnabled_NotAuthenticated(t *testing.T) {
	h := newAuthHandler("secret")
	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec.Body)
	if body["admin_enabled"] != true {
		t.Errorf("expected admin_enabled=true, got %v", body["admin_enabled"])
	}
	if body["authenticated"] != false {
		t.Errorf("expected authenticated=false without token, got %v", body["authenticated"])
	}
}

func TestAuthHandler_Status_AdminEnabled_Authenticated(t *testing.T) {
	svc := auth.NewService("secret")
	h := NewAuthHandler(svc)

	tokenStr, _, err := svc.Login("secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec.Body)
	if body["authenticated"] != true {
		t.Errorf("expected authenticated=true with valid token, got %v", body["authenticated"])
	}
}

func TestAuthHandler_Status_AdminEnabled_InvalidToken(t *testing.T) {
	h := newAuthHandler("secret")
	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()
	h.Status(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	body := decodeJSON(t, rec.Body)
	if body["authenticated"] != false {
		t.Errorf("expected authenticated=false for invalid token, got %v", body["authenticated"])
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestAuthHandler_Login_ValidPassword(t *testing.T) {
	h := newAuthHandler("secret")
	body := bytes.NewBufferString(`{"password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	resp := decodeJSON(t, rec.Body)
	if _, ok := resp["token"].(string); !ok {
		t.Errorf("expected 'token' field in response, got: %v", resp)
	}
	if _, ok := resp["expires_at"].(string); !ok {
		t.Errorf("expected 'expires_at' field in response, got: %v", resp)
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	h := newAuthHandler("secret")
	body := bytes.NewBufferString(`{"password":"wrongpassword"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_AdminDisabled(t *testing.T) {
	h := newAuthHandler("")
	body := bytes.NewBufferString(`{"password":"anything"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	h := newAuthHandler("secret")
	body := bytes.NewBufferString(`{not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_EmptyBody(t *testing.T) {
	h := newAuthHandler("secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{}`))
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	// Empty password won't match a non-empty configured password.
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandler_Login_ResponseContentType(t *testing.T) {
	h := newAuthHandler("secret")
	body := bytes.NewBufferString(`{"password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	rec := httptest.NewRecorder()
	h.Login(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// ─── Logout ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_WithValidToken(t *testing.T) {
	svc := auth.NewService("secret")
	h := NewAuthHandler(svc)

	tokenStr, _, _ := svc.Login("secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	// Token must now be invalid.
	if _, _, err := svc.Validate(tokenStr); err == nil {
		t.Error("expected token to be invalid after logout")
	}
}

func TestAuthHandler_Logout_WithNoToken(t *testing.T) {
	h := newAuthHandler("secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	// Logout without a token is a no-op but should still return 204.
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

func TestAuthHandler_Logout_WithInvalidToken(t *testing.T) {
	h := newAuthHandler("secret")
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer this.is.garbage")
	rec := httptest.NewRecorder()
	h.Logout(rec, req)

	// Revoke on invalid token is a no-op.
	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}
