package handlers

import (
	"net/http"

	"github.com/gaetanars/alertlens/internal/alertmanager"
)

// resolveClient returns the named alertmanager client, or the first available
// one when instance is empty. Writes a JSON error response and returns nil on
// failure, so callers can simply check for nil.
func resolveClient(pool *alertmanager.Pool, w http.ResponseWriter, instance string) *alertmanager.Client {
	if instance == "" {
		clients := pool.Clients()
		if len(clients) == 0 {
			writeError(w, "no alertmanager instances configured", http.StatusServiceUnavailable)
			return nil
		}
		return clients[0]
	}
	c := pool.Client(instance)
	if c == nil {
		writeError(w, "unknown alertmanager instance: "+instance, http.StatusBadRequest)
		return nil
	}
	return c
}
