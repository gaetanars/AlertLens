package auth

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

// Sentinel errors returned by the auth package.
var (
	// ErrInvalidTOTP is returned when a TOTP code does not match the secret
	// or is outside the allowed clock-skew window.
	ErrInvalidTOTP = errors.New("invalid or expired TOTP code")

	// ErrMFARequired is returned by Login when the user's account has TOTP
	// enabled and the caller must complete an MFA challenge before a full
	// session token is issued.
	ErrMFARequired = errors.New("MFA challenge required")
)

// writeAuthError writes a JSON {"error": msg} response with the given HTTP status.
//
// Using json.Marshal (rather than string concatenation) ensures that msg is
// properly escaped, preventing JSON injection if the message ever contains
// characters such as '"', '\', or control characters.
//
// SEC-CWE-116: all auth-layer error responses must go through this function.
func writeAuthError(w http.ResponseWriter, msg string, status int) {
	body, err := json.Marshal(map[string]string{"error": msg})
	if err != nil {
		// Unreachable for a plain map[string]string, but be defensive.
		body = []byte(`{"error":"internal error"}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		log.Printf("failed to write auth error response: %v", err)
	}
}
