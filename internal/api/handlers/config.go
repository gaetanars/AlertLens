package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gaetanars/alertlens/internal/alertmanager"
	"github.com/gaetanars/alertlens/internal/configbuilder"
	"github.com/gaetanars/alertlens/internal/gitops"
)

// ConfigHandler handles configuration builder requests.
type ConfigHandler struct {
	pool     *alertmanager.Pool
	ghPusher gitops.Pusher // nil when GitHub is not configured
	glPusher gitops.Pusher // nil when GitLab is not configured
}

// NewConfigHandler creates a ConfigHandler. Pass nil for pushers that are not
// configured; the interface-nil check in Save is correct only when the caller
// passes a nil gitops.Pusher (not a typed-nil concrete pointer).
func NewConfigHandler(pool *alertmanager.Pool, gh, gl gitops.Pusher) *ConfigHandler {
	return &ConfigHandler{pool: pool, ghPusher: gh, glPusher: gl}
}

// Get handles GET /api/config?instance=<name>.
func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	instance := r.URL.Query().Get("instance")
	client := resolveClient(h.pool, w, instance)
	if client == nil {
		return
	}

	status, err := client.GetStatus(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{
		"alertmanager": client.Name(),
		"raw_yaml":     status.Config.Original,
	})
}

// Validate handles POST /api/config/validate.
func (h *ConfigHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RawYAML string `json:"raw_yaml"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result := configbuilder.Validate([]byte(req.RawYAML))
	if !result.Valid {
		writeJSONStatus(w, http.StatusUnprocessableEntity, result)
		return
	}
	writeJSON(w, result)
}

// Diff handles POST /api/config/diff.
func (h *ConfigHandler) Diff(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Alertmanager string `json:"alertmanager"`
		ProposedYAML string `json:"proposed_yaml"`
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

	diff, hasChanges := configbuilder.GenerateDiff(status.Config.Original, req.ProposedYAML)
	writeJSON(w, map[string]any{
		"diff":        diff,
		"has_changes": hasChanges,
	})
}

// saveRequest is the payload for POST /api/config/save.
type saveRequest struct {
	Alertmanager string       `json:"alertmanager"`
	RawYAML      string       `json:"raw_yaml"`
	SaveMode     string       `json:"save_mode"` // "disk" | "github" | "gitlab"
	DiskOptions  *diskOptions `json:"disk_options,omitempty"`
	GitOptions   *gitOptions  `json:"git_options,omitempty"`
	WebhookURL   string       `json:"webhook_url,omitempty"`
}

type diskOptions struct {
	FilePath string `json:"file_path"`
}

type gitOptions struct {
	Repo          string `json:"repo"`
	Branch        string `json:"branch"`
	FilePath      string `json:"file_path"`
	CommitMessage string `json:"commit_message"`
	AuthorName    string `json:"author_name"`
	AuthorEmail   string `json:"author_email"`
}

// Save handles POST /api/config/save.
func (h *ConfigHandler) Save(w http.ResponseWriter, r *http.Request) {
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate config before saving.
	result := configbuilder.Validate([]byte(req.RawYAML))
	if !result.Valid {
		writeJSONStatus(w, http.StatusUnprocessableEntity, map[string]any{
			"error":  "config validation failed",
			"errors": result.Errors,
		})
		return
	}

	content := []byte(req.RawYAML)
	var commitSHA, htmlURL string

	switch req.SaveMode {
	case "disk":
		if req.DiskOptions == nil || req.DiskOptions.FilePath == "" {
			writeError(w, "disk_options.file_path is required for disk save mode", http.StatusBadRequest)
			return
		}
		// SEC-03: prevent path traversal and restrict to YAML files.
		if err := validateFilePath(req.DiskOptions.FilePath); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := configbuilder.WriteToFile(req.DiskOptions.FilePath, content); err != nil {
			writeError(w, fmt.Sprintf("writing config to disk: %s", err), http.StatusInternalServerError)
			return
		}

	case "github":
		if h.ghPusher == nil {
			writeError(w, "GitHub token not configured", http.StatusBadRequest)
			return
		}
		if req.GitOptions == nil {
			writeError(w, "git_options is required for github save mode", http.StatusBadRequest)
			return
		}
		pushResult, err := h.ghPusher.Push(r.Context(), gitops.PushOptions{
			Repo:          req.GitOptions.Repo,
			Branch:        req.GitOptions.Branch,
			FilePath:      req.GitOptions.FilePath,
			CommitMessage: req.GitOptions.CommitMessage,
			AuthorName:    req.GitOptions.AuthorName,
			AuthorEmail:   req.GitOptions.AuthorEmail,
		}, content)
		if err != nil {
			writeError(w, fmt.Sprintf("pushing to GitHub: %s", err), http.StatusBadGateway)
			return
		}
		commitSHA = pushResult.CommitSHA
		htmlURL = pushResult.HTMLURL

	case "gitlab":
		if h.glPusher == nil {
			writeError(w, "GitLab token not configured", http.StatusBadRequest)
			return
		}
		if req.GitOptions == nil {
			writeError(w, "git_options is required for gitlab save mode", http.StatusBadRequest)
			return
		}
		pushResult, err := h.glPusher.Push(r.Context(), gitops.PushOptions{
			Repo:          req.GitOptions.Repo,
			Branch:        req.GitOptions.Branch,
			FilePath:      req.GitOptions.FilePath,
			CommitMessage: req.GitOptions.CommitMessage,
			AuthorName:    req.GitOptions.AuthorName,
			AuthorEmail:   req.GitOptions.AuthorEmail,
		}, content)
		if err != nil {
			writeError(w, fmt.Sprintf("pushing to GitLab: %s", err), http.StatusBadGateway)
			return
		}
		commitSHA = pushResult.CommitSHA
		htmlURL = pushResult.HTMLURL

	default:
		writeError(w, fmt.Sprintf("unknown save_mode: %q (must be disk, github, or gitlab)", req.SaveMode), http.StatusBadRequest)
		return
	}

	// Trigger optional webhook after save.
	if req.WebhookURL != "" {
		// SEC-02: only allow HTTPS webhooks to non-private destinations.
		if err := validateWebhookURL(req.WebhookURL); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		// ERR-01: handle json.Marshal error.
		webhookPayload, err := json.Marshal(map[string]any{
			"alertmanager": req.Alertmanager,
			"save_mode":    req.SaveMode,
			"commit_sha":   commitSHA,
		})
		if err != nil {
			writeError(w, "building webhook payload: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := configbuilder.CallWebhook(r.Context(), req.WebhookURL, webhookPayload); err != nil {
			// Non-fatal: report warning but still return success.
			writeJSON(w, map[string]any{
				"saved":      true,
				"mode":       req.SaveMode,
				"commit_sha": commitSHA,
				"html_url":   htmlURL,
				"warning":    "webhook call failed: " + err.Error(),
			})
			return
		}
	}

	writeJSON(w, map[string]any{
		"saved":      true,
		"mode":       req.SaveMode,
		"commit_sha": commitSHA,
		"html_url":   htmlURL,
	})
}

// validateWebhookURL ensures the URL is HTTPS and does not target a private,
// loopback, or link-local address — including via DNS rebinding.
// SEC-02: SSRF protection.
func validateWebhookURL(rawURL string) error {
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("webhook_url is not a valid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("webhook_url must use HTTPS")
	}
	host := u.Hostname()

	// Reject well-known loopback hostnames before any DNS resolution.
	if host == "localhost" || host == "ip6-localhost" {
		return fmt.Errorf("webhook_url must not target localhost")
	}

	// If the host is a literal IP, check it directly.
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip)
	}

	// Resolve the hostname and verify every returned address.
	// Fail-safe: if DNS resolution fails, reject the URL rather than allowing it.
	addrs, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("webhook_url host %q could not be resolved: %w", host, err)
	}
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil {
			if err := checkIP(ip); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkIP returns an error if the IP is loopback, private, or link-local.
func checkIP(ip net.IP) error {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("webhook_url must not target a private or loopback address")
	}
	return nil
}

// validateFilePath ensures the path is absolute, clean, and has a YAML extension.
// SEC-03: path traversal protection for disk saves.
func validateFilePath(path string) error {
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return fmt.Errorf("disk_options.file_path must be an absolute path")
	}
	ext := strings.ToLower(filepath.Ext(clean))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("disk_options.file_path must have a .yaml or .yml extension")
	}
	return nil
}
