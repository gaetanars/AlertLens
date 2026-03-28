// Package api wires the chi router, middleware stack, and SPA fallback handler
// for the AlertLens HTTP server.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"github.com/alertlens/alertlens/internal/api/handlers"
	"github.com/alertlens/alertlens/internal/auth"
	"github.com/alertlens/alertlens/internal/confighistory"
	"github.com/alertlens/alertlens/internal/gitops"
	"github.com/alertlens/alertlens/internal/incident"
)

// maxRequestBodyBytes is the hard cap on incoming request bodies (10 MiB).
// Alertmanager configs are typically well under 1 MiB; this protects against
// trivial OOM attacks while leaving plenty of headroom.
const maxRequestBodyBytes = 10 << 20 // 10 MiB

// buildCSPPolicy constructs the Content-Security-Policy header value.
//
// scriptHashes is a space-separated list of 'sha256-<base64>' tokens for any
// inline scripts that must be allowed (e.g. SvelteKit's bootstrapper).  When
// empty, script-src falls back to 'self' only — which is correct for API-only
// responses but will block the SPA in a browser.
//
// The policy:
//   - Restricts scripts to same-origin plus the supplied hashes (no unsafe-inline).
//   - Allows inline styles (required by SvelteKit / Tailwind).
//   - Blocks <object>, <embed> and <frame*> entirely.
//   - Pins base-uri and form-action to 'self'.
func buildCSPPolicy(scriptHashes string) string {
	scriptSrc := "'self'"
	if scriptHashes != "" {
		scriptSrc += " " + scriptHashes
	}
	return "default-src 'self'; " +
		"script-src " + scriptSrc + "; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: blob:; " +
		"font-src 'self'; " +
		"connect-src 'self'; " +
		"object-src 'none'; " +
		"base-uri 'self'; " +
		"form-action 'self'; " +
		"frame-ancestors 'none';"
}

// NewRouter wires all API routes and returns the root http.Handler.
// The frontendFS is served for all non-API routes (SPA fallback).
// ghPusher / glPusher must be nil gitops.Pusher (not typed-nil) when the
// corresponding forge is not configured, so interface nil-checks in handlers work.
// secureCookies controls the Secure attribute on the CSRF cookie; set to true
// when the application is served behind HTTPS.
// scriptHashes is the space-separated list of 'sha256-<base64>' tokens for the
// inline scripts present in dist/index.html (produced by the Vite csp-hash
// plugin).  Pass an empty string when no inline scripts are present.
func NewRouter(
	pool *alertmanager.Pool,
	authSvc *auth.Service,
	ghPusher gitops.Pusher,
	glPusher gitops.Pusher,
	frontendFS http.FileSystem,
	allowedOrigins []string,
	secureCookies bool,
	version string,
	logger *zap.Logger,
	incidentStore *incident.Store,
	scriptHashes string,
	loginRateLimitBurst int,
) http.Handler {
	r := chi.NewRouter()

	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	// SEC-CWE-321: derive the CSRF secret from the auth service's JWT signing
	// key rather than using a hardcoded constant.  The derived key is unique to
	// CSRF purposes and rotates whenever the admin password changes.
	csrfSecret := authSvc.CSRFSecret()

	loginRL := auth.NewLoginRateLimiter(loginRateLimitBurst)

	// ─── Global middleware ───────────────────────────────────────────────────
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(zapMiddleware(logger))
	r.Use(middleware.Recoverer)
	// Limit every request body to maxRequestBodyBytes to prevent OOM.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
			next.ServeHTTP(w, r)
		})
	})
	// SEC-CSP: Content-Security-Policy to prevent XSS exploitation.
	// The policy is built at router-construction time so the inline-script
	// hash from dist/csp-hash.txt is captured in the closure without any
	// per-request overhead.
	cspPolicy := buildCSPPolicy(scriptHashes)
	r.Use(func(next http.Handler) http.Handler {
		return cspMiddlewareWithPolicy(cspPolicy, next)
	})
	// SEC-CSRF: Double-submit cookie CSRF protection.
	r.Use(auth.CSRFMiddleware(csrfSecret, secureCookies))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", auth.CSRFHeaderName},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// ─── Instantiate handlers ────────────────────────────────────────────────
	saveHistory := confighistory.NewStore()

	healthH := handlers.NewHealthHandler(version)
	authH   := handlers.NewAuthHandler(authSvc)
	amH     := handlers.NewAlertmanagersHandler(pool)
	alH     := handlers.NewAlertsHandler(pool)
	silH    := handlers.NewSilencesHandler(pool)
	rtH     := handlers.NewRoutingHandler(pool)
	cfgH    := handlers.NewConfigHandler(pool, ghPusher, glPusher, saveHistory)
	bulkH   := handlers.NewBulkHandler(pool)
	bldrH   := handlers.NewBuilderHandler(pool)
	hubH    := handlers.NewHubHandler(pool, logger)
	incH    := handlers.NewIncidentsHandler(incidentStore)

	// ─── Role middleware shorthands ──────────────────────────────────────────
	requireViewer       := authSvc.RequireViewer
	requireSilencer     := authSvc.RequireRole(auth.RoleSilencer)
	requireConfigEditor := authSvc.RequireRole(auth.RoleConfigEditor)

	// ─── API routes ──────────────────────────────────────────────────────────
	r.Route("/api", func(r chi.Router) {
		// Health
		r.Get("/health", healthH.Health)

		// Auth — login is rate-limited per IP to mitigate brute-force attacks.
		r.Get("/auth/status", authH.Status)
		r.With(loginRL.Middleware).Post("/auth/login", authH.Login)
		r.Post("/auth/logout", authH.Logout)

		// Alertmanager instances
		r.Get("/alertmanagers", amH.List)

		// Hub-and-spoke topology (aggregated view of all AM instances)
		r.Get("/hub/topology", hubH.Topology)

		// Alerts — viewer role minimum when auth is enabled; public otherwise.
		r.With(requireViewer).Get("/alerts", alH.List)

		// Silences:
		//   read   → public (viewer-level, no token required)
		//   write  → silencer role minimum
		r.Get("/silences", silH.List)
		r.Get("/silences/{id}", silH.Get)
		r.With(requireSilencer).Post("/silences", silH.Create)
		r.With(requireSilencer).Put("/silences/{id}", silH.Update)
		r.With(requireSilencer).Delete("/silences/{id}", silH.Delete)

		// Routing tree (public)
		r.Get("/routing", rtH.Get)
		r.Post("/routing/match", rtH.Match)

		// Config builder — config-editor role minimum
		r.With(requireConfigEditor).Get("/config", cfgH.Get)
		r.With(requireConfigEditor).Post("/config/validate", cfgH.Validate)
		r.With(requireConfigEditor).Post("/config/diff", cfgH.Diff)
		r.With(requireConfigEditor).Post("/config/save", cfgH.Save)
		r.With(requireConfigEditor).Get("/config/history", cfgH.History)
		r.With(requireConfigEditor).Get("/config/gitops-defaults", cfgH.GitopsDefaults)

		// Structured config builder — CRUD for time intervals, receivers, routes.
		// All endpoints require config-editor role and return {raw_yaml, validation}.
		r.Route("/builder", func(r chi.Router) {
			r.Use(requireConfigEditor)

			// Time intervals
			r.Get("/time-intervals", bldrH.ListTimeIntervals)
			r.Post("/time-intervals/validate", bldrH.ValidateTimeInterval)
			r.Get("/time-intervals/{name}", bldrH.GetTimeInterval)
			r.Put("/time-intervals/{name}", bldrH.UpsertTimeInterval)
			r.Delete("/time-intervals/{name}", bldrH.DeleteTimeInterval)

			// Receivers
			r.Get("/receivers", bldrH.ListReceivers)
			r.Post("/receivers/validate", bldrH.ValidateReceiver)
			r.Get("/receivers/{name}", bldrH.GetReceiver)
			r.Put("/receivers/{name}", bldrH.UpsertReceiver)
			r.Delete("/receivers/{name}", bldrH.DeleteReceiver)
			r.Get("/receivers/{name}/routes", bldrH.ReceiverRoutes)

			// Root route
			r.Get("/route", bldrH.GetRoute)
			r.Put("/route", bldrH.SetRoute)

			// Full config assembly + export (validate-only, no save)
			r.Post("/export", bldrH.ExportConfig)
		})

		// Incidents — immutable-ledger lifecycle tracking (ADR-008).
		//   read  → viewer role minimum
		//   write → silencer role minimum (ACK/INVESTIGATING/RESOLVED require accountability)
		r.Route("/incidents", func(r chi.Router) {
			r.With(requireViewer).Get("/", incH.List)
			r.With(requireSilencer).Post("/", incH.Create)
			r.With(requireViewer).Get("/{id}", incH.Get)
			r.With(requireViewer).Get("/{id}/timeline", incH.Timeline)
			r.With(requireSilencer).Post("/{id}/events", incH.AddEvent)
		})

		// Bulk actions (ADR-007) — silencer role minimum.
		// POST /api/v1/bulk: smart-merge silence/ack for multiple selected alerts.
		r.Route("/v1", func(r chi.Router) {
			r.With(requireSilencer).Post("/bulk", bulkH.Create)
		})
	})

	// ─── SPA fallback ────────────────────────────────────────────────────────
	r.Handle("/*", spaHandler(frontendFS))

	return r
}

// cspMiddlewareWithPolicy sets the Content-Security-Policy and related security
// headers on every HTTP response using the supplied policy string.
func cspMiddlewareWithPolicy(policy string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", policy)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// spaHandler serves the embedded frontend, falling back to index.html for
// client-side routing paths.
// index.html is served with Cache-Control: no-cache so that the browser always
// fetches the latest version (hashed asset filenames ensure long-term caching
// of JS/CSS bundles is safe).
func spaHandler(fs http.FileSystem) http.Handler {
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to open the requested path in the embedded FS.
		// http.FileSystem.Open expects a leading slash; the http.FS adapter
		// handles stripping it before forwarding to the underlying io/fs.FS.
		f, err := fs.Open(r.URL.Path)
		if err != nil {
			// File not found → SPA client-side route, serve index.html.
			r.URL.Path = "/"
		} else {
			_ = f.Close()
		}
		// Prevent the browser from caching index.html; hashed assets are fine.
		if r.URL.Path == "/" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		fileServer.ServeHTTP(w, r)
	})
}

// zapMiddleware returns a chi middleware that logs each request with zap.
func zapMiddleware(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("http",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}
