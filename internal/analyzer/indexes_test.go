package analyzer

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
)

func TestDuplicateAndUnusedIndexes(t *testing.T) {
	items := []models.IndexStat{{Database: "db", Schema: "public", Table: "events", Index: "a", Definition: "CREATE INDEX a ON public.events USING btree (tenant_id)", SizeBytes: 2e8}, {Database: "db", Schema: "public", Table: "events", Index: "b", Definition: "CREATE INDEX b ON public.events USING btree (tenant_id)", SizeBytes: 2e8}}
	got := IndexFindings("s", items)
	if len(got) != 3 {
		t.Fatalf("expected two unused and duplicate finding, got %d", len(got))
	}
}
