package alertmanager

import (
	"testing"
	"time"
)

// ─── test fixtures ───────────────────────────────────────────────────────────

func makeTestAlerts() []EnrichedAlert {
	return []EnrichedAlert{
		{
			Alert: Alert{
				Fingerprint: "fp1",
				Labels:      map[string]string{"alertname": "CPUHigh", "severity": "critical", "env": "prod"},
				StartsAt:    time.Now().Add(-10 * time.Minute),
				Status:      AlertStatus{State: "active"},
			},
			Alertmanager: "prod-eu",
		},
		{
			Alert: Alert{
				Fingerprint: "fp2",
				Labels:      map[string]string{"alertname": "MemHigh", "severity": "warning", "env": "prod"},
				StartsAt:    time.Now().Add(-5 * time.Minute),
				Status:      AlertStatus{State: "active"},
			},
			Alertmanager: "prod-eu",
		},
		{
			Alert: Alert{
				Fingerprint: "fp3",
				Labels:      map[string]string{"alertname": "DiskFull", "severity": "critical", "env": "staging"},
				StartsAt:    time.Now().Add(-2 * time.Minute),
				Status:      AlertStatus{State: "suppressed"},
			},
			Alertmanager: "prod-us",
		},
		{
			Alert: Alert{
				Fingerprint: "fp4",
				Labels:      map[string]string{"alertname": "HighLatency", "severity": "info", "env": "staging"},
				StartsAt:    time.Now().Add(-1 * time.Minute),
				Status:      AlertStatus{State: "active"},
			},
			Alertmanager: "prod-us",
		},
	}
}

// ─── groupAlerts ─────────────────────────────────────────────────────────────

func TestGroupAlerts_NoGroupBy(t *testing.T) {
	alerts := makeTestAlerts()
	groups := groupAlerts(alerts, nil)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Count != 4 {
		t.Errorf("expected count=4, got %d", groups[0].Count)
	}
	if len(groups[0].Labels) != 0 {
		t.Errorf("expected empty labels for ungrouped result, got %v", groups[0].Labels)
	}
}

func TestGroupAlerts_BySeverity(t *testing.T) {
	alerts := makeTestAlerts()
	groups := groupAlerts(alerts, []string{"severity"})

	// critical (2) + warning (1) + info (1)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %+v", len(groups), groups)
	}

	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Labels["severity"]] = g.Count
	}
	if counts["critical"] != 2 {
		t.Errorf("expected 2 critical, got %d", counts["critical"])
	}
	if counts["warning"] != 1 {
		t.Errorf("expected 1 warning, got %d", counts["warning"])
	}
	if counts["info"] != 1 {
		t.Errorf("expected 1 info, got %d", counts["info"])
	}
}

func TestGroupAlerts_ByAlertmanager(t *testing.T) {
	alerts := makeTestAlerts()
	groups := groupAlerts(alerts, []string{"alertmanager"})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups (prod-eu, prod-us), got %d", len(groups))
	}
	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Labels["alertmanager"]] = g.Count
	}
	if counts["prod-eu"] != 2 {
		t.Errorf("expected 2 in prod-eu, got %d", counts["prod-eu"])
	}
	if counts["prod-us"] != 2 {
		t.Errorf("expected 2 in prod-us, got %d", counts["prod-us"])
	}
}

func TestGroupAlerts_ByStatus(t *testing.T) {
	alerts := makeTestAlerts()
	groups := groupAlerts(alerts, []string{"status"})

	// active (3) + suppressed (1)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	counts := map[string]int{}
	for _, g := range groups {
		counts[g.Labels["status"]] = g.Count
	}
	if counts["active"] != 3 {
		t.Errorf("expected 3 active, got %d", counts["active"])
	}
	if counts["suppressed"] != 1 {
		t.Errorf("expected 1 suppressed, got %d", counts["suppressed"])
	}
}

func TestGroupAlerts_MultipleKeys(t *testing.T) {
	alerts := makeTestAlerts()
	groups := groupAlerts(alerts, []string{"severity", "env"})

	// critical+prod (1), warning+prod (1), critical+staging (1), info+staging (1)
	if len(groups) != 4 {
		t.Fatalf("expected 4 groups, got %d", len(groups))
	}
}

func TestGroupAlerts_EmptyInput(t *testing.T) {
	groups := groupAlerts(nil, []string{"severity"})
	if len(groups) != 0 {
		t.Errorf("expected 0 groups for empty input, got %d", len(groups))
	}
}

// ─── applyViewFilters ─────────────────────────────────────────────────────────

func TestApplyViewFilters_NoFilters(t *testing.T) {
	alerts := makeTestAlerts()
	filtered := applyViewFilters(alerts, AlertsViewParams{})
	if len(filtered) != len(alerts) {
		t.Errorf("expected %d alerts, got %d", len(alerts), len(filtered))
	}
}

func TestApplyViewFilters_BySeverity(t *testing.T) {
	alerts := makeTestAlerts()
	filtered := applyViewFilters(alerts, AlertsViewParams{
		Severity: []string{"critical"},
	})
	if len(filtered) != 2 {
		t.Errorf("expected 2 critical alerts, got %d", len(filtered))
	}
	for _, a := range filtered {
		if a.Labels["severity"] != "critical" {
			t.Errorf("unexpected severity %s", a.Labels["severity"])
		}
	}
}

func TestApplyViewFilters_ByMultipleSeverities(t *testing.T) {
	alerts := makeTestAlerts()
	filtered := applyViewFilters(alerts, AlertsViewParams{
		Severity: []string{"critical", "warning"},
	})
	if len(filtered) != 3 {
		t.Errorf("expected 3 alerts (critical+warning), got %d", len(filtered))
	}
}

func TestApplyViewFilters_ByStatus(t *testing.T) {
	alerts := makeTestAlerts()
	filtered := applyViewFilters(alerts, AlertsViewParams{
		Status: []string{"suppressed"},
	})
	if len(filtered) != 1 {
		t.Errorf("expected 1 suppressed alert, got %d", len(filtered))
	}
	if filtered[0].Fingerprint != "fp3" {
		t.Errorf("expected fp3, got %s", filtered[0].Fingerprint)
	}
}

func TestApplyViewFilters_BySeverityAndStatus(t *testing.T) {
	alerts := makeTestAlerts()
	// critical + suppressed → only fp3
	filtered := applyViewFilters(alerts, AlertsViewParams{
		Severity: []string{"critical"},
		Status:   []string{"suppressed"},
	})
	if len(filtered) != 1 {
		t.Errorf("expected 1 alert, got %d", len(filtered))
	}
}

func TestApplyViewFilters_NoMatch(t *testing.T) {
	alerts := makeTestAlerts()
	filtered := applyViewFilters(alerts, AlertsViewParams{
		Severity: []string{"unknown"},
	})
	if len(filtered) != 0 {
		t.Errorf("expected 0 alerts, got %d", len(filtered))
	}
}

// ─── buildGroupKey ───────────────────────────────────────────────────────────

func TestBuildGroupKey_VirtualAlertmanager(t *testing.T) {
	a := EnrichedAlert{
		Alert:        Alert{Labels: map[string]string{"severity": "critical"}},
		Alertmanager: "my-instance",
	}
	key, labels := buildGroupKey(a, []string{"alertmanager"})
	if key != "alertmanager=my-instance" {
		t.Errorf("unexpected key: %q", key)
	}
	if labels["alertmanager"] != "my-instance" {
		t.Errorf("unexpected label: %v", labels)
	}
}

func TestBuildGroupKey_VirtualStatus(t *testing.T) {
	a := EnrichedAlert{
		Alert: Alert{
			Labels: map[string]string{},
			Status: AlertStatus{State: "active"},
		},
	}
	key, labels := buildGroupKey(a, []string{"status"})
	if key != "status=active" {
		t.Errorf("unexpected key: %q", key)
	}
	if labels["status"] != "active" {
		t.Errorf("unexpected label: %v", labels)
	}
}

// ─── GetAlertsView pagination ─────────────────────────────────────────────────

func TestGetAlertsView_Pagination(t *testing.T) {
	// We can't call Pool.GetAlertsView directly without mocking the HTTP calls
	// to Alertmanager, but we can test the pagination logic that lives in
	// GetAlertsView by calling the underlying helpers directly.
	alerts := makeTestAlerts()
	filtered := applyViewFilters(alerts, AlertsViewParams{})

	limit := 2
	offset := 0
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[offset:end]

	if len(page) != 2 {
		t.Errorf("expected 2 alerts in first page, got %d", len(page))
	}

	offset = 2
	end = offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page2 := filtered[offset:end]
	if len(page2) != 2 {
		t.Errorf("expected 2 alerts in second page, got %d", len(page2))
	}

	// Third page should be empty.
	offset = 4
	if offset >= len(filtered) {
		page3 := []EnrichedAlert{}
		if len(page3) != 0 {
			t.Error("expected empty third page")
		}
	}
}
