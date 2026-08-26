package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestIncidentPersistenceAndTimeline(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "incidents.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "primary", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	findings := []models.Finding{
		{ID: "lock", RuleID: "blocking-queries", Fingerprint: "lock-fp", ServerID: server.ID, Category: "Locks", Severity: models.SeverityHigh, Title: "Blocked queries", Summary: "Work is blocked.", Status: "active", StartedAt: now, UpdatedAt: now},
		{ID: "connection", RuleID: "connection-utilization", Fingerprint: "connection-fp", ServerID: server.ID, Category: "Connections", Severity: models.SeverityCritical, Title: "Connection pressure", Summary: "Capacity is low.", Status: "active", StartedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute)},
	}
	if err := store.UpsertFindings(ctx, server.ID, findings); err != nil {
		t.Fatal(err)
	}
	if err := store.RebuildIncidents(ctx, server.ID, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	items, err := store.ListIncidents(ctx, IncidentFilter{Status: "active", ServerID: server.ID, Limit: 20})
	if err != nil || len(items) != 1 || items[0].Severity != models.SeverityCritical {
		t.Fatalf("incidents=%#v err=%v", items, err)
	}
	detail, err := store.GetIncident(ctx, items[0].ID)
	if err != nil || len(detail.Findings) != 2 || len(detail.Timeline) != 2 || len(detail.Rationale) == 0 {
		t.Fatalf("detail=%#v err=%v", detail, err)
	}
	if detail.Timeline[0].At.After(detail.Timeline[1].At) {
		t.Fatalf("timeline is not chronological: %#v", detail.Timeline)
	}
	if err := store.UpsertFindings(ctx, server.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.RebuildIncidents(ctx, server.ID, now.Add(20*time.Minute)); err != nil {
		t.Fatal(err)
	}
	resolved, err := store.GetIncident(ctx, items[0].ID)
	if err != nil || resolved.Status != "resolved" || resolved.ResolvedAt == nil || len(resolved.Timeline) != 4 {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
}

func TestIncidentQueriesValidatePagination(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "incident-pagination.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.ListIncidents(context.Background(), IncidentFilter{Limit: 101}); err == nil {
		t.Fatal("unbounded incident page was accepted")
	}
}
