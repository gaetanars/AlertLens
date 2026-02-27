package handlers

import (
	"testing"
)

// configYAML is a minimal but valid Alertmanager config used by routing tests.
const configYAML = `
route:
  receiver: "root"
  routes:
    - matchers:
        - env="prod"
      receiver: "prod-receiver"
      continue: true
    - matchers:
        - env="prod"
      receiver: "prod-receiver-2"
receivers:
  - name: "root"
  - name: "prod-receiver"
  - name: "prod-receiver-2"
`

// configYAMLNoContinue is a config where the first matching child has continue:false (default).
const configYAMLNoContinue = `
route:
  receiver: "root"
  routes:
    - matchers:
        - env="prod"
      receiver: "prod-receiver"
    - matchers:
        - env="prod"
      receiver: "prod-receiver-2"
receivers:
  - name: "root"
  - name: "prod-receiver"
  - name: "prod-receiver-2"
`

// TestMatchRoute_ContinueTrue verifies that when a child route has continue:true,
// subsequent sibling routes are also evaluated.
func TestMatchRoute_ContinueTrue(t *testing.T) {
	cfg, err := parseAMConfig(configYAML)
	if err != nil {
		t.Fatalf("parseAMConfig: %v", err)
	}
	labels := map[string]string{"env": "prod"}
	matched := matchRoute(cfg.Route, labels)

	// Expect: root + first child (continue:true) + second child = 3 routes.
	if len(matched) != 3 {
		t.Errorf("with continue:true, expected 3 matched routes (root + 2 children), got %d", len(matched))
	}
}

// TestMatchRoute_ContinueFalseDefault verifies that when continue is false (default),
// routing stops after the first matching child.
func TestMatchRoute_ContinueFalseDefault(t *testing.T) {
	cfg, err := parseAMConfig(configYAMLNoContinue)
	if err != nil {
		t.Fatalf("parseAMConfig: %v", err)
	}
	labels := map[string]string{"env": "prod"}
	matched := matchRoute(cfg.Route, labels)

	// Expect: root + first child only = 2 routes.
	if len(matched) != 2 {
		t.Errorf("with continue:false (default), expected 2 matched routes (root + 1st child), got %d", len(matched))
	}
}

// TestMatchRoute_NoMatch verifies that unmatched routes return nil.
func TestMatchRoute_NoMatch(t *testing.T) {
	cfg, err := parseAMConfig(configYAML)
	if err != nil {
		t.Fatalf("parseAMConfig: %v", err)
	}
	labels := map[string]string{"env": "staging"}
	matched := matchRoute(cfg.Route, labels)

	// Only the root (which has no matchers) should match, not the children.
	if len(matched) != 1 {
		t.Errorf("expected only root route to match for non-prod labels, got %d routes", len(matched))
	}
}

// TestMatchRoute_NilRoute verifies graceful handling of nil route.
func TestMatchRoute_NilRoute(t *testing.T) {
	result := matchRoute(nil, map[string]string{"env": "prod"})
	if result != nil {
		t.Errorf("expected nil result for nil route, got %v", result)
	}
}

// ─── routeToMap ──────────────────────────────────────────────────────────────

// TestRouteToMap_NilRoute verifies graceful handling.
func TestRouteToMap_NilRoute(t *testing.T) {
	if routeToMap(nil) != nil {
		t.Error("expected nil for nil route")
	}
}

// TestRouteToMap_ContinueField verifies that the continue field is preserved.
func TestRouteToMap_ContinueField(t *testing.T) {
	cfg, err := parseAMConfig(configYAML)
	if err != nil {
		t.Fatalf("parseAMConfig: %v", err)
	}
	// First child has continue:true.
	child := cfg.Route.Routes[0]
	m := routeToMap(child)
	continueVal, ok := m["continue"].(bool)
	if !ok || !continueVal {
		t.Errorf("expected continue:true on first child route, got %v", m["continue"])
	}
}

// ─── validateWebhookURL ───────────────────────────────────────────────────────

func TestValidateWebhookURL_ValidHTTPS(t *testing.T) {
	// Use a public domain; DNS resolution is required but may be skipped in CI.
	// We test at least the URL parsing and scheme check.
	err := validateWebhookURL("https://hooks.example-public.com/notify")
	// We accept either nil (resolved OK) or a DNS-resolution error (no network).
	if err != nil && err.Error() == "webhook_url must use HTTPS" {
		t.Error("should not reject a valid HTTPS URL")
	}
}

func TestValidateWebhookURL_RejectHTTP(t *testing.T) {
	if err := validateWebhookURL("http://hooks.example.com/notify"); err == nil {
		t.Error("expected error for HTTP URL")
	}
}

func TestValidateWebhookURL_RejectLoopbackIP(t *testing.T) {
	if err := validateWebhookURL("https://127.0.0.1/notify"); err == nil {
		t.Error("expected error for loopback IP")
	}
}

func TestValidateWebhookURL_RejectPrivateIP(t *testing.T) {
	if err := validateWebhookURL("https://192.168.1.100/notify"); err == nil {
		t.Error("expected error for private IP")
	}
}

func TestValidateWebhookURL_RejectLocalhost(t *testing.T) {
	if err := validateWebhookURL("https://localhost/notify"); err == nil {
		t.Error("expected error for localhost")
	}
}

func TestValidateWebhookURL_InvalidURL(t *testing.T) {
	if err := validateWebhookURL("not-a-url"); err == nil {
		t.Error("expected error for invalid URL")
	}
}
