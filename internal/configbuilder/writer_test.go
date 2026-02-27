package configbuilder

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ─── WriteToFile ──────────────────────────────────────────────────────────────

func TestWriteToFile_CreatesFileWithContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alertmanager.yaml")
	content := []byte("route:\n  receiver: 'null'\n")

	if err := WriteToFile(path, content); err != nil {
		t.Fatalf("WriteToFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch:\n  want: %q\n  got:  %q", content, got)
	}
}

func TestWriteToFile_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := WriteToFile(path, []byte("old content")); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := WriteToFile(path, []byte("new content")); err != nil {
		t.Fatalf("second write: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != "new content" {
		t.Errorf("expected 'new content', got %q", got)
	}
}

func TestWriteToFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")

	if err := WriteToFile(path, []byte{}); err != nil {
		t.Fatalf("WriteToFile with empty content: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("expected empty file, got size %d", info.Size())
	}
}

func TestWriteToFile_InvalidDirectory(t *testing.T) {
	path := "/nonexistent/dir/config.yaml"
	err := WriteToFile(path, []byte("content"))
	if err == nil {
		t.Error("expected error for non-existent parent directory")
	}
}

func TestWriteToFile_AtomicWrite_NoPartialFile(t *testing.T) {
	// After a successful write the temp file should be gone (renamed).
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	_ = WriteToFile(path, []byte("hello"))

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "config.yaml" {
			t.Errorf("unexpected leftover file: %s", e.Name())
		}
	}
}

// ─── CallWebhook ──────────────────────────────────────────────────────────────

func TestCallWebhook_EmptyURL_ReturnsNil(t *testing.T) {
	err := CallWebhook(context.Background(), "", []byte("{}"))
	if err != nil {
		t.Errorf("expected nil error for empty URL, got: %v", err)
	}
}

func TestCallWebhook_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := CallWebhook(context.Background(), srv.URL, []byte(`{"key":"value"}`))
	if err != nil {
		t.Errorf("expected no error for 200 response, got: %v", err)
	}
}

func TestCallWebhook_ServerReturns4xx_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	err := CallWebhook(context.Background(), srv.URL, []byte("{}"))
	if err == nil {
		t.Error("expected error for 400 response")
	}
}

func TestCallWebhook_ServerReturns5xx_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := CallWebhook(context.Background(), srv.URL, []byte("{}"))
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestCallWebhook_ServerReturns201_NoError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := CallWebhook(context.Background(), srv.URL, []byte("{}"))
	if err != nil {
		t.Errorf("expected no error for 201 response, got: %v", err)
	}
}

func TestCallWebhook_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := CallWebhook(ctx, srv.URL, []byte("{}"))
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestCallWebhook_NilPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := CallWebhook(context.Background(), srv.URL, nil)
	if err != nil {
		t.Errorf("expected no error for nil payload, got: %v", err)
	}
}
