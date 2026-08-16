package storage

import (
	"context"
	"github.com/matta813/pgsentinel/internal/models"
	"path/filepath"
	"testing"
	"time"
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
