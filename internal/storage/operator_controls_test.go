package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
)

func operatorControlStore(t *testing.T) (*Store, context.Context, models.Server) {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "controls.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	server := models.Server{ID: uuid.NewString(), Name: "prod", Host: "db", Port: 5432, User: "monitor", Password: "secret", SSLMode: "require", Tags: []string{"production", "payments"}}
	if err := store.CreateServer(context.Background(), &server); err != nil {
		t.Fatal(err)
	}
	return store, context.Background(), server
}

func TestOperatorControlsDecorateWithoutDeletingFinding(t *testing.T) {
	store, ctx, server := operatorControlStore(t)
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "standby-replay-lag", Fingerprint: "fingerprint", ServerID: server.ID, Severity: models.SeverityHigh, Category: "Replication", Title: "Lag", StartedAt: now, UpdatedAt: now}
	if err := store.UpsertFindings(ctx, server.ID, []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	window := models.MaintenanceWindow{ID: uuid.NewString(), Description: "Planned failover", ServerTag: "production", Category: "replication", StartsAt: now.Add(-time.Minute), EndsAt: now.Add(time.Hour)}
	if err := store.CreateMaintenanceWindow(ctx, &window); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListFindings(ctx, "open", server.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplyOperatorControls(ctx, items, now); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !items[0].Suppressed || !items[0].Maintenance {
		t.Fatalf("items=%#v", items)
	}
	var count int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM findings WHERE id='finding'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("finding was deleted count=%d err=%v", count, err)
	}
}

func TestSuppressionPreventsNewDeliveryButKeepsLifecycle(t *testing.T) {
	store, ctx, server := operatorControlStore(t)
	now := time.Now().UTC()
	suppression := models.FindingSuppression{ID: uuid.NewString(), RuleID: "blocking-queries", ServerID: server.ID, Reason: "Known load test", ExpiresAt: now.Add(time.Hour)}
	if err := store.CreateSuppression(ctx, &suppression); err != nil {
		t.Fatal(err)
	}
	finding := models.Finding{ID: "finding", RuleID: "blocking-queries", Fingerprint: "fp", ServerID: server.ID, Severity: models.SeverityCritical, Category: "Locks", Title: "Blocked", StartedAt: now, UpdatedAt: now}
	if err := store.UpsertFindings(ctx, server.ID, []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	var findings, events int
	_ = store.DB.QueryRow(`SELECT COUNT(*) FROM findings`).Scan(&findings)
	_ = store.DB.QueryRow(`SELECT COUNT(*) FROM finding_notification_events`).Scan(&events)
	if findings != 1 || events != 0 {
		t.Fatalf("findings=%d events=%d", findings, events)
	}
}

func TestThresholdResolutionUsesServerThenTagThenGlobal(t *testing.T) {
	store, ctx, server := operatorControlStore(t)
	for _, item := range []models.ThresholdOverride{{ID: uuid.NewString(), RuleID: "standby-replay-lag", ScopeType: "global", Value: 60, Reason: "default"}, {ID: uuid.NewString(), RuleID: "standby-replay-lag", ScopeType: "tag", ScopeValue: "production", Value: 120, Reason: "batch"}, {ID: uuid.NewString(), RuleID: "standby-replay-lag", ScopeType: "server", ScopeValue: server.ID, Value: 180, Reason: "delayed"}} {
		copy := item
		if err := store.CreateThresholdOverride(ctx, &copy); err != nil {
			t.Fatal(err)
		}
	}
	items, err := store.EffectiveThresholdOverrides(ctx, server)
	if err != nil || len(items) != 1 || items[0].Value != 180 {
		t.Fatalf("items=%#v err=%v", items, err)
	}
}
