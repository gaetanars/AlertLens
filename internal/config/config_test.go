package config

import (
	"os"
	"strings"
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
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
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

// ─── Password validation tests ───────────────────────────────────────────────

func TestValidate_AdminPassword_Over72Bytes(t *testing.T) {
	// A password with 73 bytes should be rejected.
	cfg := defaults()
	cfg.Auth.AdminPassword = strings.Repeat("x", 73)

	err := validate(&cfg)
	if err == nil {
		t.Error("expected validation error for admin password > 72 bytes")
	}
	if !strings.Contains(err.Error(), "72-byte") && !strings.Contains(err.Error(), "72 byte") {
		t.Errorf("error message should mention 72-byte limit, got: %v", err)
	}
	if !strings.Contains(err.Error(), "73 bytes") {
		t.Errorf("error message should show actual byte count (73), got: %v", err)
	}
}

func TestValidate_AdminPassword_Exactly72Bytes(t *testing.T) {
	// Exactly 72 bytes should be accepted.
	cfg := defaults()
	cfg.Auth.AdminPassword = strings.Repeat("a", 72)

	err := validate(&cfg)
	if err != nil {
		t.Errorf("expected no error for 72-byte password, got: %v", err)
	}
}

func TestValidate_AdminPassword_Under72Bytes(t *testing.T) {
	// Under 72 bytes should be accepted.
	cfg := defaults()
	cfg.Auth.AdminPassword = "short-password"

	err := validate(&cfg)
	if err != nil {
		t.Errorf("expected no error for short password, got: %v", err)
	}
}

func TestValidate_UserPassword_Over72Bytes(t *testing.T) {
	// A user password with > 72 bytes should be rejected.
	cfg := defaults()
	cfg.Auth.Users = []UserConfig{
		{
			Password: strings.Repeat("y", 80),
			Role:     "viewer",
		},
	}

	err := validate(&cfg)
	if err == nil {
		t.Error("expected validation error for user password > 72 bytes")
	}
	if !strings.Contains(err.Error(), "users[0]") {
		t.Errorf("error should identify which user failed, got: %v", err)
	}
	if !strings.Contains(err.Error(), "72-byte") && !strings.Contains(err.Error(), "72 byte") {
		t.Errorf("error message should mention 72-byte limit, got: %v", err)
	}
}

func TestValidate_UserPassword_Empty(t *testing.T) {
	// An empty user password should be rejected.
	cfg := defaults()
	cfg.Auth.Users = []UserConfig{
		{
			Password: "",
			Role:     "viewer",
		},
	}

	err := validate(&cfg)
	if err == nil {
		t.Error("expected validation error for empty user password")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("error message should mention empty password, got: %v", err)
	}
}

// ─── TenantID validation tests ───────────────────────────────────────────────

func TestValidateTenantID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tenantID  string
		wantError bool
	}{
		// Valid tenant IDs
		{name: "alphanumeric lowercase", tenantID: "payments", wantError: false},
		{name: "hyphen separator", tenantID: "infra-team", wantError: false},
		{name: "underscore separator", tenantID: "tenant_1", wantError: false},
		{name: "uppercase letters", tenantID: "ABC123", wantError: false},
		// Invalid tenant IDs
		{name: "space in id", tenantID: "tenant id", wantError: true},
		{name: "at-sign", tenantID: "tenant@1", wantError: true},
		{name: "path traversal", tenantID: "../evil", wantError: true},
		{name: "exclamation mark", tenantID: "tenant!1", wantError: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Build a minimal valid config YAML with the tenant_id under test.
			content := "alertmanagers:\n  - name: test\n    url: http://am:9093\n    tenant_id: " + tc.tenantID + "\n"

			f, err := os.CreateTemp("", "alertlens-tenantid-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(f.Name())
			if _, err := f.WriteString(content); err != nil {
				t.Fatal(err)
			}
			f.Close()

			_, _, loadErr := Load(f.Name())
			if tc.wantError {
				if loadErr == nil {
					t.Errorf("expected validation error for tenant_id=%q, got nil", tc.tenantID)
					return
				}
				if !strings.Contains(loadErr.Error(), "tenant_id") {
					t.Errorf("error should mention tenant_id, got: %v", loadErr)
				}
			} else {
				if loadErr != nil {
					t.Errorf("expected no error for tenant_id=%q, got: %v", tc.tenantID, loadErr)
				}
			}
		})
	}
}

// ─── Role validation tests ────────────────────────────────────────────────────

func TestValidate_UserRole(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		role      string
		wantError bool
	}{
		// Valid roles — all four must be accepted.
		{name: "viewer", role: "viewer", wantError: false},
		{name: "silencer", role: "silencer", wantError: false},
		{name: "config-editor", role: "config-editor", wantError: false},
		{name: "admin", role: "admin", wantError: false},
		// Invalid roles — must be rejected with a descriptive error.
		{name: "empty string", role: "", wantError: true},
		{name: "read-only", role: "read-only", wantError: true},
		{name: "superuser", role: "superuser", wantError: true},
		{name: "ADMIN uppercase", role: "ADMIN", wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaults()
			// Use a non-empty password so the password checks pass before role is reached.
			// Empty-role test still triggers a role error, not a password error.
			pass := "validpassword"
			if tc.role == "" {
				// Empty role: password check passes, role check must fail.
				pass = "validpassword"
			}
			cfg.Auth.Users = []UserConfig{{Password: pass, Role: tc.role}}

			err := validate(&cfg)
			if tc.wantError {
				if err == nil {
					t.Errorf("expected validation error for role %q, got nil", tc.role)
					return
				}
				if !strings.Contains(err.Error(), "role") {
					t.Errorf("error should mention role, got: %v", err)
				}
				if !strings.Contains(err.Error(), "users[0]") {
					t.Errorf("error should identify which user failed, got: %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for valid role %q, got: %v", tc.role, err)
				}
			}
		})
	}
}

// ─── Duplicate password warning tests ────────────────────────────────────────

func TestLoad_DuplicatePasswordWarning(t *testing.T) {
	t.Parallel()

	t.Run("two users with same password", func(t *testing.T) {
		t.Parallel()

		cfg := defaults()
		cfg.Auth.Users = []UserConfig{
			{Password: "shared-pass", Role: "viewer"},
			{Password: "shared-pass", Role: "silencer"},
		}
		warnings := checkDuplicatePasswords(&cfg)
		if len(warnings) == 0 {
			t.Error("expected a duplicate password warning, got none")
		}
	})

	t.Run("admin password duplicated by a user", func(t *testing.T) {
		t.Parallel()

		cfg := defaults()
		cfg.Auth.AdminPassword = "same-pass"
		cfg.Auth.Users = []UserConfig{
			{Password: "same-pass", Role: "viewer"},
		}
		warnings := checkDuplicatePasswords(&cfg)
		if len(warnings) == 0 {
			t.Error("expected a warning when admin_password equals a user password, got none")
		}
	})

	t.Run("all distinct passwords produces no warning", func(t *testing.T) {
		t.Parallel()

		cfg := defaults()
		cfg.Auth.AdminPassword = "admin-pass"
		cfg.Auth.Users = []UserConfig{
			{Password: "viewer-pass", Role: "viewer"},
			{Password: "silencer-pass", Role: "silencer"},
		}
		warnings := checkDuplicatePasswords(&cfg)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings for distinct passwords, got: %v", warnings)
		}
	})
}

func TestValidate_UserPassword_UTF8Counting(t *testing.T) {
	// Verify that byte counting is UTF-8 aware.
	// "café" = 5 bytes (c, a, f, é=2 bytes in UTF-8).
	cfg := defaults()
	// Create a password with exactly 72 bytes using UTF-8 multi-byte chars.
	// 36 instances of "é" (2 bytes each) = 72 bytes.
	cfg.Auth.Users = []UserConfig{
		{
			Password: strings.Repeat("é", 36), // exactly 72 bytes
			Role:     "viewer",
		},
	}

	err := validate(&cfg)
	if err != nil {
		t.Errorf("expected no error for 72-byte UTF-8 password, got: %v", err)
	}

	// Now test 73 bytes (36 'é' + 1 'a' = 72 + 1).
	cfg.Auth.Users[0].Password = strings.Repeat("é", 36) + "a"
	err = validate(&cfg)
	if err == nil {
		t.Error("expected validation error for 73-byte UTF-8 password")
	}
}
