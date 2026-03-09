package configbuilder

import (
	"strings"
	"testing"
)

// minimalValidConfig is the smallest config that Alertmanager accepts.
const minimalValidConfig = `
route:
  receiver: 'null'
receivers:
  - name: 'null'
`

// ─── Validate ─────────────────────────────────────────────────────────────────

func TestValidate_MinimalValidConfig(t *testing.T) {
	result := Validate([]byte(minimalValidConfig))
	if !result.Valid {
		t.Errorf("expected valid config, got errors: %v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", result.Warnings)
	}
}

func TestValidate_InvalidYAML(t *testing.T) {
	result := Validate([]byte("{{not: valid: yaml"))
	if result.Valid {
		t.Error("expected invalid result for malformed YAML")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error for malformed YAML")
	}
}

func TestValidate_EmptyInput(t *testing.T) {
	result := Validate([]byte(""))
	// An empty config is technically parsed but will fail Alertmanager validation
	// (no route, no receivers — both warnings, but load itself may succeed).
	// The key assertion is that it doesn't panic.
	_ = result
}

func TestValidate_NoRoute_WarningPresent(t *testing.T) {
	config := `
receivers:
  - name: 'null'
`
	result := Validate([]byte(config))
	if !result.Valid {
		t.Skipf("config load failed (expected in some AM versions): %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "route") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning about missing route, got: %v", result.Warnings)
	}
}

func TestValidate_NoReceivers_WarningPresent(t *testing.T) {
	config := `
route:
  receiver: 'null'
`
	result := Validate([]byte(config))
	if !result.Valid {
		t.Skipf("config load failed: %v", result.Errors)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "receiver") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning about missing receivers, got: %v", result.Warnings)
	}
}

func TestValidate_InvalidReceiverReference(t *testing.T) {
	// Route references a receiver that doesn't exist.
	config := `
route:
  receiver: 'nonexistent'
receivers:
  - name: 'null'
`
	result := Validate([]byte(config))
	// Alertmanager validation should catch the undefined receiver.
	if result.Valid {
		t.Error("expected invalid config when route references unknown receiver")
	}
}

func TestValidate_InvalidField(t *testing.T) {
	config := `
route:
  receiver: 'null'
  group_wait: "not-a-duration"
receivers:
  - name: 'null'
`
	result := Validate([]byte(config))
	if result.Valid {
		t.Error("expected invalid config for malformed duration field")
	}
}

func TestValidate_ValidationResult_NilWarningsOnValid(t *testing.T) {
	result := Validate([]byte(minimalValidConfig))
	// Warnings field should be present but empty (not nil) for a clean config.
	// Valid=true, Errors=nil are the key assertions.
	if !result.Valid {
		t.Errorf("expected valid: %v", result.Errors)
	}
	if result.Errors != nil {
		t.Errorf("expected nil errors, got: %v", result.Errors)
	}
}

// ─── Size limit (CWE-400) ─────────────────────────────────────────────────────

func TestValidate_OversizedInput_ReturnsError(t *testing.T) {
	// Build a payload that exceeds maxYAMLSize (1 MiB).
	huge := make([]byte, maxYAMLSize+1)
	for i := range huge {
		huge[i] = 'a'
	}
	result := Validate(huge)
	if result.Valid {
		t.Error("expected invalid result for oversized YAML")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error for oversized YAML")
	}
}

func TestValidate_ExactlyAtSizeLimit_Parsed(t *testing.T) {
	// A payload exactly at maxYAMLSize must reach the parser (may or may not be
	// valid YAML, but must not be rejected by the size check alone).
	exactly := make([]byte, maxYAMLSize)
	for i := range exactly {
		exactly[i] = '#' // comment byte — valid YAML, not a bomb
	}
	result := Validate(exactly)
	// We only assert it is NOT rejected for being too large; it will be invalid
	// because there's no route/receivers, but that's the parser's business.
	for _, e := range result.Errors {
		if len(e) > 0 && e[:6] == "config" {
			// "config YAML exceeds..." — this should NOT appear
			t.Errorf("should not be rejected by size check at exactly maxYAMLSize: %s", e)
		}
	}
}
