package configbuilder

import (
	"strings"
	"testing"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

const baseConfig = `
route:
  receiver: 'null'
receivers:
  - name: 'null'
`

func mustBuilder(t *testing.T, raw string) *ConfigBuilder {
	t.Helper()
	b, err := NewConfigBuilder([]byte(raw))
	if err != nil {
		t.Fatalf("NewConfigBuilder: %v", err)
	}
	return b
}

// ─── NewConfigBuilder ─────────────────────────────────────────────────────────

func TestNewConfigBuilder_Empty(t *testing.T) {
	b, err := NewConfigBuilder(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil builder")
	}
}

func TestNewConfigBuilder_ValidYAML(t *testing.T) {
	b, err := NewConfigBuilder([]byte(baseConfig))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil builder")
	}
}

func TestNewConfigBuilder_InvalidYAML(t *testing.T) {
	_, err := NewConfigBuilder([]byte("{{not valid yaml"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// ─── Build / BuildRaw ─────────────────────────────────────────────────────────

func TestBuildRaw_RoundTrip(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	out, err := b.BuildRaw()
	if err != nil {
		t.Fatalf("BuildRaw: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected non-empty YAML output")
	}
}

func TestBuild_ValidatesConfig(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	out, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(string(out), "receiver") {
		t.Error("expected 'receiver' in output YAML")
	}
}

func TestBuild_FailsOnInvalidConfig(t *testing.T) {
	b := mustBuilder(t, "")
	// An empty config has no route/receivers — Build should fail.
	_, err := b.Build()
	if err == nil {
		t.Error("expected Build to fail for an empty config")
	}
}

// ─── Time Intervals: ListTimeIntervals ───────────────────────────────────────

func TestListTimeIntervals_Empty(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	entries, err := b.ListTimeIntervals()
	if err != nil {
		t.Fatalf("ListTimeIntervals: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestListTimeIntervals_WithExisting(t *testing.T) {
	raw := baseConfig + `
time_intervals:
  - name: business-hours
    time_intervals:
      - times:
          - start_time: "09:00"
            end_time: "17:00"
        weekdays: ["monday:friday"]
`
	b := mustBuilder(t, raw)
	entries, err := b.ListTimeIntervals()
	if err != nil {
		t.Fatalf("ListTimeIntervals: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "business-hours" {
		t.Errorf("expected name 'business-hours', got %q", entries[0].Name)
	}
}

// ─── Time Intervals: UpsertTimeInterval ──────────────────────────────────────

func TestUpsertTimeInterval_Add(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	entry := TimeIntervalEntry{
		Name: "oncall",
		TimeIntervals: []TimeIntervalDef{
			{
				Times:    []TimeRangeDef{{StartTime: "08:00", EndTime: "20:00"}},
				Weekdays: []string{"monday:friday"},
			},
		},
	}
	if err := b.UpsertTimeInterval(entry); err != nil {
		t.Fatalf("UpsertTimeInterval: %v", err)
	}

	entries, err := b.ListTimeIntervals()
	if err != nil {
		t.Fatalf("ListTimeIntervals: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "oncall" {
		t.Errorf("expected name 'oncall', got %q", entries[0].Name)
	}
}

func TestUpsertTimeInterval_Update(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	entry := TimeIntervalEntry{
		Name:          "oncall",
		TimeIntervals: []TimeIntervalDef{{Weekdays: []string{"monday:friday"}}},
	}
	_ = b.UpsertTimeInterval(entry)

	// Update with different weekdays.
	updated := TimeIntervalEntry{
		Name:          "oncall",
		TimeIntervals: []TimeIntervalDef{{Weekdays: []string{"saturday", "sunday"}}},
	}
	if err := b.UpsertTimeInterval(updated); err != nil {
		t.Fatalf("UpsertTimeInterval (update): %v", err)
	}

	entries, err := b.ListTimeIntervals()
	if err != nil {
		t.Fatalf("ListTimeIntervals: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after update, got %d", len(entries))
	}
	if entries[0].TimeIntervals[0].Weekdays[0] != "saturday" {
		t.Errorf("expected updated weekday 'saturday', got %q", entries[0].TimeIntervals[0].Weekdays[0])
	}
}

func TestUpsertTimeInterval_MultipleEntries(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	for _, name := range []string{"a", "b", "c"} {
		if err := b.UpsertTimeInterval(TimeIntervalEntry{
			Name:          name,
			TimeIntervals: []TimeIntervalDef{{Weekdays: []string{"monday"}}},
		}); err != nil {
			t.Fatalf("UpsertTimeInterval(%q): %v", name, err)
		}
	}
	entries, err := b.ListTimeIntervals()
	if err != nil {
		t.Fatalf("ListTimeIntervals: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}
}

func TestUpsertTimeInterval_EmptyNameReturnsError(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	err := b.UpsertTimeInterval(TimeIntervalEntry{Name: ""})
	if err == nil {
		t.Error("expected error for empty name")
	}
}

// ─── Time Intervals: DeleteTimeInterval ──────────────────────────────────────

func TestDeleteTimeInterval_Existing(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	_ = b.UpsertTimeInterval(TimeIntervalEntry{
		Name:          "to-delete",
		TimeIntervals: []TimeIntervalDef{{Weekdays: []string{"monday"}}},
	})

	found, err := b.DeleteTimeInterval("to-delete")
	if err != nil {
		t.Fatalf("DeleteTimeInterval: %v", err)
	}
	if !found {
		t.Error("expected found=true for existing entry")
	}

	entries, _ := b.ListTimeIntervals()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}
}

func TestDeleteTimeInterval_NotFound(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	found, err := b.DeleteTimeInterval("nonexistent")
	if err != nil {
		t.Fatalf("DeleteTimeInterval: %v", err)
	}
	if found {
		t.Error("expected found=false for non-existent entry")
	}
}

func TestDeleteTimeInterval_LeavesOthers(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	for _, name := range []string{"keep-1", "remove", "keep-2"} {
		_ = b.UpsertTimeInterval(TimeIntervalEntry{
			Name:          name,
			TimeIntervals: []TimeIntervalDef{{Weekdays: []string{"monday"}}},
		})
	}
	_, _ = b.DeleteTimeInterval("remove")
	entries, _ := b.ListTimeIntervals()
	if len(entries) != 2 {
		t.Errorf("expected 2 remaining entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Name == "remove" {
			t.Error("deleted entry 'remove' still present")
		}
	}
}

// ─── Receivers: List / Upsert / Delete ───────────────────────────────────────

func TestListReceivers_FromBaseConfig(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	recs, err := b.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("expected 1 receiver, got %d", len(recs))
	}
	if recs[0].Name != "null" {
		t.Errorf("expected receiver 'null', got %q", recs[0].Name)
	}
}

func TestUpsertReceiver_Add(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	rec := ReceiverDef{
		Name: "alerts-webhook",
		WebhookConfigs: []WebhookConfigDef{
			{URL: "https://example.com/hook", SendResolved: true},
		},
	}
	if err := b.UpsertReceiver(rec); err != nil {
		t.Fatalf("UpsertReceiver: %v", err)
	}
	recs, err := b.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 receivers, got %d", len(recs))
	}
}

func TestUpsertReceiver_Update(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	_ = b.UpsertReceiver(ReceiverDef{
		Name:           "webhook",
		WebhookConfigs: []WebhookConfigDef{{URL: "https://old.example.com"}},
	})
	_ = b.UpsertReceiver(ReceiverDef{
		Name:           "webhook",
		WebhookConfigs: []WebhookConfigDef{{URL: "https://new.example.com"}},
	})
	recs, _ := b.ListReceivers()
	var found bool
	for _, r := range recs {
		if r.Name == "webhook" {
			found = true
			if len(r.WebhookConfigs) == 0 || r.WebhookConfigs[0].URL != "https://new.example.com" {
				t.Errorf("expected updated URL, got %q", r.WebhookConfigs[0].URL)
			}
		}
	}
	if !found {
		t.Error("receiver 'webhook' not found after upsert")
	}
}

func TestUpsertReceiver_EmptyNameReturnsError(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	err := b.UpsertReceiver(ReceiverDef{Name: ""})
	if err == nil {
		t.Error("expected error for empty receiver name")
	}
}

func TestDeleteReceiver_Found(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	_ = b.UpsertReceiver(ReceiverDef{Name: "to-delete"})
	found, err := b.DeleteReceiver("to-delete")
	if err != nil {
		t.Fatalf("DeleteReceiver: %v", err)
	}
	if !found {
		t.Error("expected found=true")
	}
	recs, _ := b.ListReceivers()
	for _, r := range recs {
		if r.Name == "to-delete" {
			t.Error("deleted receiver still present")
		}
	}
}

func TestDeleteReceiver_NotFound(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	found, err := b.DeleteReceiver("ghost")
	if err != nil {
		t.Fatalf("DeleteReceiver: %v", err)
	}
	if found {
		t.Error("expected found=false for non-existent receiver")
	}
}

// ─── Route ────────────────────────────────────────────────────────────────────

func TestGetRoute_FromBaseConfig(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	route, err := b.GetRoute()
	if err != nil {
		t.Fatalf("GetRoute: %v", err)
	}
	if route == nil {
		t.Fatal("expected non-nil route")
	}
	if route.Receiver != "null" {
		t.Errorf("expected receiver 'null', got %q", route.Receiver)
	}
}

func TestGetRoute_EmptyConfig(t *testing.T) {
	b := mustBuilder(t, "")
	route, err := b.GetRoute()
	if err != nil {
		t.Fatalf("GetRoute on empty config: %v", err)
	}
	if route != nil {
		t.Error("expected nil route for empty config")
	}
}

func TestSetRoute_RoundTrip(t *testing.T) {
	b := mustBuilder(t, baseConfig)
	b.SetRoute(RouteSpec{
		Receiver:       "null",
		GroupBy:        []string{"alertname", "cluster"},
		GroupWait:      "30s",
		RepeatInterval: "4h",
		Routes: []RouteSpec{
			{
				Receiver: "null",
				Matchers: []string{`severity="critical"`},
			},
		},
	})

	route, err := b.GetRoute()
	if err != nil {
		t.Fatalf("GetRoute after SetRoute: %v", err)
	}
	if len(route.GroupBy) != 2 {
		t.Errorf("expected 2 group_by labels, got %d", len(route.GroupBy))
	}
	if len(route.Routes) != 1 {
		t.Errorf("expected 1 child route, got %d", len(route.Routes))
	}
}

// ─── ValidateTimeInterval ─────────────────────────────────────────────────────

func TestValidateTimeInterval_BusinessHours(t *testing.T) {
	entry := TimeIntervalEntry{
		Name: "business-hours",
		TimeIntervals: []TimeIntervalDef{
			{
				Times:    []TimeRangeDef{{StartTime: "09:00", EndTime: "17:00"}},
				Weekdays: []string{"monday:friday"},
				Location: "UTC",
			},
		},
	}
	result := ValidateTimeInterval(entry)
	if !result.Valid {
		t.Errorf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidateTimeInterval_InvalidTime(t *testing.T) {
	entry := TimeIntervalEntry{
		Name: "bad-times",
		TimeIntervals: []TimeIntervalDef{
			{
				Times: []TimeRangeDef{{StartTime: "25:00", EndTime: "99:00"}},
			},
		},
	}
	result := ValidateTimeInterval(entry)
	if result.Valid {
		t.Error("expected invalid for out-of-range time")
	}
}

func TestValidateTimeInterval_InvalidLocation(t *testing.T) {
	entry := TimeIntervalEntry{
		Name: "bad-tz",
		TimeIntervals: []TimeIntervalDef{
			{Location: "Not/A/Timezone"},
		},
	}
	result := ValidateTimeInterval(entry)
	if result.Valid {
		t.Error("expected invalid for unknown timezone")
	}
}

// ─── End-to-end: build a full config from scratch ────────────────────────────

func TestBuild_FullConfigFromScratch(t *testing.T) {
	b, err := NewConfigBuilder(nil)
	if err != nil {
		t.Fatalf("NewConfigBuilder: %v", err)
	}

	// Add a webhook receiver.
	if err := b.UpsertReceiver(ReceiverDef{
		Name:           "webhook",
		WebhookConfigs: []WebhookConfigDef{{URL: "https://example.com/alertmanager-hook"}},
	}); err != nil {
		t.Fatalf("UpsertReceiver: %v", err)
	}

	// Add a time interval.
	if err := b.UpsertTimeInterval(TimeIntervalEntry{
		Name: "business-hours",
		TimeIntervals: []TimeIntervalDef{
			{
				Times:    []TimeRangeDef{{StartTime: "09:00", EndTime: "17:00"}},
				Weekdays: []string{"monday:friday"},
			},
		},
	}); err != nil {
		t.Fatalf("UpsertTimeInterval: %v", err)
	}

	// Set the root route.
	b.SetRoute(RouteSpec{
		Receiver: "webhook",
		GroupBy:  []string{"alertname"},
		Routes: []RouteSpec{
			{
				Receiver:          "webhook",
				Matchers:          []string{`severity="critical"`},
				MuteTimeIntervals: []string{"business-hours"},
			},
		},
	})

	out, err := b.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	s := string(out)
	if !strings.Contains(s, "business-hours") {
		t.Error("expected 'business-hours' in output YAML")
	}
	if !strings.Contains(s, "https://example.com/alertmanager-hook") {
		t.Error("expected webhook URL in output YAML")
	}
}

// ─── Unknown receiver: RawYAML escape hatch ───────────────────────────────────

// configWithVictorOps is a valid Alertmanager YAML that contains a receiver
// using victorops_configs — an integration type not modelled by ReceiverDef.
const configWithVictorOps = `
route:
  receiver: 'null'
receivers:
  - name: 'null'
  - name: 'ops-victorops'
    victorops_configs:
      - api_key: 'secret-key'
        routing_key: 'MyTeam'
        message_type: 'CRITICAL'
`

func TestListReceivers_UnknownType_SetsRawYAML(t *testing.T) {
	t.Parallel()
	b := mustBuilder(t, configWithVictorOps)
	recs, err := b.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}

	var vic *ReceiverDef
	for i := range recs {
		if recs[i].Name == "ops-victorops" {
			vic = &recs[i]
			break
		}
	}
	if vic == nil {
		t.Fatal("receiver 'ops-victorops' not found")
	}
	if vic.RawYAML == "" {
		t.Error("expected RawYAML to be non-empty for unknown integration type")
	}
	// Typed config slices must be zero — caller must use RawYAML exclusively.
	if len(vic.WebhookConfigs) != 0 || len(vic.SlackConfigs) != 0 ||
		len(vic.EmailConfigs) != 0 || len(vic.PagerdutyConfigs) != 0 ||
		len(vic.OpsgenieConfigs) != 0 {
		t.Error("expected all typed config slices to be empty for unknown integration type")
	}
	if !strings.Contains(vic.RawYAML, "victorops_configs") {
		t.Errorf("RawYAML does not contain 'victorops_configs': %s", vic.RawYAML)
	}
}

func TestUpsertReceiver_RawYAML_RoundTrip(t *testing.T) {
	t.Parallel()
	b := mustBuilder(t, configWithVictorOps)

	// Retrieve the unknown receiver.
	recs, err := b.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers: %v", err)
	}
	var vic ReceiverDef
	for _, r := range recs {
		if r.Name == "ops-victorops" {
			vic = r
			break
		}
	}
	if vic.RawYAML == "" {
		t.Fatal("pre-condition: RawYAML must be set")
	}

	// Upsert the receiver back via RawYAML path.
	if err := b.UpsertReceiver(vic); err != nil {
		t.Fatalf("UpsertReceiver (raw path): %v", err)
	}

	// The raw YAML output must still contain the original victorops_configs.
	out, err := b.BuildRaw()
	if err != nil {
		t.Fatalf("BuildRaw: %v", err)
	}
	if !strings.Contains(string(out), "victorops_configs") {
		t.Error("victorops_configs not preserved after round-trip upsert")
	}
	if !strings.Contains(string(out), "secret-key") {
		t.Error("victorops api_key not preserved after round-trip upsert")
	}
}

func TestUpsertReceiver_RawYAML_DoesNotCorruptSiblings(t *testing.T) {
	t.Parallel()
	b := mustBuilder(t, configWithVictorOps)
	recs, _ := b.ListReceivers()

	var vic ReceiverDef
	for _, r := range recs {
		if r.Name == "ops-victorops" {
			vic = r
		}
	}
	// Upsert the unknown receiver; the 'null' sibling must survive.
	_ = b.UpsertReceiver(vic)
	after, err := b.ListReceivers()
	if err != nil {
		t.Fatalf("ListReceivers after upsert: %v", err)
	}
	var foundNull bool
	for _, r := range after {
		if r.Name == "null" {
			foundNull = true
		}
	}
	if !foundNull {
		t.Error("sibling receiver 'null' was lost after raw upsert")
	}
}

func TestSetReceiverRaw_MalformedYAML_ReturnsError(t *testing.T) {
	t.Parallel()
	b := mustBuilder(t, baseConfig)
	err := b.SetReceiverRaw("bad", "{{not valid yaml")
	if err == nil {
		t.Error("expected error for malformed YAML, got nil")
	}
}
