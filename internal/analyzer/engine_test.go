package analyzer

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
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
