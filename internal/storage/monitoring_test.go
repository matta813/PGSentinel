package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestFindingLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err = s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	f := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityHigh, Category: "Vacuum", Title: "T", Status: "active", StartedAt: time.Now(), UpdatedAt: time.Now()}
	if err = s.UpsertFindings(ctx, "s", []models.Finding{f}); err != nil {
		t.Fatal(err)
	}
	if err = s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListFindings(ctx, "resolved", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d: %v", len(got), err)
	}
}

func TestFindingFiltersCanBeCombined(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "filters.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
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
	s, err := Open(filepath.Join(t.TempDir(), "servers.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "old", Host: "old.example", Port: 5432, User: "old-user", Password: "old-password", SSLMode: "prefer"}
	if err = s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	update := models.Server{ID: server.ID, Name: "new", Host: "new.example", Port: 6432, User: "new-user", SSLMode: "require", Tags: []string{"production"}}
	if err = s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetServer(ctx, server.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "new" || got.Host != "new.example" || got.Port != 6432 || got.Password != "old-password" {
		t.Fatalf("unexpected preserved update: %#v", got)
	}
	update.Password = "rotated-password"
	if err = s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetServer(ctx, server.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "rotated-password" {
		t.Fatalf("password was not rotated: %q", got.Password)
	}
}

func TestPruneRemovesOnlyExpiredSnapshots(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "retention.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "retention-server", Name: "retention", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err = s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err = s.SaveSnapshot(ctx, server.ID, "core", map[string]int{"age": 2}, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveSnapshot(ctx, server.ID, "core", map[string]int{"age": 0}, now); err != nil {
		t.Fatal(err)
	}
	if err = s.Prune(ctx, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM snapshots WHERE server_id=?`, server.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one retained snapshot, got %d", count)
	}
}
