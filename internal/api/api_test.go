package api

import (
	"bytes"
	"encoding/json"
	"github.com/matta813/pgsentinel/internal/storage"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func testAPI(t *testing.T) http.Handler {
	t.Helper()
	s, err := storage.Open(filepath.Join(t.TempDir(), "api.db"), "a sufficiently long api test key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return New(s, slog.New(slog.NewTextHandler(io.Discard, nil))).Handler()
}
func TestHealthAndServerAPI(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/health", nil))
	if r.Code != 200 {
		t.Fatalf("health=%d", r.Code)
	}
	body := `{"name":"db01","host":"localhost","port":5432,"user":"monitor","password":"super-secret","sslMode":"disable","tags":["test"]}`
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("POST", "/api/v1/servers", strings.NewReader(body)))
	if r.Code != 201 {
		t.Fatalf("create=%d %s", r.Code, r.Body.String())
	}
	if strings.Contains(r.Body.String(), "super-secret") {
		t.Fatal("credential leaked")
	}
	var server map[string]any
	if err := json.NewDecoder(bytes.NewReader(r.Body.Bytes())).Decode(&server); err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/v1/servers", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), server["id"].(string)) {
		t.Fatalf("list=%d %s", r.Code, r.Body.String())
	}
}

func TestVersionAPI(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/v1/version", nil))
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"version":"dev"`) || !strings.Contains(r.Body.String(), `"commit":"unknown"`) {
		t.Fatalf("version=%d %s", r.Code, r.Body.String())
	}
}
func TestRejectsUnknownFields(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("POST", "/api/v1/servers", strings.NewReader(`{"name":"x","unexpected":true}`)))
	if r.Code != 400 {
		t.Fatalf("got %d", r.Code)
	}
}
