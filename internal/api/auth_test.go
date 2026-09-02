package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matta813/pgsentinel/internal/auth"
	"github.com/matta813/pgsentinel/internal/storage"
)

func TestAuthenticationRequiredAndLoginLifecycle(t *testing.T) {
	s, err := storage.Open(filepath.Join(t.TempDir(), "auth.db"), "a sufficiently long api test key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	manager, err := auth.New(auth.Config{Store: s, Username: "admin", Password: "a-secure-test-password"})
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
	handler.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"a-secure-test-password"}`)))
	if r.Code != http.StatusOK || len(r.Result().Cookies()) != 1 {
		t.Fatalf("login=%d %s", r.Code, r.Body.String())
	}
	cookie := r.Result().Cookies()[0]
	if !strings.Contains(r.Body.String(), `"mustChangePassword":true`) || !strings.Contains(r.Body.String(), `"username":"admin"`) {
		t.Fatalf("login state=%s", r.Body.String())
	}
	blocked := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	blocked.AddCookie(cookie)
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, blocked)
	if r.Code != http.StatusForbidden {
		t.Fatalf("first-login access=%d %s", r.Code, r.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	req.AddCookie(cookie)
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, req)
	if r.Code != http.StatusOK {
		t.Fatalf("session=%d", r.Code)
	}
	change := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", strings.NewReader(`{"currentPassword":"a-secure-test-password","newPassword":"a-new-secure-password"}`))
	change.AddCookie(cookie)
	r = httptest.NewRecorder()
	handler.ServeHTTP(r, change)
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"mustChangePassword":false`) {
		t.Fatalf("password change=%d %s", r.Code, r.Body.String())
	}
	viewer, err := manager.CreateUser(context.Background(), "viewer", "viewer-initial-password", "viewer")
	if err != nil {
		t.Fatal(err)
	}
	viewerResponse := httptest.NewRecorder()
	if err := manager.Start(viewerResponse, viewer); err != nil {
		t.Fatal(err)
	}
	viewerCookie := viewerResponse.Result().Cookies()[0]
	viewerChange := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", nil)
	viewerChange.AddCookie(viewerCookie)
	if err := manager.ChangePassword(context.Background(), viewerChange, "viewer-initial-password", "viewer-replacement-password"); err != nil {
		t.Fatal(err)
	}
	for _, boundary := range []struct {
		method, path string
		want         int
	}{{http.MethodGet, "/api/v1/servers", http.StatusOK}, {http.MethodPost, "/api/v1/servers", http.StatusForbidden}, {http.MethodPut, "/api/v1/problems/0123456789abcdef01234567/status", http.StatusForbidden}, {http.MethodGet, "/api/v1/users", http.StatusForbidden}, {http.MethodGet, "/api/v1/audit-events", http.StatusForbidden}} {
		request := httptest.NewRequest(boundary.method, boundary.path, strings.NewReader(`{}`))
		request.AddCookie(viewerCookie)
		r = httptest.NewRecorder()
		handler.ServeHTTP(r, request)
		if r.Code != boundary.want {
			t.Errorf("viewer %s %s=%d want %d: %s", boundary.method, boundary.path, r.Code, boundary.want, r.Body.String())
		}
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
