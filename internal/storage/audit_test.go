package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestAuditEventsFilterAndExpire(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "audit.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	old := models.AuditEvent{Actor: "admin", Action: "server.created", ResourceType: "server", ResourceID: "old", Summary: "A PostgreSQL target was added.", OccurredAt: now.Add(-366 * 24 * time.Hour)}
	if err := store.RecordAuditEvent(ctx, &old); err != nil {
		t.Fatal(err)
	}
	current := models.AuditEvent{Actor: "admin", Action: "finding.acknowledged", ResourceType: "finding", ResourceID: "finding", Summary: "A finding was acknowledged.", OccurredAt: now}
	if err := store.RecordAuditEvent(ctx, &current); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListAuditEvents(ctx, AuditFilter{Actor: "admin", Search: "acknowledged", Limit: 20})
	if err != nil || len(items) != 1 || items[0].ID != current.ID {
		t.Fatalf("events=%#v err=%v", items, err)
	}
	var oldCount int
	if err := store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_events WHERE id=?`, old.ID).Scan(&oldCount); err != nil || oldCount != 0 {
		t.Fatalf("expired event retained count=%d err=%v", oldCount, err)
	}
}

func TestAuditSearchTreatsWildcardsLiterally(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "audit-search.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.RecordAuditEvent(context.Background(), &models.AuditEvent{Actor: "admin", Action: "test.action", ResourceType: "test", Summary: "100 percent"}); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListAuditEvents(context.Background(), AuditFilter{Search: "%", Limit: 20})
	if err != nil || len(items) != 0 {
		t.Fatalf("wildcard search escaped bounds: %#v err=%v", items, err)
	}
}

func TestAuditPaginationIsBounded(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "audit-pagination.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.ListAuditEvents(context.Background(), AuditFilter{Limit: 101}); err == nil {
		t.Fatal("unbounded audit page was accepted")
	}
}
