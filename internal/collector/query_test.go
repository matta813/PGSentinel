package collector

import (
	"github.com/matta813/pgsentinel/internal/models"
	"strings"
	"testing"
)

func TestImpactScoreRewardsTotalWork(t *testing.T) {
	frequent := ImpactScore(models.QueryStat{Calls: 2e7, MeanExecMS: 1, TotalExecMS: 2e7})
	rare := ImpactScore(models.QueryStat{Calls: 3, MeanExecMS: 4000, TotalExecMS: 12000})
	if frequent <= rare {
		t.Fatalf("frequent=%f rare=%f", frequent, rare)
	}
}

func TestQueryRegressionCollectionIsBoundedReadOnlyAndResetAware(t *testing.T) {
	combined := strings.ToLower(queryStatsMetadataSQL + " " + queryStatsSQL)
	for _, wanted := range []string{"pg_stat_statements_info", "stats_reset", "pg_postmaster_start_time()", "shared_blks_read", "rows", "limit 200"} {
		if !strings.Contains(combined, wanted) {
			t.Fatalf("missing %q", wanted)
		}
	}
	for _, forbidden := range []string{"explain", "analyze ", "update ", "delete ", "insert ", "alter ", "create ", "drop "} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("query collection contains %q", forbidden)
		}
	}
}
