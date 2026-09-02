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

func TestListChangeEventsRejectsMissingServer(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/change-events?serverId=11111111-1111-4111-8111-111111111111", nil))
	if r.Code != http.StatusNotFound {
		t.Fatalf("missing server change events=%d %s", r.Code, r.Body.String())
	}
}

func TestDeploymentChangeEventLifecycle(t *testing.T) {
	h := testAPI(t)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(`{"name":"prod","host":"db","user":"monitor","password":"secret"}`)))
	if response.Code != http.StatusCreated {
		t.Fatalf("server=%d %s", response.Code, response.Body.String())
	}
	var server struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &server)
	payload := fmt.Sprintf(`{"serverId":%q,"summary":"release checkout 2.4","occurredAt":%q}`, server.ID, time.Now().UTC().Format(time.RFC3339))
	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/deployments", strings.NewReader(payload)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", response.Code, response.Body.String())
	}
	var event struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &event)
	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/change-events?serverId="+server.ID, nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "release checkout 2.4") {
		t.Fatalf("list=%d %s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/v1/deployments/"+event.ID, nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("delete=%d %s", response.Code, response.Body.String())
	}
}
