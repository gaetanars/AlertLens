package alertmanager_test

// Integration-style tests for the Pool aggregation layer.
// They use httptest servers to simulate real Alertmanager instances so we can
// cover parallel fetching, partial failures, and instance filtering without
// mocking internal Pool state.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"github.com/alertlens/alertlens/internal/config"
	"go.uber.org/zap"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// newTestAlert builds a minimal Alert with the supplied labels.
func newTestAlert(fingerprint, alertname, severity, instance string) alertmanager.Alert {
	return alertmanager.Alert{
		Fingerprint: fingerprint,
		Labels: map[string]string{
			"alertname": alertname,
			"severity":  severity,
			"instance":  instance,
		},
		Annotations: map[string]string{},
		StartsAt:    time.Now().Add(-5 * time.Minute),
		EndsAt:      time.Now().Add(5 * time.Minute),
		Status:      alertmanager.AlertStatus{State: "active"},
		Receivers:   []alertmanager.Receiver{{Name: "default"}},
	}
}

// alertmanagerServer starts an httptest.Server that returns the given alerts
// via GET /api/v2/alerts and an empty silence list via GET /api/v2/silences.
func alertmanagerServer(t *testing.T, alerts []alertmanager.Alert) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(alerts) //nolint:errcheck
	})
	mux.HandleFunc("/api/v2/silences", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]alertmanager.Silence{}) //nolint:errcheck
	})
	return httptest.NewServer(mux)
}

// failingServer returns a server that always responds with 500.
func failingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
}

// buildPool creates a Pool from a list of (name, url) pairs.
func buildPool(t *testing.T, instances []struct{ name, url string }) *alertmanager.Pool {
	t.Helper()
	cfgs := make([]config.AlertmanagerConfig, len(instances))
	for i, inst := range instances {
		cfgs[i] = config.AlertmanagerConfig{Name: inst.name, URL: inst.url}
	}
	logger, _ := zap.NewDevelopment()
	return alertmanager.NewPool(cfgs, logger, "test")
}

// ─── parallel aggregation tests ──────────────────────────────────────────────

// TestGetAggregatedAlerts_MultiInstance verifies that alerts from multiple
// instances are fetched in parallel and merged into a single flat list.
func TestGetAggregatedAlerts_MultiInstance(t *testing.T) {
	srv1 := alertmanagerServer(t, []alertmanager.Alert{
		newTestAlert("fp1", "CPUHigh", "critical", "eu"),
		newTestAlert("fp2", "MemHigh", "warning", "eu"),
	})
	defer srv1.Close()

	srv2 := alertmanagerServer(t, []alertmanager.Alert{
		newTestAlert("fp3", "DiskFull", "critical", "us"),
	})
	defer srv2.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", srv1.URL},
		{"prod-us", srv2.URL},
	})

	alerts, errs := pool.GetAggregatedAlerts(context.Background(), alertmanager.AlertsQueryParams{Active: true})
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if len(alerts) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(alerts))
	}

	// Verify InstanceID / Alertmanager fields are populated.
	for _, a := range alerts {
		if a.Alertmanager == "" {
			t.Errorf("alert %s has empty Alertmanager field", a.Fingerprint)
		}
		if a.InstanceID == "" {
			t.Errorf("alert %s has empty InstanceID field", a.Fingerprint)
		}
		if a.Alertmanager != a.InstanceID {
			t.Errorf("alert %s: Alertmanager %q != InstanceID %q", a.Fingerprint, a.Alertmanager, a.InstanceID)
		}
	}

	// Verify instance names are set correctly.
	instanceCounts := map[string]int{}
	for _, a := range alerts {
		instanceCounts[a.Alertmanager]++
	}
	if instanceCounts["prod-eu"] != 2 {
		t.Errorf("expected 2 alerts from prod-eu, got %d", instanceCounts["prod-eu"])
	}
	if instanceCounts["prod-us"] != 1 {
		t.Errorf("expected 1 alert from prod-us, got %d", instanceCounts["prod-us"])
	}
}

// TestGetAggregatedAlerts_InstanceFilter verifies that the instance parameter
// restricts queries to a single instance.
func TestGetAggregatedAlerts_InstanceFilter(t *testing.T) {
	srv1 := alertmanagerServer(t, []alertmanager.Alert{
		newTestAlert("fp1", "CPUHigh", "critical", "eu"),
	})
	defer srv1.Close()

	srv2 := alertmanagerServer(t, []alertmanager.Alert{
		newTestAlert("fp2", "DiskFull", "critical", "us"),
	})
	defer srv2.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", srv1.URL},
		{"prod-us", srv2.URL},
	})

	// Filter to prod-us only.
	alerts, errs := pool.GetAggregatedAlerts(context.Background(), alertmanager.AlertsQueryParams{
		Active:   true,
		Instance: "prod-us",
	})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Alertmanager != "prod-us" {
		t.Errorf("expected alert from prod-us, got %q", alerts[0].Alertmanager)
	}
}

// ─── partial failure tests ────────────────────────────────────────────────────

// TestGetAggregatedAlerts_PartialFailure verifies degraded mode: one instance
// fails, alerts from healthy instances are still returned.
func TestGetAggregatedAlerts_PartialFailure(t *testing.T) {
	healthy := alertmanagerServer(t, []alertmanager.Alert{
		newTestAlert("fp1", "CPUHigh", "critical", "eu"),
		newTestAlert("fp2", "MemHigh", "warning", "eu"),
	})
	defer healthy.Close()

	broken := failingServer(t)
	defer broken.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", healthy.URL},
		{"prod-us", broken.URL},
	})

	alerts, errs := pool.GetAggregatedAlerts(context.Background(), alertmanager.AlertsQueryParams{Active: true})

	// Should get alerts from the healthy instance.
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts from healthy instance, got %d", len(alerts))
	}
	// Should record the failure.
	if len(errs) != 1 {
		t.Fatalf("expected 1 instance error, got %d", len(errs))
	}
	if errs[0].Instance != "prod-us" {
		t.Errorf("expected error from prod-us, got %q", errs[0].Instance)
	}
	if errs[0].Error == "" {
		t.Error("expected non-empty error message")
	}
}

// TestGetAggregatedAlerts_AllFail verifies that when all instances fail, the
// errors slice contains entries for each instance and no alerts are returned.
func TestGetAggregatedAlerts_AllFail(t *testing.T) {
	broken1 := failingServer(t)
	defer broken1.Close()
	broken2 := failingServer(t)
	defer broken2.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", broken1.URL},
		{"prod-us", broken2.URL},
	})

	alerts, errs := pool.GetAggregatedAlerts(context.Background(), alertmanager.AlertsQueryParams{Active: true})
	if len(alerts) != 0 {
		t.Errorf("expected no alerts, got %d", len(alerts))
	}
	if len(errs) != 2 {
		t.Errorf("expected 2 instance errors, got %d: %v", len(errs), errs)
	}

	instanceNames := []string{}
	for _, e := range errs {
		instanceNames = append(instanceNames, e.Instance)
	}
	sort.Strings(instanceNames)
	if instanceNames[0] != "prod-eu" || instanceNames[1] != "prod-us" {
		t.Errorf("unexpected instance names in errors: %v", instanceNames)
	}
}

// TestGetAlertsView_AllFailReturnsError verifies that GetAlertsView returns a
// hard error when all instances fail (so the API can return 502).
func TestGetAlertsView_AllFailReturnsError(t *testing.T) {
	broken := failingServer(t)
	defer broken.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"only-instance", broken.URL},
	})

	_, err := pool.GetAlertsView(context.Background(), alertmanager.AlertsViewParams{
		AlertsQueryParams: alertmanager.AlertsQueryParams{Active: true},
	})
	if err == nil {
		t.Fatal("expected error when all instances fail, got nil")
	}
}

// TestGetAlertsView_PartialFailureCarriedInResponse verifies that partial
// failure metadata is present in the AlertsResponse when one instance fails.
func TestGetAlertsView_PartialFailureCarriedInResponse(t *testing.T) {
	healthy := alertmanagerServer(t, []alertmanager.Alert{
		newTestAlert("fp1", "CPUHigh", "critical", "eu"),
	})
	defer healthy.Close()

	broken := failingServer(t)
	defer broken.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", healthy.URL},
		{"prod-us", broken.URL},
	})

	resp, err := pool.GetAlertsView(context.Background(), alertmanager.AlertsViewParams{
		AlertsQueryParams: alertmanager.AlertsQueryParams{Active: true},
	})
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected 1 alert, got %d", resp.Total)
	}
	if len(resp.PartialFailures) != 1 {
		t.Fatalf("expected 1 partial failure, got %d", len(resp.PartialFailures))
	}
	if resp.PartialFailures[0].Instance != "prod-us" {
		t.Errorf("expected failure from prod-us, got %q", resp.PartialFailures[0].Instance)
	}
}

// TestGetAggregatedAlerts_EmptyPool verifies that an empty pool returns no
// alerts and no errors.
func TestGetAggregatedAlerts_EmptyPool(t *testing.T) {
	pool := buildPool(t, nil)
	alerts, errs := pool.GetAggregatedAlerts(context.Background(), alertmanager.AlertsQueryParams{})
	if len(alerts) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(alerts))
	}
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

// TestGetAggregatedAlerts_ContextCancellation verifies that a cancelled context
// is handled gracefully (errors are logged, no panic).
func TestGetAggregatedAlerts_ContextCancellation(t *testing.T) {
	// Slow server that blocks until the client times out.
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer slow.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"slow-instance", slow.URL},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	alerts, errs := pool.GetAggregatedAlerts(ctx, alertmanager.AlertsQueryParams{Active: true})
	// We expect 0 alerts and 1 error (context deadline / connection refused).
	if len(alerts) != 0 {
		t.Errorf("expected no alerts from slow/cancelled instance, got %d", len(alerts))
	}
	if len(errs) == 0 {
		t.Error("expected at least 1 instance error on context cancellation")
	}
}

// ─── silence aggregation tests ────────────────────────────────────────────────

// TestGetAggregatedSilences_MultiInstance verifies aggregation of silences.
func TestGetAggregatedSilences_MultiInstance(t *testing.T) {
	now := time.Now()
	silence := func(id, name string) alertmanager.Silence {
		return alertmanager.Silence{
			ID:        id,
			Matchers:  []alertmanager.Matcher{{Name: "alertname", Value: name, IsEqual: true}},
			StartsAt:  now.Add(-time.Hour),
			EndsAt:    now.Add(time.Hour),
			CreatedBy: "alice",
			Comment:   "test",
			Status:    alertmanager.SilenceStatus{State: "active"},
		}
	}

	mux1 := http.NewServeMux()
	mux1.HandleFunc("/api/v2/silences", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]alertmanager.Silence{silence("s1", "CPUHigh")}) //nolint:errcheck
	})
	srv1 := httptest.NewServer(mux1)
	defer srv1.Close()

	mux2 := http.NewServeMux()
	mux2.HandleFunc("/api/v2/silences", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]alertmanager.Silence{silence("s2", "MemHigh"), silence("s3", "DiskFull")}) //nolint:errcheck
	})
	srv2 := httptest.NewServer(mux2)
	defer srv2.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", srv1.URL},
		{"prod-us", srv2.URL},
	})

	silences, errs := pool.GetAggregatedSilences(context.Background(), alertmanager.SilenceQueryParams{})
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(silences) != 3 {
		t.Fatalf("expected 3 silences, got %d", len(silences))
	}
	for _, s := range silences {
		if s.Alertmanager == "" {
			t.Errorf("silence %s has empty Alertmanager field", s.ID)
		}
	}
}

// TestGetAggregatedSilences_PartialFailure verifies degraded mode for silences.
func TestGetAggregatedSilences_PartialFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/silences", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		now := time.Now()
		json.NewEncoder(w).Encode([]alertmanager.Silence{{ //nolint:errcheck
			ID:        "s1",
			Matchers:  []alertmanager.Matcher{{Name: "alertname", Value: "CPUHigh", IsEqual: true}},
			StartsAt:  now.Add(-time.Hour),
			EndsAt:    now.Add(time.Hour),
			CreatedBy: "alice",
			Comment:   "test silence",
			Status:    alertmanager.SilenceStatus{State: "active"},
		}})
	})
	healthy := httptest.NewServer(mux)
	defer healthy.Close()

	broken := failingServer(t)
	defer broken.Close()

	pool := buildPool(t, []struct{ name, url string }{
		{"prod-eu", healthy.URL},
		{"prod-us", broken.URL},
	})

	silences, errs := pool.GetAggregatedSilences(context.Background(), alertmanager.SilenceQueryParams{})
	if len(silences) != 1 {
		t.Fatalf("expected 1 silence from healthy instance, got %d", len(silences))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 instance error, got %d", len(errs))
	}
}
