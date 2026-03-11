package alertmanager

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alertlens/alertlens/internal/config"
)

// silencePath returns the Alertmanager API v2 path for a specific silence.
// url.PathEscape is used to prevent path injection when the caller-supplied
// id contains characters such as '/', '?', or '#' (CWE-73).
func silencePath(id string) string {
	return "/api/v2/silence/" + url.PathEscape(id)
}

const (
	labelAckType    = "alertlens_ack_type"
	labelAckBy      = "alertlens_ack_by"
	labelAckComment = "alertlens_ack_comment"
	ackTypeVisual   = "visual"
)

// Client is an Alertmanager API v2 client for a single instance.
type Client struct {
	name      string
	baseURL   string
	tenantID  string
	basicAuth *basicAuthCreds
	http      *http.Client
}

type basicAuthCreds struct {
	username, password string
}

// NewClient creates a new Client from an AlertmanagerConfig.
func NewClient(cfg config.AlertmanagerConfig) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.TLSSkipVerify {
		// Only allow InsecureSkipVerify if explicitly configured via an environment variable or flag,
		// and log a security warning.
		if os.Getenv("ALERT_LENS_ALLOW_INSECURE_TLS") == "true" {
			log.Println("WARNING: TLS InsecureSkipVerify is enabled. This is insecure.")
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		}
	}

	c := &Client{
		name:    cfg.Name,
		baseURL: strings.TrimRight(cfg.URL, "/"),
		tenantID: cfg.TenantID,
		http: &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		},
	}
	if cfg.BasicAuth.Username != "" {
		c.basicAuth = &basicAuthCreds{
			username: cfg.BasicAuth.Username,
			password: cfg.BasicAuth.Password,
		}
	}
	return c
}

// Name returns the configured name for this instance.
func (c *Client) Name() string { return c.name }

// GetAlerts fetches alerts from the Alertmanager instance.
func (c *Client) GetAlerts(ctx context.Context, params AlertsQueryParams) ([]Alert, error) {
	q := url.Values{}
	for _, f := range params.Filter {
		q.Add("filter", f)
	}
	if params.Silenced {
		q.Set("silenced", "true")
	}
	if params.Inhibited {
		q.Set("inhibited", "true")
	}
	if params.Active {
		q.Set("active", "true")
	}

	endpoint := "/api/v2/alerts"
	if len(q) > 0 {
		endpoint += "?" + q.Encode()
	}

	var alerts []Alert
	if err := c.get(ctx, endpoint, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

// GetSilences fetches all silences from the Alertmanager instance.
func (c *Client) GetSilences(ctx context.Context) ([]Silence, error) {
	var silences []Silence
	if err := c.get(ctx, "/api/v2/silences", &silences); err != nil {
		return nil, err
	}
	return silences, nil
}

// GetSilence fetches a single silence by ID.
func (c *Client) GetSilence(ctx context.Context, id string) (*Silence, error) {
	var silence Silence
	if err := c.get(ctx, silencePath(id), &silence); err != nil {
		return nil, err
	}
	return &silence, nil
}

// CreateSilence creates a new silence and returns its ID.
func (c *Client) CreateSilence(ctx context.Context, input SilenceInput) (string, error) {
	var resp struct {
		SilenceID string `json:"silenceID"`
	}
	if err := c.postJSON(ctx, "/api/v2/silences", input, &resp); err != nil {
		return "", err
	}
	return resp.SilenceID, nil
}

// UpdateSilence updates an existing silence.
func (c *Client) UpdateSilence(ctx context.Context, id string, input SilenceInput) (string, error) {
	input.ID = id
	var resp struct {
		SilenceID string `json:"silenceID"`
	}
	if err := c.postJSON(ctx, "/api/v2/silences", input, &resp); err != nil {
		return "", err
	}
	return resp.SilenceID, nil
}

// ExpireSilence expires a silence by ID.
func (c *Client) ExpireSilence(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		c.baseURL+silencePath(id), nil)
	if err != nil {
		return err
	}
	c.setHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE silence %s: %w", id, err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("discarding response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DELETE silence %s: unexpected status %d", id, resp.StatusCode)
	}
	return nil
}

// GetStatus fetches the Alertmanager status including version and config.
func (c *Client) GetStatus(ctx context.Context) (*AMStatus, error) {
	var status AMStatus
	if err := c.get(ctx, "/api/v2/status", &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// CreateAck creates a visual-ack silence with the special AlertLens labels.
func (c *Client) CreateAck(ctx context.Context, by, comment string, input SilenceInput) (string, error) {
	// Inject the visual ack labels into matchers
	input.Matchers = append(input.Matchers,
		Matcher{Name: labelAckType, Value: ackTypeVisual, IsRegex: false, IsEqual: true},
		Matcher{Name: labelAckBy, Value: by, IsRegex: false, IsEqual: true},
		Matcher{Name: labelAckComment, Value: comment, IsRegex: false, IsEqual: true},
	)
	return c.CreateSilence(ctx, input)
}

// IsAckSilence returns true if the silence is an AlertLens visual ack.
func IsAckSilence(s Silence) bool {
	for _, m := range s.Matchers {
		if m.Name == labelAckType && m.Value == ackTypeVisual {
			return true
		}
	}
	return false
}

// ExtractAckInfo reads AlertLens ack metadata from a silence's matchers.
func ExtractAckInfo(s Silence) (by, comment string) {
	for _, m := range s.Matchers {
		switch m.Name {
		case labelAckBy:
			by = m.Value
		case labelAckComment:
			comment = m.Value
		}
	}
	return
}

// ─── HTTP helpers ────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request for %s: %w", path, err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %d", path, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decoding response from %s: %w", path, err)
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, path string, body, dest any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path,
		bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("building request for %s: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("POST %s: unexpected status %d", path, resp.StatusCode)
	}

	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			return fmt.Errorf("decoding response from %s: %w", path, err)
		}
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.username, c.basicAuth.password)
	}
	if c.tenantID != "" {
		req.Header.Set("X-Scope-OrgID", c.tenantID)
	}
}
