package auth

import (
	"encoding/json"
	"net/http"
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
	w.Write(body) //nolint:errcheck
}
