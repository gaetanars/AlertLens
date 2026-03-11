package configbuilder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// WriteToFile atomically writes content to the given path by writing to a
// temp file in the same directory and then renaming it.
func WriteToFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".alertlens-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		// Clean up temp file if rename failed.
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", path, err)
	}
	return nil
}

// webhookClient is a dedicated HTTP client with a conservative timeout so a
// slow or unresponsive webhook target cannot block the save handler indefinitely.
var webhookClient = &http.Client{Timeout: 10 * time.Second}

// CallWebhook sends a POST request to the given URL with an optional JSON payload.
func CallWebhook(ctx context.Context, webhookURL string, payload []byte) error {
	if webhookURL == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL,
		bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("building webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AlertLens/1.0")

	resp, err := webhookClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling webhook: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("discarding webhook response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
