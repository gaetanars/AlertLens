package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure for AlertLens.
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Auth          AuthConfig          `yaml:"auth"`
	Alertmanagers []AlertmanagerConfig `yaml:"alertmanagers"`
	Gitops        GitopsConfig        `yaml:"gitops"`
}

type ServerConfig struct {
	Host               string   `yaml:"host"                env:"ALERTLENS_SERVER_HOST"`
	Port               int      `yaml:"port"                env:"ALERTLENS_SERVER_PORT"`
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`
	// SecureCookies controls the Secure attribute on session-related cookies
	// (e.g. the CSRF token cookie).  Set to true when AlertLens is served over
	// HTTPS; leave false for plain HTTP (development / internal deployments).
	SecureCookies bool `yaml:"secure_cookies" env:"ALERTLENS_SERVER_SECURE_COOKIES"`
}

type AuthConfig struct {
	AdminPassword string       `yaml:"admin_password" env:"ALERTLENS_AUTH_ADMIN_PASSWORD"`
	Users         []UserConfig `yaml:"users"`
}

// UserConfig defines an additional role-bound user credential.
// The Role field must be one of: viewer, silencer, config-editor, admin.
// TOTPSecret, if non-empty, enables TOTP-based MFA for this user.  The value
// must be a base32-encoded secret compatible with standard authenticator apps.
type UserConfig struct {
	Password   string `yaml:"password"`
	Role       string `yaml:"role"`
	TOTPSecret string `yaml:"totp_secret,omitempty"`
}

type AlertmanagerConfig struct {
	Name           string    `yaml:"name"`
	URL            string    `yaml:"url"`
	BasicAuth      BasicAuth `yaml:"basic_auth"`
	TenantID       string    `yaml:"tenant_id"`
	TLSSkipVerify  bool      `yaml:"tls_skip_verify"`
	ConfigFilePath string    `yaml:"config_file_path"`
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type GitopsConfig struct {
	GitHub GitHubConfig `yaml:"github"`
	GitLab GitLabConfig `yaml:"gitlab"`
}

type GitHubConfig struct {
	Token string `yaml:"token" env:"ALERTLENS_GITOPS_GITHUB_TOKEN"`
}

type GitLabConfig struct {
	Token string `yaml:"token" env:"ALERTLENS_GITOPS_GITLAB_TOKEN"`
	URL   string `yaml:"url"   env:"ALERTLENS_GITOPS_GITLAB_URL"`
}

// defaults returns a Config populated with sensible default values.
func defaults() Config {
	return Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 9000,
		},
		Gitops: GitopsConfig{
			GitLab: GitLabConfig{
				URL: "https://gitlab.com",
			},
		},
		Alertmanagers: []AlertmanagerConfig{
			{
				Name: "default",
				URL:  "http://localhost:9093",
			},
		},
	}
}

// Load reads configuration from the given YAML file (if non-empty) and then
// applies environment variable overrides. Env vars always take precedence.
// The second return value contains non-fatal warnings (e.g. unparseable env
// vars that were ignored); callers should log them with their preferred logger.
func Load(path string) (*Config, []string, error) {
	cfg := defaults()

	if path != "" {
		// Sanitize path to prevent directory traversal attacks.
		// Use filepath.Clean to normalize the path and reject traversal attempts.
		cleanPath := filepath.Clean(path)
		if strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "/..") {
			return nil, nil, fmt.Errorf("invalid path: directory traversal detected in %q", path)
		}
		data, err := os.ReadFile(cleanPath)
		if err != nil {
			return nil, nil, fmt.Errorf("reading config file %q: %w", path, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, nil, fmt.Errorf("parsing config file %q: %w", path, err)
		}
	}

	warnings := applyEnvOverrides(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, warnings, nil
}

// applyEnvOverrides walks the config struct tree and applies environment
// variable overrides where the `env` struct tag is defined.
// It returns a (possibly empty) slice of non-fatal warning messages.
func applyEnvOverrides(cfg *Config) []string {
	var warnings []string
	applyEnvToStruct(reflect.ValueOf(cfg).Elem(), &warnings)
	return warnings
}

func applyEnvToStruct(v reflect.Value, warnings *[]string) {
	t := v.Type()
	for i := range t.NumField() {
		field := t.Field(i)
		fv := v.Field(i)

		if fv.Kind() == reflect.Struct {
			applyEnvToStruct(fv, warnings)
			continue
		}

		envKey := field.Tag.Get("env")
		if envKey == "" {
			continue
		}

		envVal, ok := os.LookupEnv(envKey)
		if !ok || envVal == "" {
			continue
		}

		switch fv.Kind() {
		case reflect.String:
			fv.SetString(envVal)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(envVal, 10, 64)
			if err != nil {
				*warnings = append(*warnings,
					fmt.Sprintf("env %s=%q is not a valid integer, ignoring", envKey, envVal))
			} else {
				fv.SetInt(n)
			}
		case reflect.Bool:
			b, err := strconv.ParseBool(envVal)
			if err != nil {
				*warnings = append(*warnings,
					fmt.Sprintf("env %s=%q is not a valid boolean, ignoring", envKey, envVal))
			} else {
				fv.SetBool(b)
			}
		}
	}
}

func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", cfg.Server.Port)
	}

	// Validate admin password constraints (bcrypt limit + non-empty).
	if cfg.Auth.AdminPassword != "" {
		if len([]byte(cfg.Auth.AdminPassword)) > 72 {
			return fmt.Errorf("auth.admin_password exceeds bcrypt's 72-byte limit (%d bytes); please use a shorter password",
				len([]byte(cfg.Auth.AdminPassword)))
		}
	}

	// Validate additional users' passwords.
	for i, u := range cfg.Auth.Users {
		if u.Password == "" {
			return fmt.Errorf("auth.users[%d].password cannot be empty", i)
		}
		if len([]byte(u.Password)) > 72 {
			return fmt.Errorf("auth.users[%d].password exceeds bcrypt's 72-byte limit (%d bytes); please use a shorter password",
				i, len([]byte(u.Password)))
		}
	}

	for i, am := range cfg.Alertmanagers {
		if strings.TrimSpace(am.URL) == "" {
			return fmt.Errorf("alertmanagers[%d].url is required", i)
		}
		if strings.TrimSpace(am.Name) == "" {
			return fmt.Errorf("alertmanagers[%d].name is required", i)
		}
		u, err := url.ParseRequestURI(am.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return fmt.Errorf("alertmanagers[%d].url %q is not a valid http/https URL", i, am.URL)
		}
	}
	return nil
}
