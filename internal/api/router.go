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
	"github.com/alertlens/alertlens/internal/gitops"
)

// maxRequestBodyBytes is the hard cap on incoming request bodies (10 MiB).
// Alertmanager configs are typically well under 1 MiB; this protects against
// trivial OOM attacks while leaving plenty of headroom.
const maxRequestBodyBytes = 10 << 20 // 10 MiB

// cspPolicy is the Content-Security-Policy header value applied to every
// response.  It prevents XSS exploitation even if an attacker finds a
// reflection point, by:
//   - Restricting scripts to same-origin only (no inline, no CDN).
//   - Allowing inline styles (required by SvelteKit / Tailwind).
//   - Blocking <object>, <embed> and <frame*> entirely.
//   - Pinning base-uri and form-action to 'self'.
const cspPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none';"

// NewRouter wires all API routes and returns the root http.Handler.
// The frontendFS is served for all non-API routes (SPA fallback).
// ghPusher / glPusher must be nil gitops.Pusher (not typed-nil) when the
// corresponding forge is not configured, so interface nil-checks in handlers work.
// secureCookies controls the Secure attribute on the CSRF cookie; set to true
// when the application is served behind HTTPS.
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
) http.Handler {
	r := chi.NewRouter()

	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	// SEC-CWE-321: derive the CSRF secret from the auth service's JWT signing
	// key rather than using a hardcoded constant.  The derived key is unique to
	// CSRF purposes and rotates whenever the admin password changes.
	csrfSecret := authSvc.CSRFSecret()

	loginRL := auth.NewLoginRateLimiter()

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
	r.Use(cspMiddleware)
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
	healthH := handlers.NewHealthHandler(version)
	authH   := handlers.NewAuthHandler(authSvc)
	amH     := handlers.NewAlertmanagersHandler(pool)
	alH     := handlers.NewAlertsHandler(pool)
	silH    := handlers.NewSilencesHandler(pool)
	rtH     := handlers.NewRoutingHandler(pool)
	cfgH    := handlers.NewConfigHandler(pool, ghPusher, glPusher)

	// ─── Role middleware shorthands ──────────────────────────────────────────
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

		// Alerts (read-only, public)
		r.Get("/alerts", alH.List)

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
	})

	// ─── SPA fallback ────────────────────────────────────────────────────────
	r.Handle("/*", spaHandler(frontendFS))

	return r
}

// cspMiddleware sets the Content-Security-Policy and related security headers
// on every HTTP response.
func cspMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", cspPolicy)
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
			f.Close()
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
