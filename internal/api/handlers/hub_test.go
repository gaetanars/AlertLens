package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"github.com/alertlens/alertlens/internal/config"
	"go.uber.org/zap"
)

// ─── helpers (local to this file) ─────────────────────────────────────────────

func hubTestAlert(name, sev string) alertmanager.Alert {
	return alertmanager.Alert{
		Fingerprint:  name + sev,
		Labels:       map[string]string{"alertname": name, "severity": sev},
		Annotations:  map[string]string{},
		StartsAt:     time.Now().Add(-5 * time.Minute),
		EndsAt:       time.Now().Add(5 * time.Minute),
		Status:       alertmanager.AlertStatus{State: "active"},
		Receivers:    []alertmanager.Receiver{{Name: "default"}},
	}
}

// alertmanagerHubServer starts a fake AM that serves alerts + empty silences + version status.
func alertmanagerHubServer(t *testing.T, alerts []alertmanager.Alert, version string) *httptest.Server {
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
	mux.HandleFunc("/api/v2/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := alertmanager.AMStatus{
			VersionInfo: alertmanager.VersionInfo{Version: version},
			Cluster:     alertmanager.ClusterStatus{Status: "ready"},
		}
		json.NewEncoder(w).Encode(status) //nolint:errcheck
	})
	return httptest.NewServer(mux)
}

func buildHubPool(t *testing.T, instances []struct{ name, url string }) *alertmanager.Pool {
	t.Helper()
	cfgs := make([]config.AlertmanagerConfig, len(instances))
	for i, inst := range instances {
		cfgs[i] = config.AlertmanagerConfig{Name: inst.name, URL: inst.url}
	}
	logger, _ := zap.NewDevelopment()
	return alertmanager.NewPool(cfgs, logger, "test")
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestHubTopology_SingleInstance verifies the topology endpoint returns correct
// hub stats and spoke data for a single healthy Alertmanager instance.
func TestHubTopology_SingleInstance(t *testing.T) {
	srv := alertmanagerHubServer(t, []alertmanager.Alert{
		hubTestAlert("CPUHigh", "critical"),
		hubTestAlert("MemHigh", "warning"),
	}, "0.26.0")
	defer srv.Close()

	pool := buildHubPool(t, []struct{ name, url string }{{"prod", srv.URL}})
	logger, _ := zap.NewDevelopment()
	h := NewHubHandler(pool, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/hub/topology", nil)
	rr := httptest.NewRecorder()
	h.Topology(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var topo HubTopology
	if err := json.Unmarshal(rr.Body.Bytes(), &topo); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Hub assertions
	if topo.Hub.TotalInstances != 1 {
		t.Errorf("expected 1 total instance, got %d", topo.Hub.TotalInstances)
	}
	if topo.Hub.HealthyInstances != 1 {
		t.Errorf("expected 1 healthy instance, got %d", topo.Hub.HealthyInstances)
	}
	if topo.Hub.TotalAlerts != 2 {
		t.Errorf("expected 2 total alerts, got %d", topo.Hub.TotalAlerts)
	}
	if topo.Hub.CriticalAlerts != 1 {
		t.Errorf("expected 1 critical alert, got %d", topo.Hub.CriticalAlerts)
	}

	// Spoke assertions
	if len(topo.Spokes) != 1 {
		t.Fatalf("expected 1 spoke, got %d", len(topo.Spokes))
	}
	spoke := topo.Spokes[0]
	if spoke.Name != "prod" {
		t.Errorf("expected spoke name 'prod', got %q", spoke.Name)
	}
	if !spoke.Healthy {
		t.Error("expected spoke to be healthy")
	}
	if spoke.AlertCount != 2 {
		t.Errorf("expected 2 alerts on spoke, got %d", spoke.AlertCount)
	}
	if spoke.SeverityCounts["critical"] != 1 {
		t.Errorf("expected 1 critical, got %d", spoke.SeverityCounts["critical"])
	}
	if spoke.SeverityCounts["warning"] != 1 {
		t.Errorf("expected 1 warning, got %d", spoke.SeverityCounts["warning"])
	}
}

// TestHubTopology_PartialFailure verifies degraded-mode: an unhealthy spoke is
// still returned (marked unhealthy) while healthy spokes provide full data.
func TestHubTopology_PartialFailure(t *testing.T) {
	srvOK := alertmanagerHubServer(t, []alertmanager.Alert{
		hubTestAlert("DiskFull", "critical"),
	}, "0.26.0")
	defer srvOK.Close()

	// Broken server: always 500.
	srvBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srvBad.Close()

	pool := buildHubPool(t, []struct{ name, url string }{
		{"prod-eu", srvOK.URL},
		{"prod-us", srvBad.URL},
	})
	logger, _ := zap.NewDevelopment()
	h := NewHubHandler(pool, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/hub/topology", nil)
	rr := httptest.NewRecorder()
	h.Topology(rr, req)

	// Must always return 200 even with partial failure.
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
	}

	var topo HubTopology
	if err := json.Unmarshal(rr.Body.Bytes(), &topo); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if topo.Hub.TotalInstances != 2 {
		t.Errorf("expected 2 total instances, got %d", topo.Hub.TotalInstances)
	}
	if topo.Hub.HealthyInstances != 1 {
		t.Errorf("expected 1 healthy instance, got %d", topo.Hub.HealthyInstances)
	}
	if topo.Hub.TotalAlerts != 1 {
		t.Errorf("expected 1 total alert (from healthy instance only), got %d", topo.Hub.TotalAlerts)
	}

	// Find each spoke by name.
	spokeByName := map[string]SpokeStats{}
	for _, s := range topo.Spokes {
		spokeByName[s.Name] = s
	}

	if !spokeByName["prod-eu"].Healthy {
		t.Error("prod-eu should be healthy")
	}
	if spokeByName["prod-us"].Healthy {
		t.Error("prod-us should be unhealthy (server returns 500)")
	}
	if spokeByName["prod-us"].Error == "" {
		t.Error("expected non-empty error on unhealthy spoke")
	}
}

// TestHubTopology_EmptyPool verifies the response when no instances are configured.
func TestHubTopology_EmptyPool(t *testing.T) {
	pool := buildHubPool(t, nil)
	logger, _ := zap.NewDevelopment()
	h := NewHubHandler(pool, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/hub/topology", nil)
	rr := httptest.NewRecorder()
	h.Topology(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var topo HubTopology
	if err := json.Unmarshal(rr.Body.Bytes(), &topo); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if topo.Hub.TotalInstances != 0 {
		t.Errorf("expected 0 instances, got %d", topo.Hub.TotalInstances)
	}
	if len(topo.Spokes) != 0 {
		t.Errorf("expected empty spokes, got %d", len(topo.Spokes))
	}
}
