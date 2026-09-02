package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
	if r.Code != http.StatusNotFound {
		t.Fatalf("missing server history=%d %s", r.Code, r.Body.String())
	}
}

func TestServerFreshnessReturnsFixedSafeResourceSet(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(`{"name":"prod","host":"db","user":"monitor","password":"secret"}`)))
	if r.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", r.Code, r.Body.String())
	}
	var server map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &server); err != nil {
		t.Fatal(err)
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+server["id"].(string)+"/freshness", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("freshness=%d %s", r.Code, r.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 11 || items[0]["state"] != "unavailable" || !strings.Contains(r.Body.String(), `"resource":"wait-events"`) || strings.Contains(r.Body.String(), "secret") {
		t.Fatalf("items=%#v", items)
	}
	for _, path := range []string{"/api/v1/servers/not-a-uuid/freshness", "/api/v1/servers/1b5c3f33-bfcb-4fd4-9c36-df76e2683ee5/freshness"} {
		r = httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != http.StatusBadRequest && r.Code != http.StatusNotFound {
			t.Fatalf("%s=%d %s", path, r.Code, r.Body.String())
		}
	}
}
