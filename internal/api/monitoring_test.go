package api

import (
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestQualityForFindingUsesItsEvidenceSource(t *testing.T) {
	items := []models.CollectionResourceStatus{{Resource: "locks", State: "fresh"}, {Resource: "wait-events", State: "partial"}, {Resource: "database-statistics", State: "stale"}, {Resource: "queries", State: "unavailable"}}
	for _, test := range []struct{ rule, want string }{{"blocking-queries", "locks"}, {"wait-lock-pressure", "wait-events"}, {"wait-class-concentration", "wait-events"}, {"deadlocks", "database-statistics"}, {"query-regression", "queries"}} {
		quality := qualityForFinding(models.Finding{RuleID: test.rule}, items)
		if quality == nil || quality.Resource != test.want {
			t.Fatalf("rule %s quality=%#v", test.rule, quality)
		}
	}
}
