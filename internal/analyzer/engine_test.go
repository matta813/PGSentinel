package analyzer

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestConnectionAndVacuumRules(t *testing.T) {
	s := models.Snapshot{ServerID: "s", Connections: models.ConnectionStats{Total: 96, Max: 100, Utilization: 96}, Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, Tables: []models.TableStat{{Database: "db", Schema: "public", Table: "events", LiveTuples: 70000, DeadTuples: 30000, VacuumThreshold: 20000, VacuumProgress: 150}}}
	got := New(DefaultThresholds()).Analyze(s)
	if len(got) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(got))
	}
	if got[0].Severity != models.SeverityCritical {
		t.Fatalf("expected critical, got %s", got[0].Severity)
	}
}

func TestReplicationAndCheckpointRulesAreRoleAware(t *testing.T) {
	base := models.Snapshot{ServerID: "s", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}}
	replica := base
	replica.Replication.InRecovery = true
	if got := New(DefaultThresholds()).Analyze(replica); !rulePresent(got, "wal-receiver-disconnected") {
		t.Fatal("recovery server without receiver must be visible")
	}
	primary := base
	primary.Replication.Standbys = []models.ReplicationStandby{{Application: "replica-1", State: "streaming", ReplayLagSeconds: 90}}
	primary.Replication.Slots = []models.ReplicationSlot{{Name: "abandoned", Active: false, RetainedBytes: 2 * 1024 * 1024 * 1024}}
	reset := time.Now().Add(-20 * time.Minute)
	primary.WAL = models.WALStats{TimedCheckpoints: 8, RequestedCheckpoints: 12, StatsReset: &reset}
	got := New(DefaultThresholds()).Analyze(primary)
	for _, rule := range []string{"standby-replay-lag", "inactive-slot-wal", "requested-checkpoints", "checkpoint-frequency"} {
		if !rulePresent(got, rule) {
			t.Fatalf("expected %s finding", rule)
		}
	}
}

func TestArchiveFailuresAndDelayedReplicaIntent(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-time.Hour)
	s := models.Snapshot{ServerID: "s", ServerTags: []string{"allow-delayed-replicas"}, Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, Replication: models.ReplicationStats{Standbys: []models.ReplicationStandby{{Application: "delayed", State: "streaming", ReplayLagSeconds: 600, PendingReplayBytes: 1e9}}}, WAL: models.WALStats{ArchiveMode: "on", ArchiveConfigured: true, LastArchivedAt: &old, LastFailedAt: &now, LastFailedWAL: "000000010000000000000001", FailedArchiveCount: 4}}
	got := New(DefaultThresholds()).Analyze(s)
	if rulePresent(got, "standby-replay-lag") {
		t.Fatal("intentional delayed replica raised replay lag finding")
	}
	if !rulePresent(got, "archive-failure-current") {
		t.Fatal("current archive failure not reported")
	}
}

func TestPausedRecoveryIsExplicit(t *testing.T) {
	s := models.Snapshot{ServerID: "s", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, Replication: models.ReplicationStats{InRecovery: true, RecoveryPaused: true, Receiver: &models.WALReceiver{Status: "streaming"}, ReplayLSN: "1/2", TimelineID: 3}}
	if got := New(DefaultThresholds()).Analyze(s); !rulePresent(got, "recovery-paused") {
		t.Fatal("paused recovery not reported")
	}
}

func TestRequestedRestartpointsAreStandbyOnly(t *testing.T) {
	base := models.Snapshot{ServerID: "s", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, WAL: models.WALStats{RestartpointsTimed: 8, RestartpointsRequested: 12, RestartpointsDone: 15}}
	primary := New(DefaultThresholds()).Analyze(base)
	if rulePresent(primary, "requested-restartpoints") {
		t.Fatal("primary raised restartpoint finding")
	}
	base.Replication.InRecovery = true
	base.Replication.Receiver = &models.WALReceiver{Status: "streaming"}
	if got := New(DefaultThresholds()).Analyze(base); !rulePresent(got, "requested-restartpoints") {
		t.Fatal("standby restartpoint pressure not reported")
	}
}
func TestHealthScoreWeightsSeverity(t *testing.T) {
	low := HealthScore([]models.Finding{{Status: "active", Severity: models.SeverityLow, Category: "Queries"}})
	critical := HealthScore([]models.Finding{{Status: "active", Severity: models.SeverityCritical, Category: "Queries"}})
	if critical.Overall >= low.Overall {
		t.Fatal("critical must reduce score more")
	}
}

func TestBlockingQueriesRuleRequiresLockDurationAboveThreshold(t *testing.T) {
	base := models.Snapshot{ServerID: "s", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}}
	engine := New(DefaultThresholds())

	shortLock := base
	shortLock.Locks = []models.LockInfo{{BlockedPID: 1, BlockingPID: 2, DurationSeconds: 5}}
	if got := engine.Analyze(shortLock); rulePresent(got, "blocking-queries") {
		t.Fatal("transient lock below the threshold must not raise a finding")
	}

	longLock := base
	longLock.Locks = []models.LockInfo{{BlockedPID: 1, BlockingPID: 2, DurationSeconds: 90}}
	found := false
	for _, f := range engine.Analyze(longLock) {
		if f.RuleID == "blocking-queries" {
			found = true
			if len(f.Evidence) > 0 && f.Evidence[0].Value != "1" {
				t.Fatalf("finding must count only long locks, evidence %#v", f.Evidence)
			}
		}
	}
	if !found {
		t.Fatal("long blocking lock must raise a finding")
	}
}

func rulePresent(findings []models.Finding, rule string) bool {
	for _, f := range findings {
		if f.RuleID == rule {
			return true
		}
	}
	return false
}
