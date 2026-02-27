package configbuilder

import (
	"fmt"

	amconfig "github.com/prometheus/alertmanager/config"
	"gopkg.in/yaml.v3"
)

// ValidationResult holds the outcome of a config validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
	Errors   []string `json:"errors,omitempty"`
}

// Validate parses and validates an Alertmanager config YAML using the
// official Prometheus Alertmanager package.
func Validate(rawYAML []byte) ValidationResult {
	cfg, err := amconfig.Load(string(rawYAML))
	if err != nil {
		return ValidationResult{
			Valid:  false,
			Errors: []string{err.Error()},
		}
	}

	var warnings []string
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
