package incidents

import (
	"strings"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestCorrelateRequiresTimeServerAndExplicitRelationship(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	base := models.Finding{ID: "lock", ServerID: "primary", Category: "Locks", Severity: models.SeverityHigh, Status: "active", StartedAt: now, UpdatedAt: now}
	connection := models.Finding{ID: "connections", ServerID: "primary", Category: "Connections", Severity: models.SeverityCritical, Status: "active", StartedAt: now.Add(5 * time.Minute), UpdatedAt: now.Add(5 * time.Minute)}
	query := models.Finding{ID: "query", ServerID: "primary", Category: "Queries", Severity: models.SeverityMedium, Status: "active", StartedAt: now.Add(7 * time.Minute), UpdatedAt: now.Add(7 * time.Minute)}
	got := Correlate([]models.Finding{base, connection, query})
	if len(got) != 1 || len(got[0].Findings) != 3 || got[0].Severity != models.SeverityCritical || got[0].Status != "active" {
		t.Fatalf("incident=%#v", got)
	}
	if !strings.Contains(got[0].Summary, "does not establish causation") || len(got[0].Rationale) < 2 {
		t.Fatalf("unsafe or unexplained summary: %#v", got[0])
	}
	connection.ServerID = "other"
	if got := Correlate([]models.Finding{base, connection}); len(got) != 0 {
		t.Fatalf("cross-server findings correlated: %#v", got)
	}
	connection.ServerID = "primary"
	connection.StartedAt = now.Add(16 * time.Minute)
	if got := Correlate([]models.Finding{base, connection}); len(got) != 0 {
		t.Fatalf("distant findings correlated: %#v", got)
	}
}

func TestCorrelateDoesNotGroupUnrelatedTemporalNeighbors(t *testing.T) {
	now := time.Now().UTC()
	findings := []models.Finding{
		{ID: "index", ServerID: "server", Category: "Indexes", StartedAt: now, UpdatedAt: now},
		{ID: "config", ServerID: "server", Category: "Configuration", StartedAt: now, UpdatedAt: now},
	}
	if got := Correlate(findings); len(got) != 0 {
		t.Fatalf("temporal proximity alone created an incident: %#v", got)
	}
}

func TestCorrelateDoesNotBridgeBeyondCorrelationWindow(t *testing.T) {
	now := time.Now().UTC()
	findings := []models.Finding{
		{ID: "a", ServerID: "server", Category: "Locks", StartedAt: now, UpdatedAt: now},
		{ID: "b", ServerID: "server", Category: "Locks", StartedAt: now.Add(14 * time.Minute), UpdatedAt: now},
		{ID: "c", ServerID: "server", Category: "Locks", StartedAt: now.Add(28 * time.Minute), UpdatedAt: now},
	}
	got := Correlate(findings)
	if len(got) != 1 || len(got[0].Findings) != 2 {
		t.Fatalf("transitive bridge escaped 15-minute window: %#v", got)
	}
}

func TestCorrelateBuildsResolvedIncident(t *testing.T) {
	now := time.Now().UTC()
	resolved := now.Add(20 * time.Minute)
	findings := []models.Finding{
		{ID: "replication", ServerID: "server", Category: "Replication", Severity: models.SeverityHigh, Status: "resolved", StartedAt: now, UpdatedAt: resolved, ResolvedAt: &resolved},
		{ID: "wal", ServerID: "server", Category: "WAL", Severity: models.SeverityMedium, Status: "resolved", StartedAt: now.Add(time.Minute), UpdatedAt: resolved, ResolvedAt: &resolved},
	}
	got := Correlate(findings)
	if len(got) != 1 || got[0].Status != "resolved" || got[0].ResolvedAt == nil {
		t.Fatalf("resolved incident=%#v", got)
	}
}
