package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/alertlens/alertlens/internal/alertmanager"
	"github.com/alertlens/alertlens/internal/api"
	"github.com/alertlens/alertlens/internal/auth"
	"github.com/alertlens/alertlens/internal/config"
	"github.com/alertlens/alertlens/internal/gitops"
	"github.com/alertlens/alertlens/internal/incident"
)

// version is injected at build time via -ldflags.
var version = "dev"

//go:embed all:dist
var distFS embed.FS

func main() {
	configPath := flag.String("config", "", "path to alertlens.yaml config file")
	flag.Parse()

	// ─── Logger ──────────────────────────────────────────────────────────────
	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	// ─── Config ──────────────────────────────────────────────────────────────
	cfg, cfgWarnings, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}
	for _, w := range cfgWarnings {
		logger.Warn("config", zap.String("warning", w))
	}
	logger.Info("AlertLens starting",
		zap.String("version", version),
		zap.String("host", cfg.Server.Host),
		zap.Int("port", cfg.Server.Port),
		zap.Int("alertmanagers", len(cfg.Alertmanagers)),
	)

	// SPEC-06: warn when TLS verification is disabled for any instance.
	for _, am := range cfg.Alertmanagers {
		if am.TLSSkipVerify {
			logger.Warn("TLS certificate verification disabled",
				zap.String("instance", am.Name),
				zap.String("url", am.URL),
			)
		}
	}

	// ─── Alertmanager pool ───────────────────────────────────────────────────
	pool := alertmanager.NewPool(cfg.Alertmanagers, logger)

	// ─── Auth service ────────────────────────────────────────────────────────
	authSvc := auth.NewServiceFromConfig(cfg.Auth, logger.Sugar())
	if authSvc.AdminEnabled() {
		logger.Info("admin mode enabled")
	} else {
		logger.Info("admin mode disabled (no password configured)")
	}

	// ─── GitOps clients ──────────────────────────────────────────────────────
	// Declare as interface types so that a nil value is a true interface nil,
	// not a typed-nil concrete pointer (which would pass a non-nil interface to
	// handlers and break the nil-check in ConfigHandler.Save).
	var ghPusher gitops.Pusher
	if cfg.Gitops.GitHub.Token != "" {
		ghPusher = gitops.NewGitHubPusher(cfg.Gitops.GitHub.Token)
		logger.Info("GitHub gitops enabled")
	}

	var glPusher gitops.Pusher
	if cfg.Gitops.GitLab.Token != "" {
		p, glErr := gitops.NewGitLabPusher(cfg.Gitops.GitLab.Token, cfg.Gitops.GitLab.URL)
		if glErr != nil {
			logger.Warn("failed to initialize GitLab client", zap.Error(glErr))
		} else {
			glPusher = p
			logger.Info("GitLab gitops enabled", zap.String("url", cfg.Gitops.GitLab.URL))
		}
	}

	// ─── Frontend ────────────────────────────────────────────────────────────
	// COR-03: strip the "dist/" prefix so the SPA handler serves paths correctly.
	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		logger.Fatal("failed to sub embedded FS", zap.Error(err))
	}
	frontendFS := http.FS(subFS)

	// ─── Incident store (in-memory immutable ledger) ─────────────────────────
	incidentStore := incident.NewStore()

	// ─── HTTP router ─────────────────────────────────────────────────────────
	router := api.NewRouter(pool, authSvc, ghPusher, glPusher, frontendFS, cfg.Server.CORSAllowedOrigins, cfg.Server.SecureCookies, version, logger, incidentStore)

	// ─── HTTP server ─────────────────────────────────────────────────────────
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ─── Graceful shutdown ───────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("HTTP server listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down gracefully…")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown error", zap.Error(err))
		os.Exit(1)
	}
	logger.Info("server stopped")
}
