// Package confighistory implements an in-memory, per-instance save history for
// the Config Builder (feature 009). History resets on process restart.
package confighistory

import (
	"sync"
	"time"
)

// maxEntriesPerInstance is the maximum number of save records kept per
// Alertmanager instance. When the limit is exceeded the oldest entry is evicted.
const maxEntriesPerInstance = 50

// SaveRecord describes a single config save operation.
type SaveRecord struct {
	SavedAt      time.Time `json:"saved_at"`
	Mode         string    `json:"mode"`
	Alertmanager string    `json:"alertmanager"`
	Actor        string    `json:"actor"`
	CommitSHA    string    `json:"commit_sha"`
	HTMLURL      string    `json:"html_url"`
	RawYAML      string    `json:"raw_yaml"`
}

// Store is an in-memory, per-instance save-history ledger. All operations are
// safe for concurrent use. History is lost on process restart.
type Store struct {
	mu      sync.RWMutex
	entries map[string][]SaveRecord // key = alertmanager instance name
}

// NewStore creates an empty, ready-to-use Store.
func NewStore() *Store {
	return &Store{
		entries: make(map[string][]SaveRecord),
	}
}

// Append records a save operation for the given Alertmanager instance.
// If the instance already has maxEntriesPerInstance records, the oldest is
// evicted before the new record is appended.
func (s *Store) Append(instance string, r SaveRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := s.entries[instance]
	if len(list) >= maxEntriesPerInstance {
		// Evict the oldest (first) entry.
		list = list[1:]
	}
	s.entries[instance] = append(list, r)
}

// List returns a copy of the save records for the given instance, ordered
// newest-first. Returns an empty (non-nil) slice when no records exist.
func (s *Store) List(instance string) []SaveRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	src := s.entries[instance]
	if len(src) == 0 {
		return []SaveRecord{}
	}

	// Return a reversed copy so the caller always gets newest-first order.
	out := make([]SaveRecord, len(src))
	for i, r := range src {
		out[len(src)-1-i] = r
	}
	return out
}
