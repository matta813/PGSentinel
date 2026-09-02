package storage

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

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
