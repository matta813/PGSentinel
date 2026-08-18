package collector

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

func TestScheduleNormalizesEachCollectorIndependently(t *testing.T) {
	schedule := (Schedule{Fast: 2 * time.Second, Standard: 15 * time.Second, Slow: 2 * time.Minute, Metadata: time.Hour, Retention: 48 * time.Hour}).normalized()
	if schedule.Fast != 2*time.Second || schedule.Standard != 15*time.Second || schedule.Slow != 2*time.Minute || schedule.Metadata != time.Hour || schedule.Retention != 48*time.Hour {
		t.Fatalf("custom schedule changed: %#v", schedule)
	}
	fallback := (Schedule{}).normalized()
	if fallback.Fast != 30*time.Second || fallback.Standard != 30*time.Second || fallback.Slow != 5*time.Minute || fallback.Metadata != 30*time.Minute || fallback.Retention != 30*24*time.Hour {
		t.Fatalf("unexpected fallback schedule: %#v", fallback)
	}
}

func TestCollectionCyclesSelectExpectedWork(t *testing.T) {
	tests := []struct {
		cycle                          collectionCycle
		fast, standard, slow, metadata bool
	}{{cycleFast, true, false, false, false}, {cycleStandard, false, true, false, false}, {cycleSlow, false, false, true, false}, {cycleMetadata, false, false, false, true}, {cycleAll, true, true, true, true}}
	for _, test := range tests {
		if (test.cycle&cycleFast != 0) != test.fast || (test.cycle&cycleStandard != 0) != test.standard || (test.cycle&cycleSlow != 0) != test.slow || (test.cycle&cycleMetadata != 0) != test.metadata {
			t.Fatalf("cycle %d selected unexpected work", test.cycle)
		}
	}
}

func TestManagerUsesConfiguredRetention(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "manager.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err = store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, at := range []time.Time{now.Add(-3 * time.Hour), now.Add(-30 * time.Minute)} {
		if err = store.SaveSnapshot(ctx, server.ID, "core", map[string]bool{"ok": true}, at); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{Retention: time.Hour})
	if err = manager.prune(ctx, now); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM snapshots`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("configured retention kept %d snapshots", count)
	}
}

func TestRestoreCapabilitiesPreservesKnownStateOnFastCycles(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "manager.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err = store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	serverID := "server"
	at := time.Now().UTC()
	if err = store.SaveSnapshot(ctx, serverID, "core", models.Snapshot{ServerID: serverID, CollectedAt: at, Capabilities: map[string]bool{"pg_stat_statements": true}}, at); err != nil {
		t.Fatal(err)
	}
	snapshot := models.Snapshot{ServerID: serverID, CollectedAt: at, Capabilities: map[string]bool{}}
	restoreCapabilities(ctx, store, serverID, &snapshot)
	if !snapshot.Capabilities["pg_stat_statements"] {
		t.Fatal("expected pg_stat_statements capability to be preserved on a fast cycle")
	}
}
