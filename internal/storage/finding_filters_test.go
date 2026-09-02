package storage

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestFindingFiltersCanBeCombined(t *testing.T) {
	s, ctx := testMonitoringStore(t, "filters")
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
	got, err := s.FilterFindings(ctx, FindingFilter{Status: "active", Database: "warehouse", Severity: "HIGH", Category: "vacuum", Search: "warehouse"})
	if err != nil || len(got) != 1 || got[0].ID != "vacuum" {
		t.Fatalf("combined filter=%#v err=%v", got, err)
	}
	got, err = s.FilterFindings(ctx, FindingFilter{Search: "POOL IS EXHAUSTED"})
	if err != nil || len(got) != 1 || got[0].ID != "connection" {
		t.Fatalf("case-insensitive search=%#v err=%v", got, err)
	}
	got, err = s.FilterFindings(ctx, FindingFilter{Database: "missing"})
	if err != nil || len(got) != 0 {
		t.Fatalf("database filter=%#v err=%v", got, err)
	}
}
