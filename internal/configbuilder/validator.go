package configbuilder

import (
	"fmt"

	amconfig "github.com/prometheus/alertmanager/config"
	"gopkg.in/yaml.v3"
)

// maxYAMLSize is the hard upper bound for Alertmanager configuration YAML
// submitted for validation or saving (1 MiB).
//
// Real-world Alertmanager configs are well under 100 KiB.  Enforcing a size
// limit before calling the YAML parser prevents resource-exhaustion attacks
// ("YAML bombs") that exploit deeply nested structures or anchor/alias
// expansion to consume disproportionate CPU or memory.
//
// SEC-CWE-400: reject oversized input early, before any parsing work.
const maxYAMLSize = 1 << 20 // 1 MiB

// ValidationResult holds the outcome of a config validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors,omitempty"`
}

// Validate parses and validates an Alertmanager config YAML using the
// official Prometheus Alertmanager package.
func Validate(rawYAML []byte) ValidationResult {
	// SEC-CWE-400: reject oversized payloads before involving the YAML parser.
	if len(rawYAML) > maxYAMLSize {
		return ValidationResult{
			Valid: false,
			Errors: []string{fmt.Sprintf(
				"config YAML exceeds maximum allowed size (%d bytes); got %d bytes",
				maxYAMLSize, len(rawYAML),
			)},
		}
	}

	cfg, err := amconfig.Load(string(rawYAML))
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}
	}

	// Use a non-nil slice so JSON always serialises as [] rather than null.
	warnings := make([]string, 0)
	if cfg.Route == nil {
		warnings = append(warnings, "no route defined")
	}
	if len(cfg.Receivers) == 0 {
		warnings = append(warnings, "no receivers defined")
	}

	return ValidationResult{
		Valid:    true,
		Warnings: warnings,
	}
}

// MarshalToYAML converts a parsed config back to YAML.
func MarshalToYAML(cfg *amconfig.Config) ([]byte, error) {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	return b, nil
}
