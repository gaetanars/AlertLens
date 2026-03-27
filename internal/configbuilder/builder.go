// Package configbuilder provides structured CRUD operations for Alertmanager
// routing rules, receivers, and time intervals, producing raw YAML output.
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
//
// When rec.RawYAML is set the call is delegated to SetReceiverRaw so that
// unknown integration types are preserved verbatim.  Both paths operate on
// the underlying raw-map slice so that sibling receivers with unknown types
// are never inadvertently stripped.
func (b *ConfigBuilder) UpsertReceiver(rec ReceiverDef) error {
	if rec.Name == "" {
		return fmt.Errorf("receiver name must not be empty")
	}
	if rec.RawYAML != "" {
		return b.SetReceiverRaw(rec.Name, rec.RawYAML)
	}

	// Typed path: round-trip rec through YAML to produce a raw map so that
	// unknown receivers elsewhere in the slice are not clobbered.
	recYAML, err := yaml.Marshal(rec)
	if err != nil {
		return fmt.Errorf("serializing receiver: %w", err)
	}
	var recMap map[string]interface{}
	if err := yaml.Unmarshal(recYAML, &recMap); err != nil {
		return fmt.Errorf("parsing receiver as raw map: %w", err)
	}

	raw, err := b.rawReceiversSlice()
	if err != nil {
		return err
	}
	found := false
	for i, r := range raw {
		if n, _ := r["name"].(string); n == rec.Name {
			raw[i] = recMap
			found = true
			break
		}
	}
	if !found {
		raw = append(raw, recMap)
	}
	b.raw["receivers"] = raw
	return nil
}

// SetReceiverRaw stores a receiver whose integration type is not natively
// modelled by ReceiverDef.  rawYAML must be valid YAML describing a single
// receiver block (with or without a "name" key — the name argument is always
// injected to keep it consistent with the URL parameter).
//
// Like UpsertReceiver it replaces an existing entry with the same name or
// appends if not found.
func (b *ConfigBuilder) SetReceiverRaw(name, rawYAML string) error {
	if name == "" {
		return fmt.Errorf("receiver name must not be empty")
	}
	var entry map[string]interface{}
	if err := yaml.Unmarshal([]byte(rawYAML), &entry); err != nil {
		return fmt.Errorf("parsing raw receiver YAML: %w", err)
	}
	if entry == nil {
		entry = make(map[string]interface{})
	}
	entry["name"] = name // URL param is always canonical

	raw, err := b.rawReceiversSlice()
	if err != nil {
		return err
	}
	found := false
	for i, r := range raw {
		if n, _ := r["name"].(string); n == name {
			raw[i] = entry
			found = true
			break
		}
	}
	if !found {
		raw = append(raw, entry)
	}
	b.raw["receivers"] = raw
	return nil
}

// DeleteReceiver removes the named receiver.
// Returns true if it was found and removed.
//
// Operates on the raw-map slice so that unknown integration types in sibling
// receivers are not lost during the store-back.
func (b *ConfigBuilder) DeleteReceiver(name string) (bool, error) {
	raw, err := b.rawReceiversSlice()
	if err != nil {
		return false, err
	}

	filtered := make([]map[string]interface{}, 0, len(raw))
	found := false
	for _, r := range raw {
		if n, _ := r["name"].(string); n == name {
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
//
// When rec.RawYAML is set the raw YAML fragment is embedded directly so that
// unknown integration types are validated accurately rather than being silently
// omitted by the typed marshaller.
func ValidateReceiver(rec ReceiverDef) ValidationResult {
	var receiverEntry interface{}
	if rec.RawYAML != "" {
		var raw map[string]interface{}
		if err := yaml.Unmarshal([]byte(rec.RawYAML), &raw); err != nil {
			return ValidationResult{Valid: false, Errors: []string{"invalid raw YAML: " + err.Error()}}
		}
		if raw == nil {
			raw = make(map[string]interface{})
		}
		raw["name"] = rec.Name
		receiverEntry = raw
	} else {
		receiverEntry = rec
	}
	tmp := map[string]interface{}{
		"route":     map[string]interface{}{"receiver": rec.Name},
		"receivers": []interface{}{receiverEntry},
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

// knownReceiverKeys is the set of YAML keys that ReceiverDef models natively.
// Any key outside this set signals an unknown integration type.
var knownReceiverKeys = map[string]bool{
	"name":              true,
	"webhook_configs":   true,
	"slack_configs":     true,
	"email_configs":     true,
	"pagerduty_configs": true,
	"opsgenie_configs":  true,
}

// parseReceivers round-trips the receivers section through YAML and annotates
// any receiver that contains unknown integration types with RawYAML.
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

	// Secondary pass: detect unknown integration types.
	// Re-unmarshal the same YAML into raw maps to inspect every key.
	var rawEntries []map[string]interface{}
	if err := yaml.Unmarshal(sect, &rawEntries); err == nil && len(rawEntries) == len(recs) {
		for i, raw := range rawEntries {
			for key := range raw {
				if !knownReceiverKeys[key] {
					// At least one unknown key: store the whole receiver as raw YAML
					// and clear the typed fields so callers use RawYAML exclusively.
					entryYAML, merr := yaml.Marshal(raw)
					if merr == nil {
						recs[i].RawYAML = string(entryYAML)
						recs[i].WebhookConfigs = nil
						recs[i].SlackConfigs = nil
						recs[i].EmailConfigs = nil
						recs[i].PagerdutyConfigs = nil
						recs[i].OpsgenieConfigs = nil
					}
					break
				}
			}
		}
	}

	return recs, nil
}

// rawReceiversSlice returns the receivers section as []map[string]interface{},
// preserving every key (including unknown integration types) by going through
// YAML serialisation.  UpsertReceiver, SetReceiverRaw, and DeleteReceiver all
// use this so that mutations on one receiver never corrupt another.
func (b *ConfigBuilder) rawReceiversSlice() ([]map[string]interface{}, error) {
	v, ok := b.raw["receivers"]
	if !ok || v == nil {
		return []map[string]interface{}{}, nil
	}
	sect, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("serializing receivers: %w", err)
	}
	var entries []map[string]interface{}
	if err := yaml.Unmarshal(sect, &entries); err != nil {
		return nil, fmt.Errorf("parsing receivers as raw maps: %w", err)
	}
	if entries == nil {
		return []map[string]interface{}{}, nil
	}
	return entries, nil
}
