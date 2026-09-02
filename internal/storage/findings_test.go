package storage

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestFindingLifecycle(t *testing.T) {
	s, ctx := testMonitoringStore(t, "lifecycle")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	f := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityHigh, Category: "Vacuum", Title: "T", Status: "active", StartedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{f}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListFindings(ctx, "resolved", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d: %v", len(got), err)
	}
}

func TestOpenFindingsByRulePreservesOnlyRequestedAnalyzerState(t *testing.T) {
	s, ctx := testMonitoringStore(t, "rule-findings")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	var findings []models.Finding
	for index, rule := range []string{"query-regression", "blocking-queries"} {
		finding := models.Finding{ID: rule, RuleID: rule, Fingerprint: rule, ServerID: server.ID, Severity: models.SeverityHigh, Category: "Queries", Title: rule, StartedAt: now.Add(time.Duration(index) * time.Second), UpdatedAt: now}
		findings = append(findings, finding)
	}
	if err := s.UpsertFindings(ctx, server.ID, findings); err != nil {
		t.Fatal(err)
	}
	items, err := s.OpenFindingsByRule(ctx, server.ID, "query-regression")
	if err != nil || len(items) != 1 || items[0].RuleID != "query-regression" {
		t.Fatalf("items=%#v err=%v", items, err)
	}
}

func TestAcknowledgedFindingPersistsUntilConditionResolves(t *testing.T) {
	s, ctx := testMonitoringStore(t, "acknowledged")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "rule", Fingerprint: "fingerprint", ServerID: "s", Severity: models.SeverityHigh, Category: "Connections", Title: "Saturation", Status: "active", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetFindingStatus(ctx, finding.ID, "acknowledged"); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	acknowledged, err := s.ListFindings(ctx, "acknowledged", "s")
	if err != nil || len(acknowledged) != 1 {
		t.Fatalf("acknowledged=%#v err=%v", acknowledged, err)
	}
	open, err := s.ListFindings(ctx, "open", "s")
	if err != nil || len(open) != 1 {
		t.Fatalf("open=%#v err=%v", open, err)
	}
	if err := s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.ListFindings(ctx, "resolved", "s")
	if err != nil || len(resolved) != 1 {
		t.Fatalf("resolved=%#v err=%v", resolved, err)
	}
}
