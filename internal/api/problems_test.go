package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	for _, path := range []string{"/api/v1/problems?status=pending", "/api/v1/problems?severity=urgent", "/api/v1/problems?search=" + strings.Repeat("x", 201), "/api/v1/problems?database=" + strings.Repeat("x", 101)} {
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
