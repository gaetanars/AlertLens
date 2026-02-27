package config

import (
	"os"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg, _, err := Load("")
	if err != nil {
		t.Fatalf("Load with empty path: %v", err)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("default port: got %d, want 9000", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("default host: got %q, want 0.0.0.0", cfg.Server.Host)
	}
	if len(cfg.Alertmanagers) != 1 {
		t.Errorf("default alertmanagers: got %d, want 1", len(cfg.Alertmanagers))
	}
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("ALERTLENS_SERVER_PORT", "8080")
	t.Setenv("ALERTLENS_AUTH_ADMIN_PASSWORD", "secret")
	t.Setenv("ALERTLENS_GITOPS_GITHUB_TOKEN", "gh_token")

	cfg, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("env port: got %d, want 8080", cfg.Server.Port)
	}
	if cfg.Auth.AdminPassword != "secret" {
		t.Errorf("env admin_password: got %q, want secret", cfg.Auth.AdminPassword)
	}
	if cfg.Gitops.GitHub.Token != "gh_token" {
		t.Errorf("env github token: got %q, want gh_token", cfg.Gitops.GitHub.Token)
	}
}

func TestEnvOverrideInvalidInt(t *testing.T) {
	t.Setenv("ALERTLENS_SERVER_PORT", "not-a-number")

	cfg, warnings, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Invalid integer env var should produce a warning and leave the default.
	if cfg.Server.Port != 9000 {
		t.Errorf("invalid int env should keep default port 9000, got %d", cfg.Server.Port)
	}
	if len(warnings) == 0 {
		t.Error("expected at least one warning for invalid integer env var")
	}
}

func TestLoadFile(t *testing.T) {
	content := `
server:
  port: 7777
auth:
  admin_password: "testpass"
alertmanagers:
  - name: "test"
    url: "http://am:9093"
`
	f, err := os.CreateTemp("", "alertlens-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(content)
	f.Close()

	cfg, _, err := Load(f.Name())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("file port: got %d, want 7777", cfg.Server.Port)
	}
	if cfg.Auth.AdminPassword != "testpass" {
		t.Errorf("file admin_password: got %q", cfg.Auth.AdminPassword)
	}
	if len(cfg.Alertmanagers) != 1 || cfg.Alertmanagers[0].Name != "test" {
		t.Errorf("file alertmanagers: got %+v", cfg.Alertmanagers)
	}
}
