package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
)

func TestAPIRoutesRemainRegistered(t *testing.T) {
	a := New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	routes := []struct{ method, path string }{
		{"GET", "/health"}, {"GET", "/ready"}, {"GET", "/metrics"}, {"GET", "/api/v1/version"},
		{"POST", "/api/v1/auth/login"}, {"GET", "/api/v1/auth/session"}, {"POST", "/api/v1/auth/logout"}, {"PUT", "/api/v1/auth/password"},
		{"GET", "/api/v1/users"}, {"POST", "/api/v1/users"}, {"PUT", "/api/v1/users/user-id/role"},
		{"GET", "/api/v1/servers"}, {"POST", "/api/v1/servers"}, {"GET", "/api/v1/servers/server-id"}, {"PUT", "/api/v1/servers/server-id"}, {"DELETE", "/api/v1/servers/server-id"}, {"POST", "/api/v1/servers/server-id/test"},
		{"GET", "/api/v1/servers/server-id/metric-history"}, {"GET", "/api/v1/servers/server-id/freshness"}, {"GET", "/api/v1/servers/server-id/queries"},
		{"GET", "/api/v1/problems"}, {"PUT", "/api/v1/problems/finding-id/status"}, {"GET", "/api/v1/incidents"}, {"GET", "/api/v1/incidents/incident-id"}, {"GET", "/api/v1/overview"},
		{"POST", "/api/v1/notifications/test"}, {"GET", "/api/v1/notifications"}, {"POST", "/api/v1/notifications"}, {"PUT", "/api/v1/notifications/destination-id"}, {"DELETE", "/api/v1/notifications/destination-id"},
		{"GET", "/api/v1/notification-routes"}, {"POST", "/api/v1/notification-routes"}, {"PUT", "/api/v1/notification-routes/route-id"}, {"DELETE", "/api/v1/notification-routes/route-id"}, {"GET", "/api/v1/notification-deliveries"},
		{"GET", "/api/v1/maintenance-windows"}, {"POST", "/api/v1/maintenance-windows"}, {"DELETE", "/api/v1/maintenance-windows/window-id"},
		{"GET", "/api/v1/suppressions"}, {"POST", "/api/v1/suppressions"}, {"DELETE", "/api/v1/suppressions/suppression-id"},
		{"GET", "/api/v1/threshold-overrides"}, {"POST", "/api/v1/threshold-overrides"}, {"DELETE", "/api/v1/threshold-overrides/override-id"},
		{"GET", "/api/v1/rule-profiles"}, {"POST", "/api/v1/rule-profiles"}, {"DELETE", "/api/v1/rule-profiles/profile-id"}, {"POST", "/api/v1/rule-profiles/profile-id/apply"},
		{"GET", "/api/v1/audit-events"}, {"GET", "/api/v1/change-events"}, {"POST", "/api/v1/deployments"}, {"DELETE", "/api/v1/deployments/deployment-id"}, {"GET", "/api/v1/diagnostic-bundle"},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			request := &http.Request{Method: route.method, URL: mustURL(t, route.path)}
			_, pattern := a.mux.Handler(request)
			if pattern == "" {
				t.Fatalf("route is not registered")
			}
		})
	}
}

func mustURL(t *testing.T, path string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
