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

func testAPI(t *testing.T) http.Handler {
	t.Helper()
	s, err := storage.Open(filepath.Join(t.TempDir(), "api.db"), "a sufficiently long api test key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	manager, err := auth.New(auth.Config{Store: s, Username: "admin", Password: "a-secure-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(s, slog.New(slog.NewTextHandler(io.Discard, nil)), Options{Auth: manager}).Handler()
	capture := httptest.NewRecorder()
	user, err := manager.Authenticate(context.Background(), "admin", "a-secure-test-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(capture, user); err != nil {
		t.Fatal(err)
	}
	cookie := capture.Result().Cookies()[0]
	change := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", nil)
	change.AddCookie(cookie)
	if err := manager.ChangePassword(context.Background(), change, "a-secure-test-password", "a-replacement-test-password"); err != nil {
		t.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && r.URL.Path != "/api/v1/auth/login" && r.URL.Path != "/api/v1/version" {
			r.AddCookie(cookie)
		}
		handler.ServeHTTP(w, r)
	})
}
