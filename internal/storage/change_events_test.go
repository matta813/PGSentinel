package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
)

func TestChangeEventLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "changes.db"), "a sufficiently long test encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	server := models.Server{ID: uuid.NewString(), Name: "prod", Host: "db", User: "monitor", Password: "secret", SSLMode: "require"}
	if err := store.CreateServer(context.Background(), &server); err != nil {
		t.Fatal(err)
	}
	event := models.ChangeEvent{ServerID: server.ID, Kind: "deployment", Summary: "release 2.4", OccurredAt: time.Now().UTC(), Details: []string{"commit abc"}}
	if err := store.RecordChangeEvent(context.Background(), &event); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListChangeEvents(context.Background(), server.ID, event.OccurredAt.Add(-time.Minute), event.OccurredAt.Add(time.Minute), 10)
	if err != nil || len(items) != 1 || items[0].Summary != event.Summary || len(items[0].Details) != 1 {
		t.Fatalf("items=%#v err=%v", items, err)
	}
	if err := store.DeleteChangeEvent(context.Background(), event.ID); err != nil {
		t.Fatal(err)
	}
	items, err = store.ListChangeEvents(context.Background(), server.ID, time.Time{}, time.Time{}, 10)
	if err != nil || len(items) != 0 {
		t.Fatalf("items=%#v err=%v", items, err)
	}
}

func TestChangeEventDetailsAreAlwaysAnArray(t *testing.T) {
	store, ctx := testMonitoringStore(t, "change-event-details")
	server := models.Server{ID: "server", Name: "db", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	event := models.ChangeEvent{ServerID: server.ID, Kind: "deployment", Summary: "Release", OccurredAt: time.Now().UTC()}
	if err := store.RecordChangeEvent(ctx, &event); err != nil {
		t.Fatal(err)
	}
	if event.Details == nil {
		t.Fatal("created event details must be an empty array, not nil")
	}
	items, err := store.ListChangeEvents(ctx, server.ID, time.Time{}, time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Details == nil {
		t.Fatalf("stored event details = %#v, want an empty array", items)
	}
}
