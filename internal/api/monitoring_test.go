package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestServerResourceRejectsMissingServer(t *testing.T) {
	h := testAPI(t)
	for _, test := range []struct {
		id   string
		want int
	}{{"not-a-uuid", http.StatusBadRequest}, {"11111111-1111-4111-8111-111111111111", http.StatusNotFound}} {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/v1/servers/"+test.id+"/queries", nil))
		if r.Code != test.want {
			t.Fatalf("server %q resource=%d %s, want %d", test.id, r.Code, r.Body.String(), test.want)
		}
	}
}

func TestQualityForFindingUsesItsEvidenceSource(t *testing.T) {
	items := []models.CollectionResourceStatus{{Resource: "locks", State: "fresh"}, {Resource: "wait-events", State: "partial"}, {Resource: "database-statistics", State: "stale"}, {Resource: "queries", State: "unavailable"}}
	for _, test := range []struct{ rule, want string }{{"blocking-queries", "locks"}, {"wait-lock-pressure", "wait-events"}, {"wait-class-concentration", "wait-events"}, {"deadlocks", "database-statistics"}, {"query-regression", "queries"}} {
		quality := qualityForFinding(models.Finding{RuleID: test.rule}, items)
		if quality == nil || quality.Resource != test.want {
			t.Fatalf("rule %s quality=%#v", test.rule, quality)
		}
	}
}
