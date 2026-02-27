package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gaetanars/alertlens/internal/alertmanager"
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

// Get handles GET /api/routing?instance=<name>.
func (h *RoutingHandler) Get(w http.ResponseWriter, r *http.Request) {
	instanceName := r.URL.Query().Get("instance")

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

	writeJSON(w, map[string]any{
		"alertmanager": client.Name(),
		"route":        routeToMap(cfg.Route),
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
