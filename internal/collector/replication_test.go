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

func TestReplicationQueriesExposeLSNGapsAndVersionAwareSlotAge(t *testing.T) {
	query := standbyStatsSQL()
	for _, expected := range []string{"sent_lsn", "write_lsn", "flush_lsn", "replay_lsn", "reply_time", "pg_wal_lsn_diff"} {
		if !strings.Contains(query, expected) {
			t.Fatalf("standby query missing %s", expected)
		}
	}
	if query := replicationSlotsSQL(150000); strings.Contains(query, "inactive_since") {
		t.Fatal("PostgreSQL 15 query uses unavailable inactive_since")
	}
	if query := replicationSlotsSQL(160000); strings.Contains(query, "inactive_since") {
		t.Fatal("PostgreSQL 16 query uses unavailable inactive_since")
	}
	if query := replicationSlotsSQL(170000); !strings.Contains(query, "inactive_since") {
		t.Fatal("PostgreSQL 17 query does not collect inactive_since")
	}
}
