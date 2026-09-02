package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIncidentAPIValidationAndEmptyPage(t *testing.T) {
	h := testAPI(t)
	for _, path := range []string{
		"/api/v1/incidents?status=pending",
		"/api/v1/incidents?limit=101",
		"/api/v1/incidents?offset=10001",
		"/api/v1/incidents?serverId=not-a-uuid",
	} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusUnprocessableEntity && r.Code != http.StatusBadRequest {
			t.Fatalf("%s returned %d: %s", path, r.Code, r.Body.String())
		}
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/incidents?status=active&limit=20", nil))
	if r.Code != http.StatusOK || strings.TrimSpace(r.Body.String()) != "[]" {
		t.Fatalf("empty incidents=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/incidents/not-an-id", nil))
	if r.Code != http.StatusBadRequest {
		t.Fatalf("invalid incident=%d %s", r.Code, r.Body.String())
	}
}
