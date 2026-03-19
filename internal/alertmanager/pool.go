package alertmanager

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/alertlens/alertlens/internal/config"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Pool manages multiple Alertmanager clients and provides aggregated access.
type Pool struct {
	clients []*Client
	logger  *zap.Logger
}

// NewPool creates a Pool from a list of AlertmanagerConfig entries.
// version is passed to each client for use in the User-Agent header.
func NewPool(cfgs []config.AlertmanagerConfig, logger *zap.Logger, version string) *Pool {
	clients := make([]*Client, 0, len(cfgs))
	for _, cfg := range cfgs {
		clients = append(clients, NewClient(cfg, version))
	}
	return &Pool{clients: clients, logger: logger}
}

// Client returns the client with the given name, or nil if not found.
func (p *Pool) Client(name string) *Client {
	for _, c := range p.clients {
		if c.name == name {
			return c
		}
	}
	return nil
}

// Clients returns all clients in the pool.
func (p *Pool) Clients() []*Client {
	return p.clients
}

// ─── Partial failure metadata ─────────────────────────────────────────────────

// InstanceError records a per-instance fetch error for metadata reporting.
type InstanceError struct {
	Instance string `json:"instance"`
	Error    string `json:"error"`
}

// ─── Aggregated operations ───────────────────────────────────────────────────

// alertInstanceResult holds the fetched data for a single AM instance.
type alertInstanceResult struct {
	name     string
	alerts   []Alert
	silences []Silence
}

// GetAggregatedAlerts fetches alerts from all instances concurrently using
// errgroup and returns a flat list enriched with instance name and ack info.
// Partial failures are logged and returned via the errors slice so callers can
// surface degraded-mode metadata to the UI (COR-02/ERR-04).
func (p *Pool) GetAggregatedAlerts(ctx context.Context, params AlertsQueryParams) ([]EnrichedAlert, []InstanceError) {
	// Filter to the requested instance(s).
	targets := p.selectClients(params.Instance)
	if len(targets) == 0 {
		return nil, nil
	}

	resultsMu := sync.Mutex{}
	results := make([]alertInstanceResult, 0, len(targets))
	var instanceErrors []InstanceError

	g, gCtx := errgroup.WithContext(ctx)
	for _, c := range targets {
		c := c // capture loop variable
		g.Go(func() error {
			alerts, err := c.GetAlerts(gCtx, params)
			if err != nil {
				p.logger.Warn("failed to fetch alerts",
					zap.String("instance", c.name), zap.Error(err))
				resultsMu.Lock()
				instanceErrors = append(instanceErrors, InstanceError{
					Instance: c.name,
					Error:    err.Error(),
				})
				resultsMu.Unlock()
				// Do NOT propagate the error: partial failure is acceptable.
				return nil
			}

			// Fetch silences for visual-ack computation; failure is non-fatal.
			silences, silErr := c.GetSilences(gCtx)
			if silErr != nil {
				p.logger.Warn("failed to fetch silences for ack computation",
					zap.String("instance", c.name), zap.Error(silErr))
			}

			resultsMu.Lock()
			results = append(results, alertInstanceResult{
				name:     c.name,
				alerts:   alerts,
				silences: silences,
			})
			resultsMu.Unlock()
			return nil
		})
	}

	// errgroup.Wait only returns errors from goroutines that returned non-nil,
	// which we suppressed above — so the error here is always nil.
	_ = g.Wait() //nolint:errcheck

	// COR-02/ERR-04: if every queried instance failed, surface a hard error
	// via the instanceErrors slice (the caller decides how to handle it).
	var enriched []EnrichedAlert
	for _, res := range results {
		ackIndex := buildAckIndex(res.silences)
		for _, a := range res.alerts {
			ea := EnrichedAlert{Alert: a, Alertmanager: res.name, InstanceID: res.name}
			ea.Ack = findAck(a, ackIndex)
			enriched = append(enriched, ea)
		}
	}

	return enriched, instanceErrors
}

// GetAggregatedSilences fetches silences from all instances concurrently.
// Partial failures are logged and returned as InstanceErrors.
func (p *Pool) GetAggregatedSilences(ctx context.Context, params SilenceQueryParams) ([]EnrichedSilence, []InstanceError) {
	targets := p.selectClients(params.Instance)
	if len(targets) == 0 {
		return nil, nil
	}

	mu := sync.Mutex{}
	var out []EnrichedSilence
	var instanceErrors []InstanceError

	g, gCtx := errgroup.WithContext(ctx)
	for _, c := range targets {
		c := c
		g.Go(func() error {
			silences, err := c.GetSilences(gCtx)
			if err != nil {
				p.logger.Warn("failed to fetch silences",
					zap.String("instance", c.name), zap.Error(err))
				mu.Lock()
				instanceErrors = append(instanceErrors, InstanceError{
					Instance: c.name,
					Error:    err.Error(),
				})
				mu.Unlock()
				return nil
			}

			mu.Lock()
			for _, s := range silences {
				if params.Type == "ack" && !IsAckSilence(s) {
					continue
				}
				if params.Type == "silence" && IsAckSilence(s) {
					continue
				}
				out = append(out, EnrichedSilence{Silence: s, Alertmanager: c.name})
			}
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait() //nolint:errcheck
	return out, instanceErrors
}

// GetInstanceStatuses fetches the status of all AM instances concurrently.
func (p *Pool) GetInstanceStatuses(ctx context.Context) []InstanceStatus {
	statuses := make([]InstanceStatus, len(p.clients))
	var wg sync.WaitGroup

	for i, c := range p.clients {
		wg.Add(1)
		go func(idx int, client *Client) {
			defer wg.Done()
			status := InstanceStatus{
				Name:      client.name,
				URL:       client.baseURL,
				HasTenant: client.tenantID != "",
			}
			amStatus, err := client.GetStatus(ctx)
			if err != nil {
				status.Healthy = false
				status.Error = err.Error()
			} else {
				status.Healthy = true
				status.Version = amStatus.VersionInfo.Version
			}
			statuses[idx] = status
		}(i, c)
	}

	wg.Wait()
	return statuses
}

// selectClients returns the subset of clients to query.
// When instance is empty, all clients are returned.
func (p *Pool) selectClients(instance string) []*Client {
	if instance == "" {
		return p.clients
	}
	for _, c := range p.clients {
		if c.name == instance {
			return []*Client{c}
		}
	}
	return nil
}

// ─── Ack helpers ─────────────────────────────────────────────────────────────

// ackEntry holds ack info extracted from a visual-ack silence.
type ackEntry struct {
	silenceID string
	by        string
	comment   string
	matchers  []Matcher
}

// buildAckIndex creates a list of ackEntry from active visual-ack silences.
func buildAckIndex(silences []Silence) []ackEntry {
	var entries []ackEntry
	for _, s := range silences {
		if s.Status.State != "active" {
			continue
		}
		if !IsAckSilence(s) {
			continue
		}
		by, comment := ExtractAckInfo(s)
		// Filter out AlertLens internal matchers to keep only "real" matchers.
		var matchers []Matcher
		for _, m := range s.Matchers {
			switch m.Name {
			case labelAckType, labelAckBy, labelAckComment:
				continue
			}
			matchers = append(matchers, m)
		}
		entries = append(entries, ackEntry{
			silenceID: s.ID,
			by:        by,
			comment:   comment,
			matchers:  matchers,
		})
	}
	return entries
}

// findAck returns the Ack if any active visual-ack silence matches the alert.
func findAck(a Alert, index []ackEntry) *Ack {
	for _, e := range index {
		if matchesAll(a.Labels, e.matchers) {
			return &Ack{
				Active:    true,
				By:        e.by,
				Comment:   e.comment,
				SilenceID: e.silenceID,
			}
		}
	}
	return nil
}

// regexCache stores compiled regexes to avoid recompiling the same pattern on
// every alert/silence evaluation. sync.Map is safe for concurrent access and
// handles the rare write-after-read race via LoadOrStore.
var regexCache sync.Map // map[string]*regexp.Regexp

// cachedRegex returns a compiled *regexp.Regexp for the given pattern,
// reusing a previously compiled instance when available.
func cachedRegex(pattern string) (*regexp.Regexp, error) {
	return CachedRegex(pattern)
}

// CachedRegex is the exported variant of cachedRegex, used by other packages
// (e.g. API handlers) that need to match labels against route node matchers
// without duplicating the cache.
func CachedRegex(pattern string) (*regexp.Regexp, error) {
	if v, ok := regexCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	// LoadOrStore so two goroutines compiling the same pattern simultaneously
	// end up using the same *regexp.Regexp.
	actual, _ := regexCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp), nil
}

// matchesAll returns true if all matchers match the given label set.
// COR-01: handles regex matchers (IsRegex) in addition to equality matchers.
func matchesAll(labels map[string]string, matchers []Matcher) bool {
	for _, m := range matchers {
		val, ok := labels[m.Name]
		if m.IsRegex {
			re, err := cachedRegex(m.Value)
			if err != nil {
				// Treat an invalid regex as non-matching.
				return false
			}
			matched := ok && re.MatchString(val)
			if m.IsEqual && !matched {
				return false
			}
			if !m.IsEqual && matched {
				return false
			}
			continue
		}
		if !ok && m.IsEqual {
			return false
		}
		if m.IsEqual && val != m.Value {
			return false
		}
		if !m.IsEqual && val == m.Value {
			return false
		}
	}
	return true
}

// EnrichedSilence is a Silence enriched with its Alertmanager instance name.
type EnrichedSilence struct {
	Silence
	Alertmanager string `json:"alertmanager"`
}

// ─── View-layer helpers ──────────────────────────────────────────────────────

// GetAlertsView fetches, filters and groups alerts according to AlertsViewParams.
// This is the main entry point for the alerts list/kanban API endpoint.
func (p *Pool) GetAlertsView(ctx context.Context, params AlertsViewParams) (*AlertsResponse, error) {
	// Step 1 – fetch raw enriched alerts from all matching instances.
	raw, instanceErrors := p.GetAggregatedAlerts(ctx, params.AlertsQueryParams)

	// COR-02/ERR-04: if all instances failed (nothing fetched, all errored),
	// return a hard error so the UI displays a clear failure state.
	if len(raw) == 0 && len(instanceErrors) > 0 && len(instanceErrors) == len(p.selectClients(params.Instance)) {
		return nil, fmt.Errorf("all alertmanager instances failed to respond")
	}

	// Step 2 – apply view-layer filters (severity, status) not handled by AM.
	filtered := applyViewFilters(raw, params)

	// Step 3 – group.
	groups := groupAlerts(filtered, params.GroupBy)

	// Step 4 – total before pagination.
	total := len(filtered)

	// Step 5 – pagination across the flat list (within groups is too complex for MVP).
	limit := params.Limit
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	// Re-flatten, paginate, re-group.
	if offset > 0 || len(filtered) > limit {
		end := offset + limit
		if offset >= len(filtered) {
			filtered = nil
		} else {
			if end > len(filtered) {
				end = len(filtered)
			}
			filtered = filtered[offset:end]
		}
		groups = groupAlerts(filtered, params.GroupBy)
	}

	return &AlertsResponse{
		Groups:         groups,
		Total:          total,
		Limit:          limit,
		Offset:         offset,
		PartialFailures: instanceErrors,
	}, nil
}

// applyViewFilters filters alerts by severity and status.
func applyViewFilters(alerts []EnrichedAlert, params AlertsViewParams) []EnrichedAlert {
	if len(params.Severity) == 0 && len(params.Status) == 0 {
		return alerts
	}

	severitySet := make(map[string]struct{}, len(params.Severity))
	for _, s := range params.Severity {
		severitySet[s] = struct{}{}
	}
	statusSet := make(map[string]struct{}, len(params.Status))
	for _, s := range params.Status {
		statusSet[s] = struct{}{}
	}

	out := make([]EnrichedAlert, 0, len(alerts))
	for _, a := range alerts {
		if len(severitySet) > 0 {
			sev := a.Labels["severity"]
			if _, ok := severitySet[sev]; !ok {
				continue
			}
		}
		if len(statusSet) > 0 {
			if _, ok := statusSet[a.Status.State]; !ok {
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

// groupAlerts groups alerts by the provided label keys.
// When groupBy is empty a single group with empty Labels is returned.
func groupAlerts(alerts []EnrichedAlert, groupBy []string) []AlertGroup {
	if len(groupBy) == 0 {
		return []AlertGroup{{
			Labels: map[string]string{},
			Alerts: alerts,
			Count:  len(alerts),
		}}
	}

	type groupKey = string
	order := []groupKey{}
	index := map[groupKey]*AlertGroup{}

	for _, a := range alerts {
		key, labels := buildGroupKey(a, groupBy)
		if _, exists := index[key]; !exists {
			order = append(order, key)
			g := &AlertGroup{Labels: labels, Alerts: []EnrichedAlert{}}
			index[key] = g
		}
		g := index[key]
		g.Alerts = append(g.Alerts, a)
		g.Count++
	}

	groups := make([]AlertGroup, 0, len(order))
	for _, k := range order {
		groups = append(groups, *index[k])
	}
	return groups
}

// buildGroupKey builds a stable string key and the label map for a group.
// Special virtual key "alertmanager" maps to EnrichedAlert.Alertmanager.
func buildGroupKey(a EnrichedAlert, groupBy []string) (string, map[string]string) {
	labels := make(map[string]string, len(groupBy))
	key := ""
	for i, k := range groupBy {
		var v string
		switch k {
		case "alertmanager":
			v = a.Alertmanager
		case "status":
			v = a.Status.State
		default:
			v = a.Labels[k]
		}
		labels[k] = v
		if i > 0 {
			key += "\x00"
		}
		key += k + "=" + v
	}
	return key, labels
}
