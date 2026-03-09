package alertmanager

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/alertlens/alertlens/internal/config"
	"go.uber.org/zap"
)

// Pool manages multiple Alertmanager clients and provides aggregated access.
type Pool struct {
	clients []*Client
	logger  *zap.Logger
}

// NewPool creates a Pool from a list of AlertmanagerConfig entries.
func NewPool(cfgs []config.AlertmanagerConfig, logger *zap.Logger) *Pool {
	clients := make([]*Client, 0, len(cfgs))
	for _, cfg := range cfgs {
		clients = append(clients, NewClient(cfg))
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

// ─── Aggregated operations ───────────────────────────────────────────────────

// instanceResult holds the result of querying a single AM instance.
type instanceResult[T any] struct {
	name  string
	items []T
	err   error
}

// GetAggregatedAlerts fetches alerts from all instances concurrently and
// returns a flat list enriched with instance name and ack information.
func (p *Pool) GetAggregatedAlerts(ctx context.Context, params AlertsQueryParams) ([]EnrichedAlert, error) {
	type alertResult struct {
		name    string
		alerts  []Alert
		silences []Silence
		err     error
	}

	results := make([]alertResult, len(p.clients))
	var wg sync.WaitGroup

	for i, c := range p.clients {
		// Apply instance filter: skip if a specific instance is requested and
		// this is not it.
		if params.Instance != "" && c.name != params.Instance {
			continue
		}

		wg.Add(1)
		go func(idx int, client *Client) {
			defer wg.Done()
			res := alertResult{name: client.name}

			alerts, err := client.GetAlerts(ctx, params)
			if err != nil {
				res.err = err
				p.logger.Warn("failed to fetch alerts", zap.String("instance", client.name), zap.Error(err))
				results[idx] = res
				return
			}
			res.alerts = alerts

			// Fetch silences to compute visual-ack info.
			silences, err := client.GetSilences(ctx)
			if err != nil {
				p.logger.Warn("failed to fetch silences for ack computation",
					zap.String("instance", client.name), zap.Error(err))
			}
			res.silences = silences
			results[idx] = res
		}(i, c)
	}

	wg.Wait()

	var enriched []EnrichedAlert
	queried := 0
	failed := 0
	for _, res := range results {
		if res.name == "" {
			continue // slot was not used (instance filter skipped it)
		}
		queried++
		if res.err != nil {
			failed++
			continue
		}
		// Build ack index from silences.
		ackIndex := buildAckIndex(res.silences)

		for _, a := range res.alerts {
			ea := EnrichedAlert{Alert: a, Alertmanager: res.name}
			ea.Ack = findAck(a, ackIndex)
			enriched = append(enriched, ea)
		}
	}

	// COR-02/ERR-04: return an error only when every queried instance failed.
	if queried > 0 && queried == failed {
		return nil, fmt.Errorf("all alertmanager instances failed to respond")
	}

	return enriched, nil
}

// GetAggregatedSilences fetches silences from all instances concurrently.
func (p *Pool) GetAggregatedSilences(ctx context.Context, params SilenceQueryParams) ([]EnrichedSilence, error) {
	type silResult = instanceResult[Silence]
	results := make([]silResult, len(p.clients))
	var wg sync.WaitGroup

	for i, c := range p.clients {
		if params.Instance != "" && c.name != params.Instance {
			continue
		}
		wg.Add(1)
		go func(idx int, client *Client) {
			defer wg.Done()
			silences, err := client.GetSilences(ctx)
			if err != nil {
				p.logger.Warn("failed to fetch silences", zap.String("instance", client.name), zap.Error(err))
				results[idx] = silResult{name: client.name, err: err}
				return
			}
			results[idx] = silResult{name: client.name, items: silences}
		}(i, c)
	}

	wg.Wait()

	var out []EnrichedSilence
	queried := 0
	failed := 0
	for _, res := range results {
		if res.name == "" {
			continue // slot was not used (instance filter skipped it)
		}
		queried++
		if res.err != nil {
			failed++
			continue
		}
		for _, s := range res.items {
			if params.Type == "ack" && !IsAckSilence(s) {
				continue
			}
			if params.Type == "silence" && IsAckSilence(s) {
				continue
			}
			out = append(out, EnrichedSilence{Silence: s, Alertmanager: res.name})
		}
	}

	// COR-02/ERR-04: return an error only when every queried instance failed.
	if queried > 0 && queried == failed {
		return nil, fmt.Errorf("all alertmanager instances failed to respond")
	}

	return out, nil
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
	raw, err := p.GetAggregatedAlerts(ctx, params.AlertsQueryParams)
	if err != nil {
		return nil, err
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
		Groups: groups,
		Total:  total,
		Limit:  limit,
		Offset: offset,
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
		if k == "alertmanager" {
			v = a.Alertmanager
		} else if k == "status" {
			v = a.Status.State
		} else {
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
