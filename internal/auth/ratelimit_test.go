package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// ─── LoginRateLimiter.Middleware ──────────────────────────────────────────────

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := NewLoginRateLimiter(0)
	handler := rl.Middleware(okHandler())

	// loginBurst = 5; all requests from the same IP should pass.
	for i := range loginBurst {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = "1.2.3.4:9999"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	rl := NewLoginRateLimiter(0)
	handler := rl.Middleware(okHandler())

	ip := "5.6.7.8:1234"

	// Consume the burst.
	for range loginBurst {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = ip
		httptest.NewRecorder()
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// Next request must be rejected.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after burst exhausted, got %d", rec.Code)
	}
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	rl := NewLoginRateLimiter(0)
	handler := rl.Middleware(okHandler())

	ip := "10.0.0.1:80"
	for range loginBurst {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = ip
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = ip
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

func TestRateLimiter_PerIP_IndependentLimits(t *testing.T) {
	rl := NewLoginRateLimiter(0)
	handler := rl.Middleware(okHandler())

	// Exhaust burst for IP A.
	ipA := "192.0.2.1:5000"
	for range loginBurst {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.RemoteAddr = ipA
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	// IP B should still be allowed (fresh limiter).
	ipB := "192.0.2.2:5000"
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = ipB
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("IP B should not be rate-limited; expected 200, got %d", rec.Code)
	}
}

// ─── extractRemoteIP ──────────────────────────────────────────────────────────

func TestExtractRemoteIP_HostPortFormat(t *testing.T) {
	got := extractRemoteIP("192.168.1.1:8080")
	if got != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got %q", got)
	}
}

func TestExtractRemoteIP_IPv6WithPort(t *testing.T) {
	got := extractRemoteIP("[::1]:9000")
	if got != "::1" {
		t.Errorf("expected '::1', got %q", got)
	}
}

func TestExtractRemoteIP_BareIP_ReturnedUnchanged(t *testing.T) {
	got := extractRemoteIP("10.0.0.1")
	if got != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1' unchanged, got %q", got)
	}
}

func TestExtractRemoteIP_EmptyString(t *testing.T) {
	got := extractRemoteIP("")
	// SplitHostPort will fail; the original value is returned.
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}
