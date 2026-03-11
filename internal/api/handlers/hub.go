package handlers

import (
	"context"
	"net/http"
	"sync"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"go.uber.org/zap"
)

// HubHandler implements the hub-and-spoke aggregation endpoint.
// It provides a single view of all Alertmanager instances (spokes) from the
// AlertLens hub perspective, including per-instance health and alert counts.
type HubHandler struct {
	pool   *alertmanager.Pool
	logger *zap.Logger
}

// NewHubHandler creates a HubHandler.
func NewHubHandler(pool *alertmanager.Pool, logger *zap.Logger) *HubHandler {
	return &HubHandler{pool: pool, logger: logger}
}

// SpokeStats holds per-instance aggregated statistics.
type SpokeStats struct {
	Name             string            `json:"name"`
	URL              string            `json:"url"`
	Healthy          bool              `json:"healthy"`
	Version          string            `json:"version"`
	Error            string            `json:"error,omitempty"`
	AlertCount       int               `json:"alert_count"`
	ActiveCount      int               `json:"active_count"`
	SuppressedCount  int               `json:"suppressed_count"`
	SeverityCounts   map[string]int    `json:"severity_counts"`
}

// HubTopology is the response envelope for GET /api/hub/topology.
type HubTopology struct {
	Hub    HubStats     `json:"hub"`
	Spokes []SpokeStats `json:"spokes"`
}

// HubStats describes the AlertLens hub itself.
type HubStats struct {
	Name              string `json:"name"`
	TotalInstances    int    `json:"total_instances"`
	HealthyInstances  int    `json:"healthy_instances"`
	TotalAlerts       int    `json:"total_alerts"`
	CriticalAlerts    int    `json:"critical_alerts"`
}

// Topology handles GET /api/hub/topology.
//
// It fans out concurrently to all configured Alertmanager instances (spokes),
// collects health status and alert counts, then returns an aggregated hub view.
// Partial failures (unhealthy instances) are surfaced in the response — the
// handler always returns 200 so the UI can render degraded-mode indicators.
func (h *HubHandler) Topology(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	clients := h.pool.Clients()

	// Fetch health of all instances in one fan-out.
	instanceStatuses := h.pool.GetInstanceStatuses(ctx)
	statusByName := make(map[string]alertmanager.InstanceStatus, len(instanceStatuses))
	for _, s := range instanceStatuses {
		statusByName[s.Name] = s
	}

	spokes := make([]SpokeStats, len(clients))
	var wg sync.WaitGroup

	for i, c := range clients {
		wg.Add(1)
		go func(idx int, client *alertmanager.Client) {
			defer wg.Done()
			spoke := h.collectSpokeStats(ctx, client, statusByName[client.Name()])
			spokes[idx] = spoke
		}(i, c)
	}
	wg.Wait()

	// Aggregate hub-level totals.
	hub := HubStats{
		Name:           "AlertLens",
		TotalInstances: len(clients),
	}
	for _, s := range spokes {
		if s.Healthy {
			hub.HealthyInstances++
		}
		hub.TotalAlerts += s.AlertCount
		hub.CriticalAlerts += s.SeverityCounts["critical"]
	}

	writeJSON(w, HubTopology{Hub: hub, Spokes: spokes})
}

// collectSpokeStats fetches alert statistics for a single AM instance.
// It uses the pre-fetched instanceStatus to avoid redundant health checks.
// Errors are captured into SpokeStats.Error; the spoke is marked unhealthy.
func (h *HubHandler) collectSpokeStats(
	ctx context.Context,
	client *alertmanager.Client,
	status alertmanager.InstanceStatus,
) SpokeStats {
	spoke := SpokeStats{
		Name:           client.Name(),
		URL:            status.URL,
		Healthy:        status.Healthy,
		Version:        status.Version,
		SeverityCounts: map[string]int{},
	}
	if !status.Healthy {
		spoke.Error = status.Error
		return spoke
	}

	// Fetch active alerts for per-instance alert-count statistics.
	params := alertmanager.AlertsQueryParams{
		Instance: client.Name(),
		Active:   true,
	}
	enriched, _ := h.pool.GetAggregatedAlerts(ctx, params)
	spoke.AlertCount = len(enriched)
	for _, a := range enriched {
		switch a.Status.State {
		case "active":
			spoke.ActiveCount++
		case "suppressed":
			spoke.SuppressedCount++
		}
		if sev := a.Labels["severity"]; sev != "" {
			spoke.SeverityCounts[sev]++
		}
	}

	return spoke
}
