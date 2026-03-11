package configbuilder

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ConfigBuilder provides structured, YAML-level manipulation of an Alertmanager
// configuration.  It operates on a raw map representation so that unknown fields
// (future Alertmanager additions, inline comments stripped by the YAML parser)
// are preserved round-trip.
//
// Typical workflow:
//
//	b, err := NewConfigBuilder(rawYAML)
//	b.UpsertTimeInterval(entry)
//	b.UpsertReceiver(receiver)
//	b.SetRoute(route)
//	newYAML, err := b.Build()   // validates before returning
type ConfigBuilder struct {
	raw map[string]interface{}
}

// NewConfigBuilder parses rawYAML and returns a ConfigBuilder ready for
// manipulation.  An empty or nil slice creates a builder with an empty config.
func NewConfigBuilder(rawYAML []byte) (*ConfigBuilder, error) {
	m := make(map[string]interface{})
	if len(rawYAML) > 0 {
		if err := yaml.Unmarshal(rawYAML, &m); err != nil {
			return nil, fmt.Errorf("parsing config YAML: %w", err)
		}
	}
	return &ConfigBuilder{raw: m}, nil
}

// ─── YAML output ─────────────────────────────────────────────────────────────

// Build serializes the current configuration to YAML and validates it with
// the official Alertmanager library.  Returns an error if validation fails.
func (b *ConfigBuilder) Build() ([]byte, error) {
	out, err := yaml.Marshal(b.raw)
	if err != nil {
		return nil, fmt.Errorf("serializing config: %w", err)
	}
	result := Validate(out)
	if !result.Valid {
		return nil, fmt.Errorf("config validation failed: %v", result.Errors)
	}
	return out, nil
}

// BuildRaw serializes the current state to YAML without running validation.
// Useful when the config is in a transitional (incomplete) state.
func (b *ConfigBuilder) BuildRaw() ([]byte, error) {
	out, err := yaml.Marshal(b.raw)
	if err != nil {
		return nil, fmt.Errorf("serializing config: %w", err)
	}
	return out, nil
}

// ─── Time Intervals ──────────────────────────────────────────────────────────

// ListTimeIntervals returns the current time_intervals section of the config.
// Returns an empty (non-nil) slice when no time intervals are defined.
func (b *ConfigBuilder) ListTimeIntervals() ([]TimeIntervalEntry, error) {
	return b.parseTimeIntervals("time_intervals")
}

// UpsertTimeInterval adds or updates a named time interval.
// If an entry with the same name already exists it is replaced in-place;
// otherwise the new entry is appended.
func (b *ConfigBuilder) UpsertTimeInterval(entry TimeIntervalEntry) error {
	if entry.Name == "" {
		return fmt.Errorf("time interval name must not be empty")
	}
	entries, err := b.parseTimeIntervals("time_intervals")
	if err != nil {
		return err
	}

	found := false
	for i, e := range entries {
		if e.Name == entry.Name {
			entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		entries = append(entries, entry)
	}
	b.raw["time_intervals"] = entries
	return nil
}

// DeleteTimeInterval removes the named time interval.
// Returns true if the entry existed and was removed, false if not found.
func (b *ConfigBuilder) DeleteTimeInterval(name string) (bool, error) {
	entries, err := b.parseTimeIntervals("time_intervals")
	if err != nil {
		return false, err
	}

	filtered := make([]TimeIntervalEntry, 0, len(entries))
	found := false
	for _, e := range entries {
		if e.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if found {
		b.raw["time_intervals"] = filtered
	}
	return found, nil
}

// ─── Receivers ───────────────────────────────────────────────────────────────

// ListReceivers returns all receivers currently defined in the config.
func (b *ConfigBuilder) ListReceivers() ([]ReceiverDef, error) {
	return b.parseReceivers()
}

// UpsertReceiver adds or updates a receiver.
// If a receiver with the same name already exists it is replaced in-place;
// otherwise it is appended.
func (b *ConfigBuilder) UpsertReceiver(rec ReceiverDef) error {
	if rec.Name == "" {
		return fmt.Errorf("receiver name must not be empty")
	}
	receivers, err := b.parseReceivers()
	if err != nil {
		return err
	}

	found := false
	for i, r := range receivers {
		if r.Name == rec.Name {
			receivers[i] = rec
			found = true
			break
		}
	}
	if !found {
		receivers = append(receivers, rec)
	}
	b.raw["receivers"] = receivers
	return nil
}

// DeleteReceiver removes the named receiver.
// Returns true if it was found and removed.
func (b *ConfigBuilder) DeleteReceiver(name string) (bool, error) {
	receivers, err := b.parseReceivers()
	if err != nil {
		return false, err
	}

	filtered := make([]ReceiverDef, 0, len(receivers))
	found := false
	for _, r := range receivers {
		if r.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, r)
	}
	if found {
		b.raw["receivers"] = filtered
	}
	return found, nil
}

// ─── Route ───────────────────────────────────────────────────────────────────

// GetRoute returns the current root route or nil if none is defined.
func (b *ConfigBuilder) GetRoute() (*RouteSpec, error) {
	v, ok := b.raw["route"]
	if !ok || v == nil {
		return nil, nil
	}
	// Re-serialize to YAML then re-parse into typed struct to handle any map
	// representation left over from the initial parse.
	sect, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serializing route: %w", err)
	}
	var route RouteSpec
	if err := yaml.Unmarshal(sect, &route); err != nil {
		return nil, fmt.Errorf("parsing route: %w", err)
	}
	return &route, nil
}

// SetRoute replaces the root route entirely.
func (b *ConfigBuilder) SetRoute(route RouteSpec) {
	b.raw["route"] = route
}

// ─── Validation helpers ───────────────────────────────────────────────────────

// ValidateTimeInterval embeds a single TimeIntervalEntry inside a minimal
// valid config and validates it with the official Alertmanager library.
// This lets callers verify a single entry before upserting it.
func ValidateTimeInterval(entry TimeIntervalEntry) ValidationResult {
	tmp := map[string]interface{}{
		"route":          map[string]interface{}{"receiver": "null"},
		"receivers":      []interface{}{map[string]interface{}{"name": "null"}},
		"time_intervals": []TimeIntervalEntry{entry},
	}
	out, err := yaml.Marshal(tmp)
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{"serialization error: " + err.Error()}}
	}
	return Validate(out)
}

// ValidateReceiver embeds a single ReceiverDef in a minimal valid config and
// validates it with the official Alertmanager library.
func ValidateReceiver(rec ReceiverDef) ValidationResult {
	tmp := map[string]interface{}{
		"route":     map[string]interface{}{"receiver": rec.Name},
		"receivers": []ReceiverDef{rec},
	}
	out, err := yaml.Marshal(tmp)
	if err != nil {
		return ValidationResult{Valid: false, Errors: []string{"serialization error: " + err.Error()}}
	}
	return Validate(out)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// parseTimeIntervals round-trips the named section through YAML to get a typed
// []TimeIntervalEntry slice, handling both the "already typed" case (when we've
// set it ourselves) and the "raw map" case (when it came from a YAML parse).
func (b *ConfigBuilder) parseTimeIntervals(key string) ([]TimeIntervalEntry, error) {
	v, ok := b.raw[key]
	if !ok || v == nil {
		return []TimeIntervalEntry{}, nil
	}
	sect, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", key, err)
	}
	var entries []TimeIntervalEntry
	if err := yaml.Unmarshal(sect, &entries); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", key, err)
	}
	if entries == nil {
		return []TimeIntervalEntry{}, nil
	}
	return entries, nil
}

// parseReceivers round-trips the receivers section through YAML.
func (b *ConfigBuilder) parseReceivers() ([]ReceiverDef, error) {
	v, ok := b.raw["receivers"]
	if !ok || v == nil {
		return []ReceiverDef{}, nil
	}
	sect, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serializing receivers: %w", err)
	}
	var recs []ReceiverDef
	if err := yaml.Unmarshal(sect, &recs); err != nil {
		return nil, fmt.Errorf("parsing receivers: %w", err)
	}
	if recs == nil {
		return []ReceiverDef{}, nil
	}
	return recs, nil
}
