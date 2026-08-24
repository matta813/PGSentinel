package collector

import (
	"strings"
	"testing"
)

func TestCheckpointStatsSQLSupportsPostgreSQL15And17(t *testing.T) {
	if query := checkpointStatsSQL(150000); !strings.Contains(query, "pg_stat_bgwriter") || !strings.Contains(query, "checkpoints_req") {
		t.Fatalf("unexpected PostgreSQL 15 query: %s", query)
	}
	if query := checkpointStatsSQL(170000); !strings.Contains(query, "pg_stat_checkpointer") || !strings.Contains(query, "num_requested") {
		t.Fatalf("unexpected PostgreSQL 17 query: %s", query)
	}
}
