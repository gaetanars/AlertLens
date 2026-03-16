package alertmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alertlens/alertlens/internal/config"
)

// TestNewClientUserAgent verifies that every outgoing request to Alertmanager
// carries the correct User-Agent header in the form "alertlens/<version>".
func TestNewClientUserAgent(t *testing.T) {
	t.Parallel()

	var gotUserAgent string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserAgent = r.Header.Get("User-Agent")
		// Return a valid (empty) alerts JSON response.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]Alert{}) //nolint:errcheck
	}))
	defer srv.Close()

	c := NewClient(config.AlertmanagerConfig{Name: "test", URL: srv.URL}, "1.2.3")

	_, err := c.GetAlerts(context.Background(), AlertsQueryParams{})
	if err != nil {
		t.Fatalf("GetAlerts: %v", err)
	}

	want := "alertlens/1.2.3"
	if gotUserAgent != want {
		t.Errorf("User-Agent: got %q, want %q", gotUserAgent, want)
	}
}

// TestUpstreamErrorSanitization verifies that when Alertmanager returns a
// non-success status, the raw response body is NOT forwarded to the caller via
// err.Error() — only a generic message is surfaced.
func TestUpstreamErrorSanitization(t *testing.T) {
	t.Parallel()

	secretBody := "internal AM stack trace secret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(secretBody)) //nolint:errcheck
	}))
	defer srv.Close()

	c := NewClient(config.AlertmanagerConfig{Name: "test", URL: srv.URL}, "0.0.0")

	_, err := c.GetAlerts(context.Background(), AlertsQueryParams{})
	if err == nil {
		t.Fatal("expected an error from GetAlerts, got nil")
	}

	// Must be detected as an upstream error.
	if _, ok := IsUpstreamError(err); !ok {
		t.Errorf("expected IsUpstreamError(err) == true, got false; err=%v", err)
	}

	// The raw AM body must NOT be visible in err.Error().
	if strings.Contains(err.Error(), secretBody) {
		t.Errorf("err.Error() must not contain the raw AM body; got: %q", err.Error())
	}
}
