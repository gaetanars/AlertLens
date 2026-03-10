package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testCSRFSecret = []byte("test-secret-csrf")

// ─── CSRFMiddleware ───────────────────────────────────────────────────────────

func TestCSRF_SafeMethod_SetsCookieAndHeader(t *testing.T) {
	handler := CSRFMiddleware(testCSRFSecret, false)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// X-CSRF-Token response header must be set.
	if rec.Header().Get(CSRFHeaderName) == "" {
		t.Error("expected X-CSRF-Token response header to be set")
	}
	// Cookie must be set.
	cookies := rec.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == csrfCookieName {
			found = true
			if c.Value == "" {
				t.Error("csrf_token cookie value is empty")
			}
		}
	}
	if !found {
		t.Error("expected csrf_token cookie to be set")
	}
}

func TestCSRF_BearerRequest_SkipsValidation(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret", "")

	handler := CSRFMiddleware(testCSRFSecret, false)(okHandler())

	// POST without CSRF cookie/header but with Bearer token — should pass.
	req := httptest.NewRequest(http.MethodPost, "/api/silences", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Bearer-authenticated POST should bypass CSRF: got %d", rec.Code)
	}
}

func TestCSRF_MutatingRequest_NoCookie_Returns403(t *testing.T) {
	handler := CSRFMiddleware(testCSRFSecret, false)(okHandler())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 without CSRF cookie, got %d", rec.Code)
	}
}

func TestCSRF_MutatingRequest_NoHeader_Returns403(t *testing.T) {
	handler := CSRFMiddleware(testCSRFSecret, false)(okHandler())

	// Obtain a valid token first via GET.
	getReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()

	// POST with cookie but without header.
	postReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}"))
	for _, c := range cookies {
		postReq.AddCookie(c)
	}
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 without X-CSRF-Token header, got %d", postRec.Code)
	}
}

func TestCSRF_MutatingRequest_ValidTokens_Passes(t *testing.T) {
	handler := CSRFMiddleware(testCSRFSecret, false)(okHandler())

	// Step 1: GET to obtain the token.
	getReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	csrfToken := getRec.Header().Get(CSRFHeaderName)
	cookies := getRec.Result().Cookies()

	// Step 2: POST with matching cookie and header.
	postReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}"))
	for _, c := range cookies {
		postReq.AddCookie(c)
	}
	postReq.Header.Set(CSRFHeaderName, csrfToken)
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Errorf("expected 200 with valid CSRF tokens, got %d", postRec.Code)
	}
}

func TestCSRF_MutatingRequest_MismatchedTokens_Returns403(t *testing.T) {
	handler := CSRFMiddleware(testCSRFSecret, false)(okHandler())

	// Step 1: GET to obtain a valid cookie.
	getReq := httptest.NewRequest(http.MethodGet, "/api/auth/status", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	cookies := getRec.Result().Cookies()

	// Step 2: POST with cookie but a *different* header value.
	postReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}"))
	for _, c := range cookies {
		postReq.AddCookie(c)
	}
	postReq.Header.Set(CSRFHeaderName, "tampered-value")
	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for mismatched tokens, got %d", postRec.Code)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func TestCSRFToken_GenerateAndValidate(t *testing.T) {
	secret := []byte("mysecret")
	tok := generateCSRFToken(secret)
	if !validateCSRFToken(tok, secret) {
		t.Error("freshly generated token should be valid")
	}
}

func TestCSRFToken_TamperedPayload_Invalid(t *testing.T) {
	secret := []byte("mysecret")
	tok := generateCSRFToken(secret)
	tampered := "deadbeef" + tok[8:] // flip first bytes
	if validateCSRFToken(tampered, secret) {
		t.Error("tampered token should be invalid")
	}
}

func TestCSRFToken_WrongSecret_Invalid(t *testing.T) {
	tok := generateCSRFToken([]byte("secret-a"))
	if validateCSRFToken(tok, []byte("secret-b")) {
		t.Error("token signed with different secret should be invalid")
	}
}
