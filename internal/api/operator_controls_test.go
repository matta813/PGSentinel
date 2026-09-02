package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOperatorControlsAPIValidationAndLifecycle(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(`{"name":"prod","host":"db","user":"monitor","password":"secret","tags":["production"]}`)))
	if r.Code != http.StatusCreated {
		t.Fatalf("server=%d %s", r.Code, r.Body.String())
	}
	var server map[string]any
	_ = json.Unmarshal(r.Body.Bytes(), &server)
	serverID := server["id"].(string)
	now := time.Now().UTC()
	valid := fmt.Sprintf(`{"description":"Planned failover","serverId":%q,"category":"Replication","startsAt":%q,"endsAt":%q}`, serverID, now.Add(time.Minute).Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339))
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/maintenance-windows", strings.NewReader(valid)))
	if r.Code != http.StatusCreated || !strings.Contains(r.Body.String(), `"state":"upcoming"`) {
		t.Fatalf("maintenance=%d %s", r.Code, r.Body.String())
	}
	for _, request := range []struct{ path, body string }{{"/api/v1/maintenance-windows", fmt.Sprintf(`{"description":"global","startsAt":%q,"endsAt":%q}`, now.Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339))}, {"/api/v1/suppressions", fmt.Sprintf(`{"ruleId":"blocking-queries","reason":"global silence","expiresAt":%q}`, now.Add(time.Hour).Format(time.RFC3339))}, {"/api/v1/threshold-overrides", `{"ruleId":"standby-replay-lag","scopeType":"global","value":999999,"reason":"disable"}`}} {
		r = httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, request.path, strings.NewReader(request.body)))
		if r.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s=%d %s", request.path, r.Code, r.Body.String())
		}
	}
	threshold := fmt.Sprintf(`{"ruleId":"standby-replay-lag","scopeType":"server","scopeValue":%q,"value":120,"reason":"Delayed reporting replica"}`, serverID)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/threshold-overrides", strings.NewReader(threshold)))
	if r.Code != http.StatusCreated {
		t.Fatalf("threshold=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/threshold-overrides", nil))
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), "Delayed reporting replica") || strings.Contains(r.Body.String(), "password") {
		t.Fatalf("list=%d %s", r.Code, r.Body.String())
	}
	for _, request := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/v1/auth/login", `{"username":"admin","password":"wrong-secret-password"}`},
		{http.MethodPost, "/api/v1/notifications", `{"name":"pager","provider":"webhook","enabled":true,"webhookUrl":"https://hooks.example/super-secret-token"}`},
	} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(request.method, request.path, strings.NewReader(request.body)))
		if request.path == "/api/v1/auth/login" {
			if r.Code != http.StatusUnauthorized {
				t.Fatalf("failed login=%d %s", r.Code, r.Body.String())
			}
		} else if r.Code != http.StatusCreated {
			t.Fatalf("%s=%d %s", request.path, r.Code, r.Body.String())
		}
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/audit-events?limit=20", nil))
	body := r.Body.String()
	if r.Code != http.StatusOK || !strings.Contains(body, "auth.login.failed") || !strings.Contains(body, "server.created") || !strings.Contains(body, "notification_destination.created") || !strings.Contains(body, "threshold_override.created") {
		t.Fatalf("audit=%d %s", r.Code, body)
	}
	for _, secret := range []string{"wrong-secret-password", "super-secret-token", "hooks.example", "Delayed reporting replica"} {
		if strings.Contains(body, secret) {
			t.Fatalf("audit leaked %q: %s", secret, body)
		}
	}
	for _, path := range []string{"/api/v1/audit-events?limit=101", "/api/v1/audit-events?offset=10001", "/api/v1/audit-events?from=yesterday", "/api/v1/audit-events?from=2026-08-26T00:00:00Z&to=2026-08-25T00:00:00Z", "/api/v1/audit-events?search=" + strings.Repeat("x", 201)} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusUnprocessableEntity {
			t.Fatalf("%s=%d %s", path, r.Code, r.Body.String())
		}
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/users", strings.NewReader(`{"username":"readonly","password":"temporary-secret-password","role":"viewer"}`)))
	if r.Code != http.StatusCreated || strings.Contains(r.Body.String(), "temporary-secret-password") || strings.Contains(r.Body.String(), "passwordHash") {
		t.Fatalf("create user=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if r.Code != http.StatusOK || !strings.Contains(r.Body.String(), `"role":"viewer"`) || strings.Contains(r.Body.String(), "passwordHash") {
		t.Fatalf("list users=%d %s", r.Code, r.Body.String())
	}
}
