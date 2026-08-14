package analyzer

import (
	"gitlab.scruzzi.com/root/postgresqlui/internal/models"
	"testing"
)

func TestConnectionAndVacuumRules(t *testing.T) {
	s := models.Snapshot{ServerID: "s", Connections: models.ConnectionStats{Total: 96, Max: 100, Utilization: 96}, Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, Tables: []models.TableStat{{Database: "db", Schema: "public", Table: "events", LiveTuples: 700, DeadTuples: 300, VacuumThreshold: 200, VacuumProgress: 150}}}
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
