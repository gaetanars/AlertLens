package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

const (
	csrfCookieName = "csrf_token"
	// CSRFHeaderName is the request/response header used for CSRF token exchange.
	// Exported so the CORS configuration can include it in AllowedHeaders.
	CSRFHeaderName = "X-CSRF-Token"
	csrfTokenTTL   = 8 * time.Hour
)

// CSRFMiddleware provides stateless CSRF protection using the
// **signed double-submit cookie** pattern (OWASP recommended for SPAs).
//
// How it works:
//
//  1. On safe methods (GET, HEAD, OPTIONS) the middleware generates a
//     cryptographically signed token, sets it as a SameSite=Lax cookie
//     (readable by JS — intentional), and echoes it in the X-CSRF-Token
//     response header so the SPA can prime its in-memory CSRF store.
//
//  2. On state-mutating methods (POST, PUT, PATCH, DELETE):
//     - Requests carrying a valid Bearer Authorization header are exempt:
//       browsers cannot set that header in a cross-site request without
//       a CORS preflight, so Bearer auth already acts as a CSRF defence.
//     - All other requests (e.g. the unauthenticated /auth/login POST)
//       must supply the X-CSRF-Token header whose value matches the
//       csrf_token cookie.  Both are verified against the server secret
//       to prevent cookie-injection attacks.
//
// Token format: <16-byte-random-hex>.<HMAC-SHA256(secret, random-hex)-hex>
//
// The secureCookies parameter controls the Secure attribute on the csrf_token
// cookie; set to true when the application is served over HTTPS.
func CSRFMiddleware(secret []byte, secureCookies bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				// Always (re)set a fresh token on safe requests so the SPA
				// always has a valid token ready.
				token := generateCSRFToken(secret)
				setCSRFCookie(w, token, secureCookies)
				w.Header().Set(CSRFHeaderName, token)
				next.ServeHTTP(w, r)
				return
			}

			// Bearer-authenticated requests are CSRF-exempt.
			if ExtractBearerToken(r) != "" {
				next.ServeHTTP(w, r)
				return
			}

			// ── Double-submit validation ──────────────────────────────────
			cookie, err := r.Cookie(csrfCookieName)
			if err != nil || cookie.Value == "" {
				writeAuthError(w, "missing CSRF cookie", http.StatusForbidden)
				return
			}
			headerVal := r.Header.Get(CSRFHeaderName)
			if headerVal == "" {
				writeAuthError(w, "missing X-CSRF-Token header", http.StatusForbidden)
				return
			}
			// Constant-time comparison to prevent timing oracle.
			if !hmac.Equal([]byte(cookie.Value), []byte(headerVal)) {
				writeAuthError(w, "CSRF token mismatch", http.StatusForbidden)
				return
			}
			// Verify the token is genuinely signed by our secret (prevents
			// an attacker from crafting matching cookie + header values).
			if !validateCSRFToken(cookie.Value, secret) {
				writeAuthError(w, "invalid CSRF token signature", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// generateCSRFToken creates a new signed token: <random-hex>.<hmac-hex>
func generateCSRFToken(secret []byte) string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		panic("csrf: rand.Read failed: " + err.Error())
	}
	rawHex := hex.EncodeToString(raw)
	mac := computeCSRFMAC(secret, rawHex)
	return rawHex + "." + mac
}

// validateCSRFToken returns true if the token was signed by secret.
func validateCSRFToken(token string, secret []byte) bool {
	dot := strings.LastIndex(token, ".")
	if dot < 0 {
		return false
	}
	rawHex := token[:dot]
	providedMAC := token[dot+1:]
	expectedMAC := computeCSRFMAC(secret, rawHex)
	return hmac.Equal([]byte(providedMAC), []byte(expectedMAC))
}

// computeCSRFMAC returns hex(HMAC-SHA256(secret, data)).
func computeCSRFMAC(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data)) //nolint:errcheck
	return hex.EncodeToString(mac.Sum(nil))
}

// setCSRFCookie writes the csrf_token cookie.
// When secureCookies is true the Secure attribute is set, which restricts
// delivery to HTTPS connections (required in production).
func setCSRFCookie(w http.ResponseWriter, token string, secureCookies bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(csrfTokenTTL),
		HttpOnly: false, // must be readable by JS so the SPA can copy it to the header
		SameSite: http.SameSiteLaxMode,
		Secure:   secureCookies,
	})
}

// isSafeMethod returns true for HTTP methods that do not mutate state.
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	return false
}
