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

func TestDifferentIndexSemanticsAreNotDuplicates(t *testing.T) {
	tests := []struct {
		name    string
		indexes []models.IndexStat
	}{
		{
			name: "unique and non-unique",
			indexes: []models.IndexStat{
				{Database: "db", Schema: "public", Table: "events", Index: "unique_tenant", Definition: "CREATE UNIQUE INDEX unique_tenant ON public.events USING btree (tenant_id)", Unique: true},
				{Database: "db", Schema: "public", Table: "events", Index: "tenant", Definition: "CREATE INDEX tenant ON public.events USING btree (tenant_id)"},
			},
		},
		{
			name: "different access methods",
			indexes: []models.IndexStat{
				{Database: "db", Schema: "public", Table: "events", Index: "tenant_btree", Definition: "CREATE INDEX tenant_btree ON public.events USING btree (tenant_id)"},
				{Database: "db", Schema: "public", Table: "events", Index: "tenant_hash", Definition: "CREATE INDEX tenant_hash ON public.events USING hash (tenant_id)"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, finding := range IndexFindings("server", test.indexes) {
				if finding.RuleID == "duplicate-index" {
					t.Fatalf("semantically different indexes reported as duplicates: %#v", finding)
				}
			}
		})
	}
}
