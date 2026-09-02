package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
