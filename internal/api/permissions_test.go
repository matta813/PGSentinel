package api

import (
	"net/http"
	"testing"
)

func TestRolePermissionBoundaries(t *testing.T) {
	cases := []struct {
		role, method, path string
		want               bool
	}{
		{"administrator", http.MethodDelete, "/api/v1/servers/id", true},
		{"operator", http.MethodGet, "/api/v1/servers", true},
		{"operator", http.MethodPut, "/api/v1/problems/abc/status", true},
		{"operator", http.MethodPost, "/api/v1/servers", false},
		{"operator", http.MethodGet, "/api/v1/audit-events", false},
		{"operator", http.MethodGet, "/api/v1/diagnostic-bundle", false},
		{"viewer", http.MethodGet, "/api/v1/incidents", true},
		{"viewer", http.MethodPut, "/api/v1/problems/abc/status", false},
		{"viewer", http.MethodGet, "/api/v1/users", false},
		{"viewer", http.MethodGet, "/api/v1/diagnostic-bundle", false},
		{"viewer", http.MethodPut, "/api/v1/auth/password", true},
	}
	for _, tc := range cases {
		if got := allowed(tc.role, tc.method, tc.path); got != tc.want {
			t.Errorf("allowed(%q,%q,%q)=%v want %v", tc.role, tc.method, tc.path, got, tc.want)
		}
	}
}
