package handlers

import (
	"testing"

	"github.com/alertlens/alertlens/internal/alertmanager"
)

// ─── annotateRouteCounts tests ────────────────────────────────────────────────

func makeAlert(labels map[string]string) alertmanager.Alert {
	return alertmanager.Alert{
		Labels: labels,
		Status: alertmanager.AlertStatus{State: "active"},
	}
}

// TestAnnotateRouteCounts_CatchAll verifies that a root node with no matchers
// receives the full alert count.
func TestAnnotateRouteCounts_CatchAll(t *testing.T) {
	node := map[string]any{
		"receiver": "root",
		"matchers": []map[string]any{},
		"routes":   []map[string]any{},
	}
	alerts := []alertmanager.Alert{
		makeAlert(map[string]string{"severity": "critical"}),
		makeAlert(map[string]string{"severity": "warning"}),
	}
	annotateRouteCounts(node, alerts)

	cnt, ok := node["alert_count"].(int)
	if !ok || cnt != 2 {
		t.Errorf("expected alert_count=2 on catch-all root, got %v", node["alert_count"])
	}
}

// TestAnnotateRouteCounts_MatcherFilter verifies that child nodes with matchers
// only count alerts that satisfy those matchers.
func TestAnnotateRouteCounts_MatcherFilter(t *testing.T) {
	critChild := map[string]any{
		"receiver": "critical-receiver",
		"matchers": []map[string]any{
			{"name": "severity", "value": "critical", "is_regex": false, "is_equal": true},
		},
		"routes": []map[string]any{},
	}
	root := map[string]any{
		"receiver": "root",
		"matchers": []map[string]any{},
		"routes":   []map[string]any{critChild},
	}

	alerts := []alertmanager.Alert{
		makeAlert(map[string]string{"severity": "critical", "alertname": "A"}),
		makeAlert(map[string]string{"severity": "critical", "alertname": "B"}),
		makeAlert(map[string]string{"severity": "warning", "alertname": "C"}),
	}
	annotateRouteCounts(root, alerts)

	// Root: all 3
	rootCnt := root["alert_count"].(int)
	if rootCnt != 3 {
		t.Errorf("root: expected 3, got %d", rootCnt)
	}

	// Child: only the 2 critical alerts
	routes := root["routes"].([]map[string]any)
	childCnt := routes[0]["alert_count"].(int)
	if childCnt != 2 {
		t.Errorf("child: expected 2 critical alerts, got %d", childCnt)
	}
}

// TestAnnotateRouteCounts_SeverityCounts verifies that the severity_counts map
// is correctly populated.
func TestAnnotateRouteCounts_SeverityCounts(t *testing.T) {
	node := map[string]any{
		"receiver": "root",
		"matchers": []map[string]any{},
		"routes":   []map[string]any{},
	}
	alerts := []alertmanager.Alert{
		makeAlert(map[string]string{"severity": "critical"}),
		makeAlert(map[string]string{"severity": "critical"}),
		makeAlert(map[string]string{"severity": "warning"}),
		makeAlert(map[string]string{}), // no severity label
	}
	annotateRouteCounts(node, alerts)

	sc := node["severity_counts"].(map[string]int)
	if sc["critical"] != 2 {
		t.Errorf("expected 2 critical, got %d", sc["critical"])
	}
	if sc["warning"] != 1 {
		t.Errorf("expected 1 warning, got %d", sc["warning"])
	}
	// Alert with no severity label should not create a "" entry.
	if _, exists := sc[""]; exists {
		t.Error("empty severity label should not create an entry in severity_counts")
	}
}

// TestAnnotateRouteCounts_NegationMatcher verifies that is_equal=false (!=) works.
func TestAnnotateRouteCounts_NegationMatcher(t *testing.T) {
	node := map[string]any{
		"receiver": "non-critical",
		"matchers": []map[string]any{
			{"name": "severity", "value": "critical", "is_regex": false, "is_equal": false},
		},
		"routes": []map[string]any{},
	}
	alerts := []alertmanager.Alert{
		makeAlert(map[string]string{"severity": "critical"}),
		makeAlert(map[string]string{"severity": "warning"}),
		makeAlert(map[string]string{"severity": "info"}),
	}
	annotateRouteCounts(node, alerts)

	cnt := node["alert_count"].(int)
	if cnt != 2 {
		t.Errorf("expected 2 non-critical alerts, got %d", cnt)
	}
}

// TestAnnotateRouteCounts_NoAlerts verifies that zero-alert nodes are annotated.
func TestAnnotateRouteCounts_NoAlerts(t *testing.T) {
	node := map[string]any{
		"receiver": "root",
		"matchers": []map[string]any{},
		"routes":   []map[string]any{},
	}
	annotateRouteCounts(node, nil)

	cnt := node["alert_count"].(int)
	if cnt != 0 {
		t.Errorf("expected alert_count=0 for no alerts, got %d", cnt)
	}
}
