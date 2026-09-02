package storage

import (
	"errors"
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestFindingTransitionsQueueEachDestinationOnce(t *testing.T) {
	s, ctx := testMonitoringStore(t, "finding-notifications")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	destination := models.NotificationDestination{ID: "destination", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com/hook"}}
	if err := s.CreateNotificationDestination(ctx, &destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "rule", Fingerprint: "fingerprint", ServerID: "s", Severity: models.SeverityHigh, Category: "Replication", Title: "Lag", Status: "active", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingFindingNotifications(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].EventType != "new" {
		t.Fatalf("pending=%#v err=%v", pending, err)
	}
	if err := s.RecordFindingNotification(ctx, pending[0].EventID, pending[0].DestinationID, nil); err != nil {
		t.Fatal(err)
	}
	if pending, _ = s.PendingFindingNotifications(ctx, 10); len(pending) != 0 {
		t.Fatalf("delivered event queued again: %#v", pending)
	}
	if err := s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	pending, _ = s.PendingFindingNotifications(ctx, 10)
	if len(pending) != 1 || pending[0].EventType != "resolved" {
		t.Fatalf("resolution not queued: %#v", pending)
	}
	if err := s.RecordFindingNotification(ctx, pending[0].EventID, pending[0].DestinationID, nil); err != nil {
		t.Fatal(err)
	}
	finding.Severity = models.SeverityCritical
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	pending, _ = s.PendingFindingNotifications(ctx, 10)
	if len(pending) != 1 || pending[0].EventType != "reopened" {
		t.Fatalf("reopen not queued: %#v", pending)
	}
	if err := s.RecordFindingNotification(ctx, pending[0].EventID, pending[0].DestinationID, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	if pending, _ = s.PendingFindingNotifications(ctx, 10); len(pending) != 0 {
		t.Fatalf("unchanged finding queued: %#v", pending)
	}
}

func TestFindingNotificationRetriesAreBounded(t *testing.T) {
	s, ctx := testMonitoringStore(t, "notification-retries")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	destination := models.NotificationDestination{ID: "d", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com"}}
	if err := s.CreateNotificationDestination(ctx, &destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityHigh, Category: "WAL", Title: "Pressure", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		pending, _ := s.PendingFindingNotifications(ctx, 10)
		if len(pending) != 1 {
			t.Fatalf("attempt %d pending=%#v", attempt, pending)
		}
		if err := s.RecordFindingNotification(ctx, pending[0].EventID, pending[0].DestinationID, errors.New("temporary failure")); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DB.Exec(`UPDATE finding_notification_deliveries SET next_attempt_at='2000-01-01T00:00:00Z' WHERE status='retry'`); err != nil {
			t.Fatal(err)
		}
	}
	if pending, _ := s.PendingFindingNotifications(ctx, 10); len(pending) != 0 {
		t.Fatalf("delivery exceeded retry bound: %#v", pending)
	}
}

func TestSeverityIncreaseQueuesFirstActionableNotification(t *testing.T) {
	s, ctx := testMonitoringStore(t, "notification-escalation")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	destination := models.NotificationDestination{ID: "d", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com"}}
	if err := s.CreateNotificationDestination(ctx, &destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityMedium, Category: "Replication", Title: "Lag", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	if pending, _ := s.PendingFindingNotifications(ctx, 10); len(pending) != 0 {
		t.Fatalf("medium finding alerted: %#v", pending)
	}
	finding.Severity = models.SeverityHigh
	if err := s.UpsertFindings(ctx, "s", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	pending, _ := s.PendingFindingNotifications(ctx, 10)
	if len(pending) != 1 || pending[0].EventType != "severity_increased" {
		t.Fatalf("escalation not queued: %#v", pending)
	}
}
