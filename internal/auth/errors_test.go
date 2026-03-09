package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestWriteAuthError_SetsContentTypeJSON ensures the helper always sets the
// correct Content-Type, regardless of the error message.
func TestWriteAuthError_SetsContentTypeJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeAuthError(rec, "something went wrong", http.StatusUnauthorized)

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", got)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

// TestWriteAuthError_ValidJSON ensures the response body is always valid JSON.
func TestWriteAuthError_ValidJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeAuthError(rec, "test error", http.StatusForbidden)

	var out map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}
	if out["error"] != "test error" {
		t.Errorf("expected error='test error', got %q", out["error"])
	}
}

// TestWriteAuthError_EscapesSpecialChars is a CWE-116 regression test.
// If the message contains JSON-special characters ('"', '\', newlines),
// the response must still be valid JSON and must not allow injection.
func TestWriteAuthError_EscapesSpecialChars(t *testing.T) {
	evil := `he said "hello" and \n escaped`
	rec := httptest.NewRecorder()
	writeAuthError(rec, evil, http.StatusBadRequest)

	var out map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("response with special chars is not valid JSON: %v\nbody: %s", err, rec.Body.String())
	}
	if out["error"] != evil {
		t.Errorf("error message was mangled: got %q, want %q", out["error"], evil)
	}
}
