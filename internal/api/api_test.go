package api

import (
	"bytes"
	"encoding/json"
	"github.com/matta813/pgsentinel/internal/auth"
	"github.com/matta813/pgsentinel/internal/storage"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
	manager, err := auth.New(auth.Config{Password: "a-secure-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(s, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{Auth: manager}).Handler()
	capture := httptest.NewRecorder()
	if err := manager.Start(capture); err != nil {
		t.Fatal(err)
	}
	cookie := capture.Result().Cookies()[0]
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && r.URL.Path != "/api/v1/auth/login" && r.URL.Path != "/api/v1/version" {
			r.AddCookie(cookie)
		}
		handler.ServeHTTP(w, r)
	})
}

func TestAuthenticationRequiredAndLoginLifecycle(t *testing.T) {
	s, err := storage.Open(filepath.Join(t.TempDir(), "auth.db"), "a sufficiently long api test key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	manager, err := auth.New(auth.Config{Password: "a-secure-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(s, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{Auth: manager}).Handler()

	r := httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil))
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated=%d", r.Code)
	}
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"password":"a-secure-test-password"}`)))
	if r.Code != http.StatusOK || len(r.Result().Cookies()) != 1 {
		t.Fatalf("login=%d %s", r.Code, r.Body.String())
	}
	cookie := r.Result().Cookies()[0]
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	req.AddCookie(cookie)
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, req)
	if r.Code != http.StatusOK {
		t.Fatalf("session=%d", r.Code)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(cookie)
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, req)
	if r.Code != http.StatusOK {
		t.Fatalf("logout=%d", r.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	req.AddCookie(cookie)
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, req)
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("logged-out session=%d", r.Code)
	}
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

func TestServerTagsAreNormalizedAndFilterable(t *testing.T) {
	h := testAPI(t)
	servers := []string{
		`{"name":"production","host":"prod","user":"monitor","password":"secret","tags":[" Production ","EU","production",""]}`,
		`{"name":"staging","host":"stage","user":"monitor","password":"secret","tags":["staging"]}`,
	}
	for _, body := range servers {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(body)))
		if r.Code != http.StatusCreated {
			t.Fatalf("create=%d %s", r.Code, r.Body.String())
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/servers?tag=production", nil))
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"name":"production"`) || strings.Contains(r.Body.String(), `"name":"staging"`) {
		t.Fatalf("filtered=%d %s", r.Code, r.Body.String())
	}
	if strings.Count(r.Body.String(), "Production") != 1 || !strings.Contains(r.Body.String(), `"tags":["EU","Production"]`) {
		t.Fatalf("tags were not normalized: %s", r.Body.String())
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
