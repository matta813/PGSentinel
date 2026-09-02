package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func TestDeleteMissingServerReturnsNotFound(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodDelete, "/api/v1/servers/11111111-1111-4111-8111-111111111111", nil))
	if r.Code != http.StatusNotFound {
		t.Fatalf("delete missing server=%d %s", r.Code, r.Body.String())
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
