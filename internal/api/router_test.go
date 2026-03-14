package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── buildCSPPolicy ───────────────────────────────────────────────────────────

func TestBuildCSPPolicy_NoHashes(t *testing.T) {
	policy := buildCSPPolicy("")

	// script-src must be 'self' only — no unsafe-inline, no hashes.
	scriptSrc := extractDirective(policy, "script-src")
	if scriptSrc == "" {
		t.Fatal("script-src directive missing from CSP policy")
	}
	if !strings.Contains(scriptSrc, "'self'") {
		t.Errorf("expected script-src to contain 'self', got: %q", scriptSrc)
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src must not contain 'unsafe-inline', got: %q", scriptSrc)
	}
	if strings.Contains(scriptSrc, "sha256-") {
		t.Errorf("script-src should not contain a hash when none was provided, got: %q", scriptSrc)
	}
}

func TestBuildCSPPolicy_WithSingleHash(t *testing.T) {
	hash := "'sha256-sT2qA9cf7BKsSdySbq8RHggzSGve/KytHCRr70CnQxY='"
	policy := buildCSPPolicy(hash)

	scriptSrc := extractDirective(policy, "script-src")
	if !strings.Contains(scriptSrc, "'self'") {
		t.Errorf("expected script-src to contain 'self', got: %q", scriptSrc)
	}
	if !strings.Contains(scriptSrc, hash) {
		t.Errorf("expected script-src to contain the supplied hash, got: %q", scriptSrc)
	}
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src must not contain 'unsafe-inline', got: %q", scriptSrc)
	}
}

func TestBuildCSPPolicy_WithMultipleHashes(t *testing.T) {
	hash1 := "'sha256-aaaa='"
	hash2 := "'sha256-bbbb='"
	policy := buildCSPPolicy(hash1 + " " + hash2)

	scriptSrc := extractDirective(policy, "script-src")
	if !strings.Contains(scriptSrc, hash1) {
		t.Errorf("expected script-src to contain first hash, got: %q", scriptSrc)
	}
	if !strings.Contains(scriptSrc, hash2) {
		t.Errorf("expected script-src to contain second hash, got: %q", scriptSrc)
	}
}

func TestBuildCSPPolicy_OtherDirectivesUnaffected(t *testing.T) {
	policy := buildCSPPolicy("")

	checks := map[string]string{
		"default-src":     "'self'",
		"style-src":       "'unsafe-inline'",
		"object-src":      "'none'",
		"frame-ancestors": "'none'",
		"base-uri":        "'self'",
		"form-action":     "'self'",
	}

	for directive, expected := range checks {
		value := extractDirective(policy, directive)
		if !strings.Contains(value, expected) {
			t.Errorf("directive %q: expected %q in %q", directive, expected, value)
		}
	}
}

// ─── cspMiddlewareWithPolicy ──────────────────────────────────────────────────

func TestCSPMiddlewareWithPolicy_SetsHeaders(t *testing.T) {
	policy := buildCSPPolicy("'sha256-test='")

	handler := cspMiddlewareWithPolicy(policy, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Security-Policy"); got != policy {
		t.Errorf("Content-Security-Policy header mismatch\n  want: %q\n  got:  %q", policy, got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options: want %q, got %q", "nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options: want %q, got %q", "DENY", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got == "" {
		t.Error("Referrer-Policy header must not be empty")
	}
}

func TestCSPMiddlewareWithPolicy_NoUnsafeInlineInScriptSrc(t *testing.T) {
	// Even when the caller passes an empty hash string, the resulting CSP
	// header must never advertise 'unsafe-inline' in script-src.
	policy := buildCSPPolicy("")
	handler := cspMiddlewareWithPolicy(policy, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cspHeader := rec.Header().Get("Content-Security-Policy")
	scriptSrc := extractDirective(cspHeader, "script-src")
	if strings.Contains(scriptSrc, "'unsafe-inline'") {
		t.Errorf("script-src must not contain 'unsafe-inline'; CSP: %q", cspHeader)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// extractDirective returns the value portion of a CSP directive (e.g.
// "script-src 'self' 'sha256-xxx'" → "'self' 'sha256-xxx'") or an empty
// string if the directive is not found.
func extractDirective(policy, directive string) string {
	for _, part := range strings.Split(policy, ";") {
		part = strings.TrimSpace(part)
		prefix := directive + " "
		if strings.HasPrefix(part, prefix) {
			return strings.TrimPrefix(part, prefix)
		}
	}
	return ""
}
