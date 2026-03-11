package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/alertlens/alertlens/internal/alertmanager"
	amconfig "github.com/prometheus/alertmanager/config"
	"gopkg.in/yaml.v3"
)

// RoutingHandler handles routing-tree requests.
type RoutingHandler struct {
	pool *alertmanager.Pool
}

// NewRoutingHandler creates a RoutingHandler.
func NewRoutingHandler(pool *alertmanager.Pool) *RoutingHandler {
	return &RoutingHandler{pool: pool}
}

// Get handles GET /api/routing?instance=<name>&annotate_alerts=true.
//
// When annotate_alerts=true the handler fetches live alerts from the instance
// and annotates every route node with the count of alerts whose labels satisfy
// that node's matchers.  The root node receives the total alert count; child
// nodes receive the subset that matches their additional constraints.
func (h *RoutingHandler) Get(w http.ResponseWriter, r *http.Request) {
	instanceName := r.URL.Query().Get("instance")
	annotate := r.URL.Query().Get("annotate_alerts") == "true"

	client := resolveClient(h.pool, w, instanceName)
	if client == nil {
		return
	}

	status, err := client.GetStatus(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}

	cfg, err := parseAMConfig(status.Config.Original)
	if err != nil {
		writeError(w, "parsing alertmanager config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	routeMap := routeToMap(cfg.Route)

	if annotate {
		// Fetch active alerts for annotation — failure is non-fatal; we log and
		// return the tree without counts rather than error the whole request.
		alerts, fetchErr := client.GetAlerts(r.Context(), alertmanager.AlertsQueryParams{Active: true})
		if fetchErr == nil {
			annotateRouteCounts(routeMap, alerts)
		}
	}

	writeJSON(w, map[string]any{
		"alertmanager": client.Name(),
		"route":        routeMap,
	})
}

// Match handles POST /api/routing/match.
func (h *RoutingHandler) Match(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Alertmanager string            `json:"alertmanager"`
		Labels       map[string]string `json:"labels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	client := resolveClient(h.pool, w, req.Alertmanager)
	if client == nil {
		return
	}

	status, err := client.GetStatus(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}

	cfg, err := parseAMConfig(status.Config.Original)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	matched := matchRoute(cfg.Route, req.Labels)
	writeJSON(w, map[string]any{"matched_routes": matched})
}


// ─── Prometheus config parsing ───────────────────────────────────────────────

func parseAMConfig(raw string) (*amconfig.Config, error) {
	var cfg amconfig.Config
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// routeToMap converts a Prometheus Route to a JSON-serializable map.
func routeToMap(route *amconfig.Route) map[string]any {
	if route == nil {
		return nil
	}

	matchers := make([]map[string]any, 0, len(route.Matchers))
	for _, m := range route.Matchers {
		matchers = append(matchers, map[string]any{
			"name":     m.Name,
			"value":    m.Value,
			"is_regex": m.Type.String() == "=~" || m.Type.String() == "!~",
			"is_equal": m.Type.String() == "=" || m.Type.String() == "=~",
		})
	}

	children := make([]map[string]any, 0, len(route.Routes))
	for _, child := range route.Routes {
		children = append(children, routeToMap(child))
	}

	r := map[string]any{
		"receiver":        route.Receiver,
		"matchers":        matchers,
		"group_by":        labelNamesToStrings(route.GroupByStr),
		"continue":        route.Continue,
		"routes":          children,
	}
	if route.GroupWait != nil {
		r["group_wait"] = route.GroupWait.String()
	}
	if route.GroupInterval != nil {
		r["group_interval"] = route.GroupInterval.String()
	}
	if route.RepeatInterval != nil {
		r["repeat_interval"] = route.RepeatInterval.String()
	}
	if len(route.MuteTimeIntervals) > 0 {
		r["mute_time_intervals"] = route.MuteTimeIntervals
	}
	if len(route.ActiveTimeIntervals) > 0 {
		r["active_time_intervals"] = route.ActiveTimeIntervals
	}
	return r
}

func labelNamesToStrings(names []string) []string {
	if names == nil {
		return []string{}
	}
	return names
}

// matchRoute walks the route tree and returns the chain of matched routes.
func matchRoute(route *amconfig.Route, labels map[string]string) []map[string]any {
	if route == nil {
		return nil
	}
	if !routeMatchesLabels(route, labels) {
		return nil
	}

	result := []map[string]any{routeToMap(route)}

	for _, child := range route.Routes {
		if matches := matchRoute(child, labels); len(matches) > 0 {
			result = append(result, matches...)
			// child.Continue means "even though this child matched, keep evaluating
			// the next sibling routes". Default (false) stops at the first match.
			if !child.Continue {
				break
			}
		}
	}
	return result
}

func routeMatchesLabels(route *amconfig.Route, labels map[string]string) bool {
	for _, m := range route.Matchers {
		val, ok := labels[m.Name]
		if !ok {
			if m.Type.String() == "=" || m.Type.String() == "=~" {
				return false
			}
			continue
		}
		matched := m.Matches(val)
		if !matched {
			return false
		}
	}
	return true
}

// ─── Alert count annotation ───────────────────────────────────────────────────

// routeNodeMatcher is a lightweight matcher extracted from a routeToMap node.
type routeNodeMatcher struct {
	name    string
	value   string
	isRegex bool
	isEqual bool
}

// annotateRouteCounts walks a routeToMap result (a nested map[string]any tree)
// and adds "alert_count" and "severity_counts" fields to each node based on
// how many of the provided alerts satisfy that node's matchers.
//
// The root node (which typically has no matchers) will receive the total alert
// count.  Child nodes receive the subset of alerts whose labels satisfy their
// own matchers — regardless of routing stop semantics, because for the
// visualiser we want to show "which alerts flow through each branch" rather
// than "which alerts are finally delivered here".
func annotateRouteCounts(node map[string]any, alerts []alertmanager.Alert) {
	var nodeMatchers []routeNodeMatcher
	if raw, ok := node["matchers"].([]map[string]any); ok {
		for _, m := range raw {
			nm := routeNodeMatcher{
				name:    mapStrVal(m, "name"),
				value:   mapStrVal(m, "value"),
				isEqual: true, // default
			}
			if v, ok := m["is_regex"].(bool); ok {
				nm.isRegex = v
			}
			if v, ok := m["is_equal"].(bool); ok {
				nm.isEqual = v
			}
			nodeMatchers = append(nodeMatchers, nm)
		}
	}

	count := 0
	severityCounts := map[string]int{}
	for _, a := range alerts {
		if nodeMatchesAlertLabels(nodeMatchers, a.Labels) {
			count++
			if sev := a.Labels["severity"]; sev != "" {
				severityCounts[sev]++
			}
		}
	}
	node["alert_count"] = count
	node["severity_counts"] = severityCounts

	// Recurse into children.
	if routes, ok := node["routes"].([]map[string]any); ok {
		for _, child := range routes {
			annotateRouteCounts(child, alerts)
		}
	}
}

// nodeMatchesAlertLabels returns true if all routeNodeMatchers are satisfied
// by the given label set.  Empty matcher list = catch-all (always matches).
func nodeMatchesAlertLabels(matchers []routeNodeMatcher, labels map[string]string) bool {
	for _, m := range matchers {
		val := labels[m.name]
		if m.isRegex {
			re, err := alertmanager.CachedRegex(m.value)
			if err != nil {
				return false // treat invalid regex as non-matching
			}
			matched := re.MatchString(val)
			if m.isEqual && !matched {
				return false
			}
			if !m.isEqual && matched {
				return false
			}
		} else {
			if m.isEqual && val != m.value {
				return false
			}
			if !m.isEqual && val == m.value {
				return false
			}
		}
	}
	return true
}

func mapStrVal(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
