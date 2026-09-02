package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matta813/pgsentinel/internal/storage"
)

func TestServeFrontendDoesNotUseSPAFallbackForAPIRequests(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "index.html"), []byte("<!doctype html>"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := &API{mux: http.NewServeMux()}
	a.ServeFrontend(directory)

	r := httptest.NewRecorder()
	a.mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil))
	if r.Code != http.StatusNotFound || r.Header().Get("Content-Type") != "application/json" || !strings.Contains(r.Body.String(), "API endpoint not found") {
		t.Fatalf("unexpected API fallback: status=%d content-type=%q body=%q", r.Code, r.Header().Get("Content-Type"), r.Body.String())
	}
}

func TestFrontendDoesNotServeFilesOutsideRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "web")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("frontend-index"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "asset.txt"), []byte("public-asset"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, "secret.txt"), []byte("must-not-leak"), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := storage.Open(filepath.Join(t.TempDir(), "frontend.db"), "a sufficiently long frontend test key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	app := New(s, slog.New(slog.NewTextHandler(io.Discard, nil)))
	app.ServeFrontend(root)

	for _, requestPath := range []string{"/../secret.txt", "/%2e%2e/secret.txt"} {
		r := httptest.NewRecorder()
		app.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if strings.Contains(r.Body.String(), "must-not-leak") {
			t.Fatalf("path %q exposed a file outside the frontend root", requestPath)
		}
	}

	r := httptest.NewRecorder()
	app.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/asset.txt", nil))
	if r.Code != http.StatusOK || r.Body.String() != "public-asset" {
		t.Fatalf("asset response=%d %q", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	app.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/client-side-route", nil))
	if r.Code != http.StatusOK || r.Body.String() != "frontend-index" {
		t.Fatalf("fallback response=%d %q", r.Code, r.Body.String())
	}
}
