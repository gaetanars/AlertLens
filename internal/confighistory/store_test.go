package confighistory

import (
	"sync"
	"testing"
	"time"
)

func record(mode, instance string) SaveRecord {
	return SaveRecord{
		SavedAt:      time.Now(),
		Mode:         mode,
		Alertmanager: instance,
		Actor:        "config-editor",
	}
}

func TestStore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "list on unknown instance returns empty slice",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				got := s.List("nonexistent")
				if got == nil {
					t.Fatal("expected non-nil slice, got nil")
				}
				if len(got) != 0 {
					t.Fatalf("expected 0 records, got %d", len(got))
				}
			},
		},
		{
			name: "append single record, list returns it",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				r := record("disk", "am1")
				s.Append("am1", r)
				got := s.List("am1")
				if len(got) != 1 {
					t.Fatalf("expected 1 record, got %d", len(got))
				}
				if got[0].Mode != "disk" {
					t.Errorf("expected mode disk, got %q", got[0].Mode)
				}
			},
		},
		{
			name: "newest-first ordering",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				modes := []string{"disk", "github", "gitlab"}
				for _, m := range modes {
					s.Append("am1", record(m, "am1"))
				}
				got := s.List("am1")
				if len(got) != 3 {
					t.Fatalf("expected 3 records, got %d", len(got))
				}
				// Newest first: last appended should be at index 0.
				if got[0].Mode != "gitlab" {
					t.Errorf("expected gitlab first, got %q", got[0].Mode)
				}
				if got[2].Mode != "disk" {
					t.Errorf("expected disk last, got %q", got[2].Mode)
				}
			},
		},
		{
			name: "cap at 50, oldest evicted on overflow",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				// Append 51 records tagged with their index via CommitSHA.
				for i := range 51 {
					r := record("disk", "am1")
					r.CommitSHA = string(rune('a' + i%26)) // just a tag
					if i == 0 {
						r.CommitSHA = "oldest"
					}
					if i == 50 {
						r.CommitSHA = "newest"
					}
					s.Append("am1", r)
				}
				got := s.List("am1")
				if len(got) != maxEntriesPerInstance {
					t.Fatalf("expected %d records, got %d", maxEntriesPerInstance, len(got))
				}
				// First entry (newest) should be "newest".
				if got[0].CommitSHA != "newest" {
					t.Errorf("expected newest first, got %q", got[0].CommitSHA)
				}
				// "oldest" must have been evicted.
				for _, rec := range got {
					if rec.CommitSHA == "oldest" {
						t.Error("oldest record should have been evicted")
					}
				}
			},
		},
		{
			name: "instances are isolated",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				s.Append("am1", record("disk", "am1"))
				s.Append("am2", record("github", "am2"))
				got1 := s.List("am1")
				got2 := s.List("am2")
				if len(got1) != 1 || got1[0].Mode != "disk" {
					t.Errorf("am1: unexpected result %v", got1)
				}
				if len(got2) != 1 || got2[0].Mode != "github" {
					t.Errorf("am2: unexpected result %v", got2)
				}
			},
		},
		{
			name: "list returns a copy (mutations do not affect stored slice)",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				s.Append("am1", record("disk", "am1"))
				got := s.List("am1")
				got[0].Mode = "mutated"
				again := s.List("am1")
				if again[0].Mode == "mutated" {
					t.Error("List returned a reference to internal slice; mutations leaked")
				}
			},
		},
		{
			name: "concurrent append and list are race-free",
			run: func(t *testing.T) {
				t.Parallel()
				s := NewStore()
				var wg sync.WaitGroup
				for range 20 {
					wg.Add(2)
					go func() {
						defer wg.Done()
						s.Append("am1", record("disk", "am1"))
					}()
					go func() {
						defer wg.Done()
						_ = s.List("am1")
					}()
				}
				wg.Wait()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.run)
	}
}
