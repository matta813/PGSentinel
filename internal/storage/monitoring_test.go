package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func testMonitoringStore(t *testing.T, name string) (*Store, context.Context) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), name+".db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, context.Background()
}

func TestFindingLifecycle(t *testing.T) {
	s, ctx := testMonitoringStore(t, "lifecycle")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	f := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityHigh, Category: "Vacuum", Title: "T", Status: "active", StartedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{f}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListFindings(ctx, "resolved", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d: %v", len(got), err)
	}
}

func TestAcknowledgedFindingPersistsUntilConditionResolves(t *testing.T) {
	s, ctx := testMonitoringStore(t, "acknowledged")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "rule", Fingerprint: "fingerprint", ServerID: "s", Severity: models.SeverityHigh, Category: "Connections", Title: "Saturation", Status: "active", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFindingStatus(ctx, finding.ID, "acknowledged"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	acknowledged, err := s.ListFindings(ctx, "acknowledged", "s")
	if err != nil || len(acknowledged) != 1 {
		t.Fatalf("acknowledged=%#v err=%v", acknowledged, err)
	}
	open, err := s.ListFindings(ctx, "open", "s")
	if err != nil || len(open) != 1 {
		t.Fatalf("open=%#v err=%v", open, err)
	}
	if err := s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.ListFindings(ctx, "resolved", "s")
	if err != nil || len(resolved) != 1 {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
}

func TestFindingFiltersCanBeCombined(t *testing.T) {
	s, ctx := testMonitoringStore(t, "filters")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	findings := []models.Finding{
		{ID: "connection", RuleID: "connection", Fingerprint: "connection", ServerID: "s", Severity: models.SeverityCritical, Category: "Connections", Title: "Connection saturation", Summary: "Pool is exhausted", Status: "active", StartedAt: now, UpdatedAt: now},
		{ID: "vacuum", RuleID: "vacuum", Fingerprint: "vacuum", ServerID: "s", Database: "warehouse", Severity: models.SeverityHigh, Category: "Vacuum", Title: "Dead tuples", Summary: "Vacuum is behind", Status: "active", StartedAt: now, UpdatedAt: now},
	}
	if err := s.UpsertFindings(ctx, "s", findings); err != nil {
		t.Fatal(err)
	}
	got, err := s.FilterFindings(ctx, FindingFilter{Status: "active", Severity: "HIGH", Category: "vacuum", Search: "warehouse"})
	if err != nil || len(got) != 1 || got[0].ID != "vacuum" {
		t.Fatalf("combined filter=%#v err=%v", got, err)
	}
	got, err = s.FilterFindings(ctx, FindingFilter{Search: "POOL IS EXHAUSTED"})
	if err != nil || len(got) != 1 || got[0].ID != "connection" {
		t.Fatalf("case-insensitive search=%#v err=%v", got, err)
	}
}

func TestUpdateServerPreservesAndRotatesPassword(t *testing.T) {
	s, ctx := testMonitoringStore(t, "servers")
	server := models.Server{ID: "server", Name: "old", Host: "old.example", Port: 5432, User: "old-user", Password: "old-password", SSLMode: "prefer"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	update := models.Server{ID: server.ID, Name: "new", Host: "new.example", Port: 6432, User: "new-user", SSLMode: "require", Tags: []string{"production"}}
	if err := s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetServer(ctx, server.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "new" || got.Host != "new.example" || got.Port != 6432 || got.Password != "old-password" {
		t.Fatalf("unexpected update: %#v", got)
	}
	update.Password = "rotated-password"
	if err := s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetServer(ctx, server.ID, true)
	if err != nil || got.Password != "rotated-password" {
		t.Fatalf("password=%q err=%v", got.Password, err)
	}
}

func TestPruneRemovesOnlyExpiredSnapshots(t *testing.T) {
	s, ctx := testMonitoringStore(t, "retention")
	server := models.Server{ID: "retention-server", Name: "retention", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := s.SaveSnapshot(ctx, server.ID, "core", map[string]int{"age": 2}, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSnapshot(ctx, server.ID, "core", map[string]int{"age": 0}, now); err != nil {
		t.Fatal(err)
	}
	if err := s.Prune(ctx, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM snapshots WHERE server_id=?`, server.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one retained snapshot, got %d", count)
	}
}
