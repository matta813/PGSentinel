package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
)

func TestCollectionResourceFreshnessLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "freshness.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := models.Server{ID: uuid.NewString(), Name: "primary", Host: "db", Port: 5432, User: "monitor", Password: "secret", SSLMode: "require"}
	if err := store.CreateServer(context.Background(), &server); err != nil {
		t.Fatal(err)
	}
	collected := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	if err := store.RecordCollectionResource(context.Background(), server.ID, "queries", "fresh", 30*time.Second, collected, ""); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListCollectionResources(context.Background(), server.ID, collected.Add(20*time.Second))
	if err != nil || len(items) != 1 || items[0].State != "fresh" || items[0].ConsecutiveFailures != 0 {
		t.Fatalf("items=%#v err=%v", items, err)
	}
	if err := store.RecordCollectionResource(context.Background(), server.ID, "queries", "unavailable", 30*time.Second, collected.Add(30*time.Second), "Collection failed."); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListCollectionResources(context.Background(), server.ID, collected.Add(40*time.Second))
	if err != nil || items[0].State != "unavailable" || items[0].ConsecutiveFailures != 1 || !items[0].LastSuccessfulAt.Equal(collected) {
		t.Fatalf("items=%#v err=%v", items, err)
	}
	if err := store.RecordCollectionResource(context.Background(), server.ID, "queries", "fresh", 30*time.Second, collected.Add(time.Minute), ""); err != nil {
		t.Fatal(err)
	}
	items, _ = store.ListCollectionResources(context.Background(), server.ID, collected.Add(2*time.Minute+time.Second))
	if items[0].State != "stale" || items[0].ConsecutiveFailures != 0 {
		t.Fatalf("items=%#v", items)
	}
}
