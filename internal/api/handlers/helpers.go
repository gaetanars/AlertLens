// Package handlers contains the HTTP handler functions for each AlertLens API
// resource, wired into the chi router by the api package.
package handlers

import (
	"fmt"
	"net/http"

	"github.com/alertlens/alertlens/internal/alertmanager"
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
		// Use fmt.Sprintf so the instance name is a typed argument rather than
		// a raw string concatenation; writeError routes through json.Encoder
		// which handles any special characters correctly.
		writeError(w, fmt.Sprintf("unknown alertmanager instance: %q", instance), http.StatusBadRequest)
		return nil
	}
	return c
}
