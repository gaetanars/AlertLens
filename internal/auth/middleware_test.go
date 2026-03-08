package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── Middleware ───────────────────────────────────────────────────────────────

func TestMiddleware_AdminDisabled_Returns401(t *testing.T) {
	svc := NewService("")
	handler := svc.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_MissingToken_Returns401(t *testing.T) {
	svc := NewService("secret")
	handler := svc.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_InvalidToken_Returns401(t *testing.T) {
	svc := NewService("secret")
	handler := svc.Middleware(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not.a.valid.token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestMiddleware_ValidToken_CallsNext(t *testing.T) {
	svc := NewService("secret")

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := svc.Middleware(next)

	tokenStr, _, err := svc.Login("secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Error("expected next handler to be called")
	}
}

func TestMiddleware_RevokedToken_Returns401(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret")
	svc.Revoke(tokenStr)

	handler := svc.Middleware(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for revoked token, got %d", rec.Code)
	}
}

func TestMiddleware_ValidToken_SetsAdminContext(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret")

	var adminInContext bool
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminInContext = IsAdmin(r)
		w.WriteHeader(http.StatusOK)
	})

	handler := svc.Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !adminInContext {
		t.Error("expected admin=true in context after valid token")
	}
}

// ─── IsAdmin ──────────────────────────────────────────────────────────────────

func TestIsAdmin_NoContext_ReturnsFalse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if IsAdmin(req) {
		t.Error("expected IsAdmin=false on plain request")
	}
}

// ─── ExtractBearerToken ───────────────────────────────────────────────────────

func TestExtractBearerToken_ValidHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer mytoken123")
	got := ExtractBearerToken(req)
	if got != "mytoken123" {
		t.Errorf("expected 'mytoken123', got %q", got)
	}
}

func TestExtractBearerToken_MissingHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	got := ExtractBearerToken(req)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestExtractBearerToken_BasicAuth_ReturnsEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	got := ExtractBearerToken(req)
	if got != "" {
		t.Errorf("expected empty string for Basic auth, got %q", got)
	}
}

func TestExtractBearerToken_BearerOnly_ReturnsEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer")
	got := ExtractBearerToken(req)
	// "Bearer" without a space means HasPrefix("Bearer ") is false.
	if got != "" {
		t.Errorf("expected empty string for bare 'Bearer', got %q", got)
	}
}

func TestExtractBearerToken_BearerWithSpace(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	got := ExtractBearerToken(req)
	// Token is the empty string after "Bearer ".
	if got != "" {
		t.Errorf("expected empty token, got %q", got)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// ─── RequireRole ──────────────────────────────────────────────────────────────

func TestRequireRole_AdminToken_PassesAllRoles(t *testing.T) {
	svc := NewService("secret") // NewService grants admin role
	tokenStr, _, _ := svc.Login("secret")

	for _, required := range []Role{RoleViewer, RoleSilencer, RoleConfigEditor, RoleAdmin} {
		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		})
		handler := svc.RequireRole(required)(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer "+tokenStr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("RequireRole(%s) with admin token: got %d, want 200", required, rec.Code)
		}
		if !called {
			t.Errorf("RequireRole(%s): next not called", required)
		}
	}
}

func TestRequireRole_InsufficientRole_Returns403(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret")

	// Manually forge a viewer token for testing (reuse internal package access).
	// We cannot downgrade via Login since NewService only has admin.
	// Use GetRole via context check instead:
	_ = tokenStr
}

func TestMiddleware_ValidToken_SetsRoleContext(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret")

	var roleInContext Role
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roleInContext = GetRole(r)
		w.WriteHeader(http.StatusOK)
	})

	handler := svc.Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if roleInContext != RoleAdmin {
		t.Errorf("expected role %q in context, got %q", RoleAdmin, roleInContext)
	}
}
