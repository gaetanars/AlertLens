package handlers

import (
	"testing"

	"github.com/alertlens/alertlens/internal/alertmanager"
)

// ─── smartMerge ──────────────────────────────────────────────────────────────

func TestSmartMerge_Empty(t *testing.T) {
	result := smartMerge(nil)
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestSmartMerge_SingleAlert(t *testing.T) {
	alerts := []BulkAlertRef{
		{
			Fingerprint:  "abc",
			Alertmanager: "prod",
			Labels:       map[string]string{"alertname": "HighCPU", "severity": "critical"},
		},
	}
	matchers := smartMerge(alerts)
	if len(matchers) != 2 {
		t.Fatalf("expected 2 matchers, got %d", len(matchers))
	}
}

func TestSmartMerge_CommonLabels(t *testing.T) {
	alerts := []BulkAlertRef{
		{Labels: map[string]string{"alertname": "HighCPU", "severity": "critical", "env": "prod"}},
		{Labels: map[string]string{"alertname": "HighCPU", "severity": "critical", "env": "staging"}},
	}
	matchers := smartMerge(alerts)

	// env differs → only alertname + severity should be in the intersection.
	if len(matchers) != 2 {
		t.Fatalf("expected 2 matchers (alertname+severity), got %d: %v", len(matchers), matchers)
	}
	mmap := matchersToMap(matchers)
	if mmap["alertname"] != "HighCPU" {
		t.Errorf("expected alertname=HighCPU, got %q", mmap["alertname"])
	}
	if mmap["severity"] != "critical" {
		t.Errorf("expected severity=critical, got %q", mmap["severity"])
	}
}

func TestSmartMerge_NoCommonLabels(t *testing.T) {
	alerts := []BulkAlertRef{
		{Labels: map[string]string{"alertname": "CPUHigh", "pod": "app-1"}},
		{Labels: map[string]string{"alertname": "MemHigh", "pod": "app-2"}},
	}
	matchers := smartMerge(alerts)
	if matchers != nil {
		t.Fatalf("expected nil (no common labels), got %v", matchers)
	}
}

func TestSmartMerge_AllCommonLabels(t *testing.T) {
	alerts := []BulkAlertRef{
		{Labels: map[string]string{"alertname": "HighCPU", "severity": "critical"}},
		{Labels: map[string]string{"alertname": "HighCPU", "severity": "critical"}},
	}
	matchers := smartMerge(alerts)
	if len(matchers) != 2 {
		t.Fatalf("expected 2 matchers, got %d: %v", len(matchers), matchers)
	}
}

func TestSmartMerge_MetaLabelsExcluded(t *testing.T) {
	alerts := []BulkAlertRef{
		{Labels: map[string]string{"alertname": "HighCPU", "alertlens_ack_type": "visual", "__name__": "ALERTS"}},
		{Labels: map[string]string{"alertname": "HighCPU", "alertlens_ack_type": "visual", "__name__": "ALERTS"}},
	}
	matchers := smartMerge(alerts)
	mmap := matchersToMap(matchers)
	if _, ok := mmap["alertlens_ack_type"]; ok {
		t.Error("alertlens_ack_type should be excluded from matchers")
	}
	if _, ok := mmap["__name__"]; ok {
		t.Error("__name__ should be excluded from matchers")
	}
	if mmap["alertname"] != "HighCPU" {
		t.Errorf("expected alertname=HighCPU, got %q", mmap["alertname"])
	}
}

// ─── filterMetaLabels ────────────────────────────────────────────────────────

func TestFilterMetaLabels(t *testing.T) {
	labels := map[string]string{
		"alertname":          "Test",
		"__name__":           "ALERTS",
		"alertlens_ack_type": "visual",
		"alertlens_ack_by":   "alice",
		"severity":           "warning",
	}
	filtered := filterMetaLabels(labels)
	if _, ok := filtered["__name__"]; ok {
		t.Error("__name__ should be filtered")
	}
	if _, ok := filtered["alertlens_ack_type"]; ok {
		t.Error("alertlens_ack_type should be filtered")
	}
	if _, ok := filtered["alertlens_ack_by"]; ok {
		t.Error("alertlens_ack_by should be filtered")
	}
	if filtered["alertname"] != "Test" {
		t.Errorf("alertname should be preserved, got %q", filtered["alertname"])
	}
	if filtered["severity"] != "warning" {
		t.Errorf("severity should be preserved, got %q", filtered["severity"])
	}
}

// ─── groupAlertsByInstance ───────────────────────────────────────────────────

func TestGroupAlertsByInstance(t *testing.T) {
	alerts := []BulkAlertRef{
		{Fingerprint: "a1", Alertmanager: "prod"},
		{Fingerprint: "a2", Alertmanager: "staging"},
		{Fingerprint: "a3", Alertmanager: "prod"},
	}
	groups := groupAlertsByInstance(alerts)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if len(groups["prod"]) != 2 {
		t.Errorf("expected 2 prod alerts, got %d", len(groups["prod"]))
	}
	if len(groups["staging"]) != 1 {
		t.Errorf("expected 1 staging alert, got %d", len(groups["staging"]))
	}
}

// ─── labelsToMatchers ────────────────────────────────────────────────────────

func TestLabelsToMatchers(t *testing.T) {
	labels := map[string]string{"alertname": "Test", "severity": "critical"}
	matchers := labelsToMatchers(labels)
	if len(matchers) != 2 {
		t.Fatalf("expected 2 matchers, got %d", len(matchers))
	}
	for _, m := range matchers {
		if m.IsRegex {
			t.Error("matcher should not be regex")
		}
		if !m.IsEqual {
			t.Error("matcher should be equality")
		}
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func matchersToMap(matchers []alertmanager.Matcher) map[string]string {
	m := make(map[string]string, len(matchers))
	for _, matcher := range matchers {
		m[matcher.Name] = matcher.Value
	}
	return m
}
