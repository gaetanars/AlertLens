package alertmanager

import (
	"testing"
)

// ─── matchesAll ───────────────────────────────────────────────────────────────

func TestMatchesAll_EqualityMatch(t *testing.T) {
	labels := map[string]string{"env": "prod", "team": "platform"}
	matchers := []Matcher{
		{Name: "env", Value: "prod", IsEqual: true},
		{Name: "team", Value: "platform", IsEqual: true},
	}
	if !matchesAll(labels, matchers) {
		t.Error("expected all matchers to match")
	}
}

func TestMatchesAll_EqualityNoMatch(t *testing.T) {
	labels := map[string]string{"env": "staging"}
	matchers := []Matcher{
		{Name: "env", Value: "prod", IsEqual: true},
	}
	if matchesAll(labels, matchers) {
		t.Error("expected matcher to not match")
	}
}

func TestMatchesAll_NotEqual(t *testing.T) {
	labels := map[string]string{"env": "staging"}
	matchers := []Matcher{
		{Name: "env", Value: "prod", IsEqual: false, IsRegex: false},
	}
	if !matchesAll(labels, matchers) {
		t.Error("expected not-equal matcher to match when value differs")
	}
}

func TestMatchesAll_NotEqualSameValue(t *testing.T) {
	labels := map[string]string{"env": "prod"}
	matchers := []Matcher{
		{Name: "env", Value: "prod", IsEqual: false, IsRegex: false},
	}
	if matchesAll(labels, matchers) {
		t.Error("expected not-equal matcher to not match when value is the same")
	}
}

func TestMatchesAll_RegexMatch(t *testing.T) {
	labels := map[string]string{"env": "production"}
	matchers := []Matcher{
		{Name: "env", Value: "prod.*", IsEqual: true, IsRegex: true},
	}
	if !matchesAll(labels, matchers) {
		t.Error("expected regex matcher to match")
	}
}

func TestMatchesAll_RegexNoMatch(t *testing.T) {
	labels := map[string]string{"env": "staging"}
	matchers := []Matcher{
		{Name: "env", Value: "prod.*", IsEqual: true, IsRegex: true},
	}
	if matchesAll(labels, matchers) {
		t.Error("expected regex matcher to not match")
	}
}

func TestMatchesAll_RegexNotEqual(t *testing.T) {
	labels := map[string]string{"env": "staging"}
	// IsEqual=false with IsRegex=true: should match when pattern does NOT match.
	matchers := []Matcher{
		{Name: "env", Value: "prod.*", IsEqual: false, IsRegex: true},
	}
	if !matchesAll(labels, matchers) {
		t.Error("expected negated regex matcher to match when value doesn't match pattern")
	}
}

func TestMatchesAll_MissingLabelEqualMatcher(t *testing.T) {
	labels := map[string]string{"team": "platform"}
	matchers := []Matcher{
		{Name: "env", Value: "prod", IsEqual: true},
	}
	if matchesAll(labels, matchers) {
		t.Error("missing label should not satisfy an equality matcher")
	}
}

func TestMatchesAll_MissingLabelNotEqualMatcher(t *testing.T) {
	// A label absent from an alert satisfies a not-equal matcher (AM behaviour).
	labels := map[string]string{"team": "platform"}
	matchers := []Matcher{
		{Name: "env", Value: "prod", IsEqual: false, IsRegex: false},
	}
	if !matchesAll(labels, matchers) {
		t.Error("missing label should satisfy a not-equal matcher")
	}
}

func TestMatchesAll_InvalidRegexTreatedAsNoMatch(t *testing.T) {
	labels := map[string]string{"env": "prod"}
	matchers := []Matcher{
		{Name: "env", Value: "[invalid", IsEqual: true, IsRegex: true},
	}
	if matchesAll(labels, matchers) {
		t.Error("invalid regex should be treated as non-matching")
	}
}

func TestMatchesAll_RegexCachedBetweenCalls(t *testing.T) {
	// Call twice with the same pattern to exercise the cache path.
	labels := map[string]string{"env": "production"}
	matchers := []Matcher{
		{Name: "env", Value: "prod.*", IsEqual: true, IsRegex: true},
	}
	for range 3 {
		if !matchesAll(labels, matchers) {
			t.Fatal("expected match on repeated calls (cache test)")
		}
	}
}

// ─── buildAckIndex ────────────────────────────────────────────────────────────

func TestBuildAckIndex_FiltersInactiveSilences(t *testing.T) {
	silences := []Silence{
		{
			ID:      "expired",
			Status:  SilenceStatus{State: "expired"},
			Matchers: []Matcher{{Name: labelAckType, Value: ackTypeVisual, IsEqual: true}},
		},
		{
			ID:     "active",
			Status: SilenceStatus{State: "active"},
			Matchers: []Matcher{
				{Name: labelAckType, Value: ackTypeVisual, IsEqual: true},
				{Name: labelAckBy, Value: "alice", IsEqual: true},
				{Name: "alertname", Value: "CPUHigh", IsEqual: true},
			},
		},
	}
	index := buildAckIndex(silences)
	if len(index) != 1 {
		t.Errorf("expected 1 active ack entry, got %d", len(index))
	}
	if index[0].by != "alice" {
		t.Errorf("expected ack by alice, got %q", index[0].by)
	}
	// Internal labels must be stripped from real matchers.
	for _, m := range index[0].matchers {
		if m.Name == labelAckType || m.Name == labelAckBy || m.Name == labelAckComment {
			t.Errorf("internal label %q leaked into ack matchers", m.Name)
		}
	}
}
