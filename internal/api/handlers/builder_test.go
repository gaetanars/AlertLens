package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"github.com/alertlens/alertlens/internal/config"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// ─── Helpers ──────────────────────────────────────────────────────────────────

// builderAMServer starts a fake Alertmanager that returns cfgYAML in the
// /api/v2/status response.  The caller must call srv.Close().
func builderAMServer(t *testing.T, cfgYAML string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		status := alertmanager.AMStatus{
			Cluster: alertmanager.ClusterStatus{Status: "ready"},
			Config:  alertmanager.AMConfig{Original: cfgYAML},
		}
		json.NewEncoder(w).Encode(status) //nolint:errcheck
	})
	return httptest.NewServer(mux)
}

func buildBuilderPool(t *testing.T, name, url string) *alertmanager.Pool {
	t.Helper()
	cfgs := []config.AlertmanagerConfig{{Name: name, URL: url}}
	logger, _ := zap.NewDevelopment()
	return alertmanager.NewPool(cfgs, logger, "test")
}

// withChiParam injects a chi URL parameter into the request context so that
// chi.URLParam works without going through a full router.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// ─── ReceiverRoutes ───────────────────────────────────────────────────────────

// receiverRoutesResponse mirrors the JSON shape of ReceiverRoutes responses.
type receiverRoutesResponse struct {
	Receiver     string             `json:"receiver"`
	ReferencedBy []receiverRouteRef `json:"referenced_by"`
}

func TestReceiverRoutes(t *testing.T) {
	t.Parallel()

	// A config where:
	//   - root route → "root-recv"
	//   - child 0    → "child-recv" (depth 1)
	//   - child 0 > grandchild 0 → "deep-recv" (depth 2)
	//   - child 1    → "other-recv" (depth 1)
	const cfg = `
route:
  receiver: "root-recv"
  routes:
    - matchers:
        - env="prod"
      receiver: "child-recv"
      routes:
        - matchers:
            - severity="critical"
          receiver: "deep-recv"
    - matchers:
        - env="staging"
      receiver: "other-recv"
receivers:
  - name: "root-recv"
  - name: "child-recv"
  - name: "deep-recv"
  - name: "other-recv"
  - name: "unref-recv"
`

	cases := []struct {
		name           string
		lookupReceiver string
		wantLen        int
		wantDepth0     *int // expected depth of first result, nil to skip
	}{
		{
			name:           "root receiver referenced at depth 0",
			lookupReceiver: "root-recv",
			wantLen:        1,
			wantDepth0:     intPtr(0),
		},
		{
			name:           "child receiver referenced at depth 1",
			lookupReceiver: "child-recv",
			wantLen:        1,
			wantDepth0:     intPtr(1),
		},
		{
			name:           "deeply nested receiver at depth 2",
			lookupReceiver: "deep-recv",
			wantLen:        1,
			wantDepth0:     intPtr(2),
		},
		{
			name:           "receiver exists in config but not in any route",
			lookupReceiver: "unref-recv",
			wantLen:        0,
			wantDepth0:     nil,
		},
		{
			name:           "receiver name not in config at all",
			lookupReceiver: "ghost-recv",
			wantLen:        0,
			wantDepth0:     nil,
		},
	}

	srv := builderAMServer(t, cfg)
	t.Cleanup(srv.Close) // must use Cleanup, not defer, so the server stays up for parallel sub-tests
	pool := buildBuilderPool(t, "test", srv.URL)
	h := NewBuilderHandler(pool)

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet,
				"/api/builder/receivers/"+tc.lookupReceiver+"/routes", nil)
			req = withChiParam(req, "name", tc.lookupReceiver)
			rr := httptest.NewRecorder()

			h.ReceiverRoutes(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d — body: %s", rr.Code, rr.Body.String())
			}

			var resp receiverRoutesResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if resp.Receiver != tc.lookupReceiver {
				t.Errorf("expected receiver %q, got %q", tc.lookupReceiver, resp.Receiver)
			}
			if len(resp.ReferencedBy) != tc.wantLen {
				t.Errorf("expected %d refs, got %d", tc.wantLen, len(resp.ReferencedBy))
			}
			if tc.wantDepth0 != nil && len(resp.ReferencedBy) > 0 {
				if resp.ReferencedBy[0].Depth != *tc.wantDepth0 {
					t.Errorf("expected depth %d, got %d", *tc.wantDepth0, resp.ReferencedBy[0].Depth)
				}
			}
			// referenced_by must never be JSON null — always an array.
			if resp.ReferencedBy == nil {
				t.Error("referenced_by must be a JSON array, not null")
			}
		})
	}
}

func intPtr(v int) *int { return &v }
