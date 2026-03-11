package incident_test

import (
	"testing"
	"time"

	"github.com/alertlens/alertlens/internal/incident"
)

func newTestStore(t *testing.T) *incident.Store {
	t.Helper()
	return incident.NewStore()
}

// ─── Create ───────────────────────────────────────────────────────────────────

func TestCreate_Basic(t *testing.T) {
	s := newTestStore(t)
	inc, err := s.Create(incident.CreateIncidentRequest{
		Title:     "Redis latency spike",
		Severity:  "critical",
		CreatedBy: "oncall-bot",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inc.ID == "" {
		t.Error("expected non-empty ID")
	}
	if inc.Status != incident.StatusOpen {
		t.Errorf("expected status OPEN, got %s", inc.Status)
	}
	if len(inc.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(inc.Events))
	}
	evt := inc.Events[0]
	if evt.Kind != incident.EventKindCreated {
		t.Errorf("expected CREATED event, got %s", evt.Kind)
	}
	if evt.Status != incident.StatusOpen {
		t.Errorf("expected event status OPEN, got %s", evt.Status)
	}
	if evt.Seq != 1 {
		t.Errorf("expected seq 1, got %d", evt.Seq)
	}
}

func TestCreate_RequiresTitle(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Create(incident.CreateIncidentRequest{CreatedBy: "bot"})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

// ─── State machine ────────────────────────────────────────────────────────────

func TestStateTransitions_HappyPath(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{
		Title: "DB down", CreatedBy: "bot",
	})

	// OPEN → ACK
	inc, err := s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindAck, Actor: "alice", Message: "I'm on it",
	})
	if err != nil {
		t.Fatalf("ACK failed: %v", err)
	}
	if inc.Status != incident.StatusAck {
		t.Errorf("expected ACK, got %s", inc.Status)
	}

	// ACK → INVESTIGATING
	inc, err = s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindInvestigating, Actor: "alice",
	})
	if err != nil {
		t.Fatalf("INVESTIGATING failed: %v", err)
	}
	if inc.Status != incident.StatusInvestigating {
		t.Errorf("expected INVESTIGATING, got %s", inc.Status)
	}

	// INVESTIGATING → RESOLVED
	inc, err = s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindResolved, Actor: "alice", Message: "Fixed via rollback",
	})
	if err != nil {
		t.Fatalf("RESOLVED failed: %v", err)
	}
	if inc.Status != incident.StatusResolved {
		t.Errorf("expected RESOLVED, got %s", inc.Status)
	}
	if inc.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
}

func TestStateTransition_InvalidBlocked(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{Title: "X", CreatedBy: "bot"})

	// Directly from OPEN to RESOLVED then try to ACK → must fail
	_, _ = s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindResolved, Actor: "bot",
	})
	_, err := s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindAck, Actor: "alice",
	})
	if err == nil {
		t.Fatal("expected transition error RESOLVED → ACK")
	}
}

func TestStateTransition_Reopen(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{Title: "Y", CreatedBy: "bot"})
	_, _ = s.AddEvent(inc.ID, incident.AddEventRequest{Kind: incident.EventKindResolved, Actor: "bot"})

	inc, err := s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindReopened, Actor: "carol", Message: "Still happening",
	})
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	if inc.Status != incident.StatusOpen {
		t.Errorf("expected OPEN after reopen, got %s", inc.Status)
	}
	if inc.ResolvedAt != nil {
		t.Error("expected ResolvedAt to be cleared after reopen")
	}
}

func TestStateTransition_Comment_NoStatusChange(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{Title: "Z", CreatedBy: "bot"})

	inc, err := s.AddEvent(inc.ID, incident.AddEventRequest{
		Kind: incident.EventKindComment, Actor: "dave", Message: "Still digging",
	})
	if err != nil {
		t.Fatalf("comment failed: %v", err)
	}
	if inc.Status != incident.StatusOpen {
		t.Errorf("expected OPEN unchanged, got %s", inc.Status)
	}
	if len(inc.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(inc.Events))
	}
	if inc.Events[1].Status != "" {
		t.Errorf("expected empty status in COMMENT event, got %q", inc.Events[1].Status)
	}
}

// ─── Ledger integrity ─────────────────────────────────────────────────────────

func TestLedger_SeqMonotonic(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{Title: "Seq test", CreatedBy: "bot"})
	kinds := []incident.EventKind{
		incident.EventKindAck,
		incident.EventKindComment,
		incident.EventKindInvestigating,
		incident.EventKindResolved,
	}
	for _, k := range kinds {
		inc, _ = s.AddEvent(inc.ID, incident.AddEventRequest{Kind: k, Actor: "bot"})
	}
	for i, evt := range inc.Events {
		if evt.Seq != i+1 {
			t.Errorf("event[%d]: expected seq %d, got %d", i, i+1, evt.Seq)
		}
	}
}

func TestLedger_TimestampsMonotonic(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{Title: "Time test", CreatedBy: "bot"})
	time.Sleep(time.Millisecond)
	inc, _ = s.AddEvent(inc.ID, incident.AddEventRequest{Kind: incident.EventKindAck, Actor: "bot"})

	for i := 1; i < len(inc.Events); i++ {
		if !inc.Events[i].OccurredAt.After(inc.Events[i-1].OccurredAt) &&
			!inc.Events[i].OccurredAt.Equal(inc.Events[i-1].OccurredAt) {
			t.Errorf("event timestamps not monotonically non-decreasing at index %d", i)
		}
	}
}

// ─── List / Get ───────────────────────────────────────────────────────────────

func TestList_Pagination(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 5; i++ {
		s.Create(incident.CreateIncidentRequest{Title: "Inc", CreatedBy: "bot"}) //nolint:errcheck
	}
	resp := s.List(incident.ListFilter{Limit: 2, Offset: 0})
	if len(resp.Incidents) != 2 {
		t.Errorf("expected 2 incidents, got %d", len(resp.Incidents))
	}
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
}

func TestList_StatusFilter(t *testing.T) {
	s := newTestStore(t)
	inc1, _ := s.Create(incident.CreateIncidentRequest{Title: "A", CreatedBy: "bot"})
	inc2, _ := s.Create(incident.CreateIncidentRequest{Title: "B", CreatedBy: "bot"})
	s.AddEvent(inc1.ID, incident.AddEventRequest{Kind: incident.EventKindResolved, Actor: "bot"}) //nolint:errcheck
	_ = inc2

	resp := s.List(incident.ListFilter{Status: incident.StatusResolved})
	if resp.Total != 1 {
		t.Errorf("expected 1 resolved incident, got %d", resp.Total)
	}
}

func TestGet_NotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("nonexistent")
	if err != incident.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_ImmutableCopy(t *testing.T) {
	s := newTestStore(t)
	inc, _ := s.Create(incident.CreateIncidentRequest{Title: "Copy test", CreatedBy: "bot"})
	got, _ := s.Get(inc.ID)
	// Mutate the returned copy — must not affect the stored version.
	got.Title = "mutated"
	got2, _ := s.Get(inc.ID)
	if got2.Title == "mutated" {
		t.Error("Get returned a reference to the stored incident, not a copy")
	}
}
