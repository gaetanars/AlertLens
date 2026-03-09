package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alertlens/alertlens/internal/alertmanager"
)

// ─── test fixtures ────────────────────────────────────────────────────────────

func makeHandlerTestAlerts() []alertmanager.EnrichedAlert {
	return []alertmanager.EnrichedAlert{
		{Alert: alertmanager.Alert{
			Fingerprint: "fp1",
			Labels:      map[string]string{"alertname": "CPUHigh", "severity": "critical", "env": "prod"},
			Status:      alertmanager.AlertStatus{State: "active"},
		}, Alertmanager: "prod-eu"},
		{Alert: alertmanager.Alert{
			Fingerprint: "fp2",
			Labels:      map[string]string{"alertname": "MemHigh", "severity": "warning", "env": "prod"},
			Status:      alertmanager.AlertStatus{State: "active"},
		}, Alertmanager: "prod-eu"},
		{Alert: alertmanager.Alert{
			Fingerprint: "fp3",
			Labels:      map[string]string{"alertname": "DiskFull", "severity": "critical", "env": "staging"},
			Status:      alertmanager.AlertStatus{State: "suppressed"},
		}, Alertmanager: "prod-us"},
	}
}

// fakeViewPool is a test double for AlertsHandler that bypasses real AM calls.
type fakeViewPool struct {
	alerts []alertmanager.EnrichedAlert
	err    error
}

// GetAlertsView simulates Pool.GetAlertsView with local filtering/grouping logic.
func (f *fakeViewPool) GetAlertsView(_ context.Context, params alertmanager.AlertsViewParams) (*alertmanager.AlertsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}

	// Apply severity filter.
	filtered := f.alerts
	if len(params.Severity) > 0 {
		sevSet := make(map[string]bool, len(params.Severity))
		for _, s := range params.Severity {
			sevSet[s] = true
		}
		var out []alertmanager.EnrichedAlert
		for _, a := range filtered {
			if sevSet[a.Labels["severity"]] {
				out = append(out, a)
			}
		}
		filtered = out
	}
	// Apply status filter.
	if len(params.Status) > 0 {
		stateSet := make(map[string]bool, len(params.Status))
		for _, s := range params.Status {
			stateSet[s] = true
		}
		var out []alertmanager.EnrichedAlert
		for _, a := range filtered {
			if stateSet[a.Status.State] {
				out = append(out, a)
			}
		}
		filtered = out
	}

	total := len(filtered)
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

	// Apply pagination.
	if offset > 0 || len(filtered) > limit {
		if offset >= len(filtered) {
			filtered = nil
		} else {
			end := offset + limit
			if end > len(filtered) {
				end = len(filtered)
			}
			filtered = filtered[offset:end]
		}
	}

	// Build groups.
	groups := localGroupAlerts(filtered, params.GroupBy)

	return &alertmanager.AlertsResponse{
		Groups: groups,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// localGroupAlerts mirrors pool.groupAlerts for the handler test package.
func localGroupAlerts(alerts []alertmanager.EnrichedAlert, groupBy []string) []alertmanager.AlertGroup {
	if len(groupBy) == 0 {
		if alerts == nil {
			alerts = []alertmanager.EnrichedAlert{}
		}
		return []alertmanager.AlertGroup{{
			Labels: map[string]string{},
			Alerts: alerts,
			Count:  len(alerts),
		}}
	}
	order := []string{}
	index := map[string]*alertmanager.AlertGroup{}
	for _, a := range alerts {
		key := ""
		lbls := map[string]string{}
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
			lbls[k] = v
			if i > 0 {
				key += "\x00"
			}
			key += k + "=" + v
		}
		if _, ok := index[key]; !ok {
			order = append(order, key)
			g := &alertmanager.AlertGroup{Labels: lbls, Alerts: []alertmanager.EnrichedAlert{}}
			index[key] = g
		}
		g := index[key]
		g.Alerts = append(g.Alerts, a)
		g.Count++
	}
	result := make([]alertmanager.AlertGroup, 0, len(order))
	for _, k := range order {
		result = append(result, *index[k])
	}
	return result
}

// alertsViewPoolIface is the interface AlertsHandler would use if refactored;
// here we use it for test injection without changing the production handler.
type alertsViewPoolIface interface {
	GetAlertsView(ctx context.Context, params alertmanager.AlertsViewParams) (*alertmanager.AlertsResponse, error)
}

// testAlertsHandler builds a http.HandlerFunc using the same logic as
// AlertsHandler.List but accepting the interface instead of *alertmanager.Pool.
func testAlertsHandler(pool alertsViewPoolIface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		base := alertmanager.AlertsQueryParams{
			Filter:    q["filter"],
			Instance:  q.Get("instance"),
			Silenced:  parseBool(q.Get("silenced"), false),
			Inhibited: parseBool(q.Get("inhibited"), false),
			Active:    parseBool(q.Get("active"), true),
		}
		groupBy := splitCommaSeparated(q["group_by"])
		severity := splitCommaSeparated(q["severity"])
		status := splitCommaSeparated(q["status"])

		limit, _ := parseIntParam(q.Get("limit"))
		offset, _ := parseIntParam(q.Get("offset"))

		params := alertmanager.AlertsViewParams{
			AlertsQueryParams: base,
			GroupBy:           groupBy,
			Severity:          severity,
			Status:            status,
			Limit:             limit,
			Offset:            offset,
		}

		resp, err := pool.GetAlertsView(r.Context(), params)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, resp)
	}
}

// parseIntParam parses an optional integer query parameter.
func parseIntParam(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// ─── HTTP handler tests ───────────────────────────────────────────────────────

func TestAlertsHandler_List_DefaultResponse(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("expected total=3, got %d", resp.Total)
	}
	// No group_by → 1 group containing all alerts
	if len(resp.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(resp.Groups))
	}
	if resp.Groups[0].Count != 3 {
		t.Errorf("expected group count=3, got %d", resp.Groups[0].Count)
	}
}

func TestAlertsHandler_List_GroupBySeverity(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?group_by=severity", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	// critical (2) + warning (1) = 2 groups
	if len(resp.Groups) != 2 {
		t.Errorf("expected 2 severity groups, got %d", len(resp.Groups))
	}
}

func TestAlertsHandler_List_GroupByAlertmanager(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?group_by=alertmanager", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp.Groups) != 2 {
		t.Errorf("expected 2 alertmanager groups, got %d", len(resp.Groups))
	}
}

func TestAlertsHandler_List_GroupByStatus(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?group_by=status", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if len(resp.Groups) != 2 {
		t.Errorf("expected 2 status groups (active+suppressed), got %d", len(resp.Groups))
	}
}

func TestAlertsHandler_List_FilterBySeverity(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?severity=critical", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total=2 for critical filter, got %d", resp.Total)
	}
}

func TestAlertsHandler_List_FilterByStatus(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?status=suppressed", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1 for suppressed filter, got %d", resp.Total)
	}
}

func TestAlertsHandler_List_FilterBySeverityAndStatus(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?severity=critical&status=suppressed", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("expected total=1 for critical+suppressed filter, got %d", resp.Total)
	}
}

func TestAlertsHandler_List_Pagination(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?limit=2&offset=0", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected total=3 (before pagination), got %d", resp.Total)
	}
	if resp.Limit != 2 {
		t.Errorf("expected limit=2, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected offset=0, got %d", resp.Offset)
	}
	// With limit=2 on 3 alerts → 1 group with 2 alerts
	totalInGroups := 0
	for _, g := range resp.Groups {
		totalInGroups += len(g.Alerts)
	}
	if totalInGroups != 2 {
		t.Errorf("expected 2 alerts in page, got %d", totalInGroups)
	}
}

func TestAlertsHandler_List_PaginationOffset(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts?limit=2&offset=2", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	totalInGroups := 0
	for _, g := range resp.Groups {
		totalInGroups += len(g.Alerts)
	}
	if totalInGroups != 1 {
		t.Errorf("expected 1 alert on second page (offset=2, limit=2 of 3), got %d", totalInGroups)
	}
}

func TestAlertsHandler_List_CommaGroupBy(t *testing.T) {
	pool := &fakeViewPool{alerts: makeHandlerTestAlerts()}
	handler := testAlertsHandler(pool)

	// Test comma-separated group_by param
	req := httptest.NewRequest(http.MethodGet, "/api/alerts?group_by=severity,status", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp alertmanager.AlertsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	// critical+active(1), warning+active(1), critical+suppressed(1) = 3 groups
	if len(resp.Groups) != 3 {
		t.Errorf("expected 3 groups (severity+status combos), got %d", len(resp.Groups))
	}
}

func TestAlertsHandler_List_PoolError(t *testing.T) {
	pool := &fakeViewPool{
		err: fmt.Errorf("all alertmanager instances failed to respond"),
	}
	handler := testAlertsHandler(pool)

	req := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 on pool error, got %d", w.Code)
	}
}

// ─── splitCommaSeparated ──────────────────────────────────────────────────────

func TestSplitCommaSeparated_Nil(t *testing.T) {
	got := splitCommaSeparated(nil)
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestSplitCommaSeparated_Single(t *testing.T) {
	got := splitCommaSeparated([]string{"severity"})
	if len(got) != 1 || got[0] != "severity" {
		t.Errorf("unexpected result: %v", got)
	}
}

func TestSplitCommaSeparated_Multi(t *testing.T) {
	got := splitCommaSeparated([]string{"severity,status"})
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d: %v", len(got), got)
	}
	if got[0] != "severity" || got[1] != "status" {
		t.Errorf("unexpected values: %v", got)
	}
}

func TestSplitCommaSeparated_Mixed(t *testing.T) {
	got := splitCommaSeparated([]string{"severity", "status,env"})
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d: %v", len(got), got)
	}
}
