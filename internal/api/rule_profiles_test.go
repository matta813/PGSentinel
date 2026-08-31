package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRuleProfileImportPreviewAndApply(t *testing.T) {
	h := testAPI(t)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/servers", strings.NewReader(`{"name":"prod","host":"db","user":"monitor","password":"secret"}`)))
	var s struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(r.Body.Bytes(), &s)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/rule-profiles", strings.NewReader(`{"name":"OLTP","description":"reviewed","entries":[{"ruleId":"blocking-queries","value":30}]}`)))
	if r.Code != 201 {
		t.Fatalf("create=%d %s", r.Code, r.Body.String())
	}
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(r.Body.Bytes(), &p)
	body := fmt.Sprintf(`{"scopeType":"server","scopeValue":%q,"reason":"reviewed profile","preview":true}`, s.ID)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/rule-profiles/"+p.ID+"/apply", strings.NewReader(body)))
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"willApply":1`) {
		t.Fatalf("preview=%d %s", r.Code, r.Body.String())
	}
	body = strings.Replace(body, `,"preview":true`, "", 1)
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/rule-profiles/"+p.ID+"/apply", strings.NewReader(body)))
	if r.Code != 200 {
		t.Fatalf("apply=%d %s", r.Code, r.Body.String())
	}
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/threshold-overrides", nil))
	if !strings.Contains(r.Body.String(), `"value":30`) {
		t.Fatalf("overrides=%s", r.Body.String())
	}
}
func TestRuleProfileRejectsUnsafeValues(t *testing.T) {
	h := testAPI(t)
	for _, b := range []string{`{"name":"bad","entries":[{"ruleId":"unknown","value":1}]}`, `{"name":"bad","entries":[{"ruleId":"blocking-queries","value":99999}]}`, `{"name":"bad","entries":[{"ruleId":"blocking-queries","value":30},{"ruleId":"blocking-queries","value":40}]}`} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodPost, "/api/v1/rule-profiles", strings.NewReader(b)))
		if r.Code != 422 {
			t.Fatalf("status=%d %s", r.Code, r.Body.String())
		}
	}
}
