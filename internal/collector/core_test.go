package collector

import (
	"strings"
	"testing"
)

func TestConnectionsSQLAppliesFilterToAggregate(t *testing.T) {
	for _, aggregate := range []string{"min(xact_start) FILTER", "min(state_change) FILTER"} {
		if !strings.Contains(connectionsSQL, aggregate) {
			t.Fatalf("connection age filter must be attached to %s", aggregate)
		}
	}
	if strings.Contains(connectionsSQL, ")) FILTER") {
		t.Fatal("FILTER cannot be attached to the surrounding EXTRACT expression")
	}
}
