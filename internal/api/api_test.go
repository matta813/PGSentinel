package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	update := `{"name":"db01-renamed","host":"db.internal","port":6432,"user":"monitor-v2","sslMode":"require","tags":["production"]}`
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("PUT", "/api/v1/servers/"+server["id"].(string), strings.NewReader(update)))
	if r.Code != 200 || !strings.Contains(r.Body.String(), "db01-renamed") {
		t.Fatalf("update=%d %s", r.Code, r.Body.String())
	}
	if strings.Contains(r.Body.String(), "password") {
		t.Fatal("credential field leaked from update response")
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

func TestServerPortValidation(t *testing.T) {
	h := testAPI(t)
	for _, port := range []int{-1, 65536} {
		body := fmt.Sprintf(`{"name":"invalid-%d","host":"localhost","port":%d,"user":"monitor","password":"secret"}`, port, port)
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(body)))
		if r.Code != http.StatusUnprocessableEntity || !strings.Contains(r.Body.String(), "Port must be between 1 and 65535") {
			t.Fatalf("create with port %d returned %d: %s", port, r.Code, r.Body.String())
		}
	}

	body := `{"name":"valid","host":"localhost","user":"monitor","password":"secret"}`
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(body)))
	if r.Code != http.StatusCreated {
		t.Fatalf("create valid server=%d %s", r.Code, r.Body.String())
	}
	var server map[string]any
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		t.Fatal(err)
	}
	update := `{"name":"valid","host":"localhost","port":70000,"user":"monitor"}`
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPut, "/api/v1/servers/"+server["id"].(string), strings.NewReader(update)))
	if r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("update with invalid port returned %d: %s", r.Code, r.Body.String())
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

func TestPrometheusMetricsExposeOnlyAggregateState(t *testing.T) {
	h := testAPI(t)
	server := `{"name":"secret-production-name","host":"private.internal","user":"monitor","password":"super-secret"}`
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(server)))
	if r.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if r.Code != http.StatusOK || !strings.HasPrefix(r.Header().Get("Content-Type"), "text/plain") {
		t.Fatalf("metrics=%d content-type=%q", r.Code, r.Header().Get("Content-Type"))
	}
	body := r.Body.String()
	for _, expected := range []string{"pgsentinel_up 1", `pgsentinel_servers{status="unknown"} 1`, "pgsentinel_health_score 100"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %q in metrics:\n%s", expected, body)
		}
	}
	for _, secret := range []string{"secret-production-name", "private.internal", "super-secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("metrics leaked %q", secret)
		}
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

func TestProblemStatusValidation(t *testing.T) {
	h := testAPI(t)
	findingID := "a1b2c3d4e5f6a7b8c9d0e1f2"
	tests := []struct {
		path string
		body string
		want int
	}{
		{path: "/api/v1/problems/not-a-fingerprint/status", body: `{"status":"acknowledged"}`, want: http.StatusBadRequest},
		{path: "/api/v1/problems/" + findingID + "/status", body: `{"status":"resolved"}`, want: http.StatusUnprocessableEntity},
		{path: "/api/v1/problems/" + findingID + "/status", body: `{"status":"acknowledged"}`, want: http.StatusNotFound},
	}
	for _, test := range tests {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPut, test.path, strings.NewReader(test.body)))
		if r.Code != test.want {
			t.Fatalf("%s returned %d, want %d: %s", test.path, r.Code, test.want, r.Body.String())
		}
	}
}

func TestProblemFilterValidation(t *testing.T) {
	h := testAPI(t)
	for _, path := range []string{"/api/v1/problems?status=pending", "/api/v1/problems?severity=urgent", "/api/v1/problems?search=" + strings.Repeat("x", 201)} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s returned %d: %s", path, r.Code, r.Body.String())
		}
	}
}

func TestRejectsOversizedAndMultipleJSONValues(t *testing.T) {
	h := testAPI(t)
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "oversized", body: `{"name":"` + strings.Repeat("x", 70<<10) + `"}`, want: http.StatusRequestEntityTooLarge},
		{name: "multiple values", body: `{"name":"one"} {"name":"two"}`, want: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(test.body)))
			if r.Code != test.want {
				t.Fatalf("got %d, want %d: %s", r.Code, test.want, r.Body.String())
			}
		})
	}
}

func TestNotificationRoutingAPIValidatesReferencesAndNeverReturnsSecrets(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/notifications", strings.NewReader(`{"name":"Pager","provider":"webhook","enabled":true,"webhookUrl":"https://hooks.example/secret-path"}`)))
	if r.Code != http.StatusCreated {
		t.Fatalf("destination=%d %s", r.Code, r.Body.String())
	}
	var destination map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &destination); err != nil {
		t.Fatal(err)
	}
	destinationID := destination["id"].(string)
	if strings.Contains(r.Body.String(), "secret-path") {
		t.Fatal("destination credential leaked")
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(`{"name":"prod","host":"db","user":"monitor","password":"secret","tags":["production"]}`)))
	if r.Code != http.StatusCreated {
		t.Fatalf("server=%d %s", r.Code, r.Body.String())
	}
	var server map[string]any
	_ = json.Unmarshal(r.Body.Bytes(), &server)
	body := fmt.Sprintf(`{"name":"Critical production","enabled":true,"priority":10,"severities":["CRITICAL"],"categories":["Replication"],"serverIds":[%q],"serverTags":["PRODUCTION"],"transitions":["new","resolved"],"destinationIds":[%q],"cooldownSeconds":300}`, server["id"], destinationID)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/notification-routes", strings.NewReader(body)))
	if r.Code != http.StatusCreated || !strings.Contains(r.Body.String(), `"categories":["replication"]`) || !strings.Contains(r.Body.String(), `"serverTags":["production"]`) {
		t.Fatalf("route=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/notification-routes", nil))
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "Critical production") {
		t.Fatalf("list=%d %s", r.Code, r.Body.String())
	}
	invalid := fmt.Sprintf(`{"name":"bad","enabled":true,"priority":10,"severities":["URGENT"],"destinationIds":[%q],"cooldownSeconds":0}`, destinationID)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/notification-routes", strings.NewReader(invalid)))
	if r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid=%d %s", r.Code, r.Body.String())
	}
}

func TestNotificationDeliveryHistoryPaginationIsBounded(t *testing.T) {
	h := testAPI(t)
	for _, path := range []string{"/api/v1/notification-deliveries?limit=201", "/api/v1/notification-deliveries?limit=nope", "/api/v1/notification-deliveries?offset=-1"} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s=%d %s", path, r.Code, r.Body.String())
		}
	}
}

func TestMetricHistoryValidation(t *testing.T) {
	h := testAPI(t)
	serverID := "1b5c3f33-bfcb-4fd4-9c36-df76e2683ee5"
	for _, path := range []string{
		"/api/v1/servers/not-a-uuid/metric-history?name=connections.total",
		"/api/v1/servers/" + serverID + "/metric-history?name=unknown",
		"/api/v1/servers/" + serverID + "/metric-history?name=connections.total&limit=1001",
		"/api/v1/servers/" + serverID + "/metric-history?name=connections.total&from=yesterday",
	} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusBadRequest && r.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s returned %d: %s", path, r.Code, r.Body.String())
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+serverID+"/metric-history?name=connections.total", nil))
	if r.Code != http.StatusOK || strings.TrimSpace(r.Body.String()) != "[]" {
		t.Fatalf("empty history=%d %s", r.Code, r.Body.String())
	}
}

func TestNotificationDestinationAPIKeepsSecretsPrivate(t *testing.T) {
	h := testAPI(t)
	body := `{"name":"Operations","provider":"ntfy","enabled":true,"serverUrl":"https://ntfy.sh","topic":"ops","token":"top-secret"}`
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/notifications", strings.NewReader(body)))
	if r.Code != http.StatusCreated || strings.Contains(r.Body.String(), "top-secret") || strings.Contains(r.Body.String(), "serverUrl") {
		t.Fatalf("create=%d %s", r.Code, r.Body.String())
	}
	var destination map[string]any
	if err := json.NewDecoder(r.Body).Decode(&destination); err != nil {
		t.Fatal(err)
	}
	id := destination["id"].(string)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil))
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "Operations") || strings.Contains(r.Body.String(), "top-secret") {
		t.Fatalf("list=%d %s", r.Code, r.Body.String())
	}
	update := `{"name":"Platform","provider":"webhook","enabled":false,"webhookUrl":"https://example.com/hooks/private"}`
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPut, "/api/v1/notifications/"+id, strings.NewReader(update)))
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "Platform") || strings.Contains(r.Body.String(), "hooks/private") {
		t.Fatalf("update=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/"+id, nil))
	if r.Code != http.StatusNoContent {
		t.Fatalf("delete=%d %s", r.Code, r.Body.String())
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
