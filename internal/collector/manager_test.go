package collector

import (
	"context"
	"errors"
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

func TestCollectibleDatabasesSkipsTemplatesSortsAndCaps(t *testing.T) {
	stats := []models.DatabaseStat{
		{Name: "template0", SizeBytes: 10000000},
		{Name: "template1", SizeBytes: 9000000},
		{Name: "analytics", SizeBytes: 50000000},
		{Name: "postgres", SizeBytes: 8000000},
		{Name: "app", SizeBytes: 20000000},
		{Name: "reports", SizeBytes: 1000000},
	}
	got := collectibleDatabases(stats, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 databases, got %d: %#v", len(got), got)
	}
	wantOrder := []string{"analytics", "app", "postgres"}
	for i, want := range wantOrder {
		if got[i].Name != want {
			t.Fatalf("position %d: got %s, want %s", i, got[i].Name, want)
		}
	}
	if names := collectibleDatabases(nil, 5); len(names) != 0 {
		t.Fatalf("empty input must yield no targets, got %#v", names)
	}
}

func TestScheduleNormalizesFanoutLimit(t *testing.T) {
	if got := (Schedule{}).normalized().FanoutLimit; got != 32 {
		t.Fatalf("default fanout limit = %d, want 32", got)
	}
	if got := (Schedule{FanoutLimit: 8}).normalized().FanoutLimit; got != 8 {
		t.Fatalf("custom fanout limit = %d, want 8", got)
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

func TestIncompleteCollectionDoesNotResolveExistingFindings(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "manager.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "query-impact", Fingerprint: "fingerprint", ServerID: server.ID, Severity: models.SeverityMedium, Category: "Queries", Title: "High impact", Status: "active", StartedAt: now, UpdatedAt: now}
	if err := store.UpsertFindings(ctx, server.ID, []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	if err := manager.reconcileFindings(ctx, server.ID, nil, false); err != nil {
		t.Fatal(err)
	}
	active, err := store.ListFindings(ctx, "active", server.ID)
	if err != nil || len(active) != 1 {
		t.Fatalf("incomplete collection changed active findings: %#v, err=%v", active, err)
	}

	if err := manager.reconcileFindings(ctx, server.ID, nil, true); err != nil {
		t.Fatal(err)
	}
	resolved, err := store.ListFindings(ctx, "resolved", server.ID)
	if err != nil || len(resolved) != 1 {
		t.Fatalf("complete collection did not reconcile findings: %#v, err=%v", resolved, err)
	}
}

func TestIncompleteCollectionIsDegradedRatherThanHealthy(t *testing.T) {
	status, detail := collectionOutcome(false)
	if status != "degraded" || detail == "" {
		t.Fatalf("status=%q detail=%q", status, detail)
	}
	status, detail = collectionOutcome(true)
	if status != "healthy" || detail != "" {
		t.Fatalf("status=%q detail=%q", status, detail)
	}
}

func TestMissingFallbackSnapshotKeepsCollectionIncomplete(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "missing-snapshot.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	var value []models.QueryStat
	if manager.restoreSnapshot(context.Background(), "missing", "queries", &value) {
		t.Fatal("missing cached data was treated as complete")
	}
}

func TestCoreCollectionFailureReplacesStaleHealthyStatus(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "manager.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable", Version: "16.4", Status: "healthy"}
	if err := store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	if err := store.UpdateServerStatus(ctx, server.ID, "healthy", server.Version, "", true); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	manager.recordCollectionFailure(ctx, server, errors.New("permission denied for pg_stat_activity"))

	got, err := store.GetServer(ctx, server.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.LastError != "permission denied for pg_stat_activity" {
		t.Fatalf("server status=%q error=%q, want collection error", got.Status, got.LastError)
	}
}
