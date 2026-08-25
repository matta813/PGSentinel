package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestNotificationRoutesMatchDeterministicallyAndDedupe(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "routing.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "Primary", Host: "db", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable", Tags: []string{"Production", "DBA"}}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pager", "dba"} {
		d := models.NotificationDestination{ID: id, Name: id, Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com"}}
		if err := s.CreateNotificationDestination(ctx, &d); err != nil {
			t.Fatal(err)
		}
	}
	routes := []models.NotificationRoute{
		{ID: "route-b", Name: "replication", Enabled: true, Priority: 200, Categories: []string{"replication"}, ServerTags: []string{"dba"}, Transitions: []string{"new"}, DestinationIDs: []string{"dba", "pager"}},
		{ID: "route-a", Name: "critical", Enabled: true, Priority: 100, Severities: []string{"CRITICAL"}, DestinationIDs: []string{"pager"}},
	}
	for i := range routes {
		if err := s.CreateNotificationRoute(ctx, &routes[i]); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "replica_lag", Fingerprint: "fp", ServerID: server.ID, Severity: models.SeverityCritical, Category: "Replication", Title: "Lag", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, server.ID, []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	pending, err := s.PendingFindingNotifications(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || pending[0].DestinationID != "dba" || pending[1].DestinationID != "pager" {
		t.Fatalf("pending=%#v", pending)
	}
}

func TestNotificationDeliveryHistoryIsBounded(t *testing.T) {
	s, ctx := testMonitoringStore(t, "bounded-history")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	destination := models.NotificationDestination{ID: "d", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com"}}
	if err := s.CreateNotificationDestination(ctx, &destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.DB.Exec(`INSERT INTO findings(id,rule_id,fingerprint,server_id,severity,category,title,summary,cause,impact,evidence_json,suggestions_json,confidence,status,started_at,updated_at) VALUES('f','r','fp','s','HIGH','WAL','title','','','','[]','[]','HIGH','active',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	tx, err := s.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2005; i++ {
		id := fmt.Sprintf("event-%04d", i)
		if _, err := tx.Exec(`INSERT INTO finding_notification_events(id,finding_id,event_type,finding_json,created_at) VALUES(?,'f','new','{}',?)`, id, fmt.Sprintf("2026-08-25T00:%02d:%02dZ", (i/60)%60, i%60)); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(`INSERT INTO finding_notification_deliveries(event_id,destination_id,status) VALUES(?,'d','delivered')`, id); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := s.pruneNotificationHistory(ctx, 2000); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM finding_notification_deliveries`).Scan(&count); err != nil || count != 2000 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestExplicitRoutesCanDeliverLowerSeverityAndApplyCooldown(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "cooldown.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "db", Host: "db", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	_ = s.CreateServer(ctx, &server)
	destination := models.NotificationDestination{ID: "ops", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com"}}
	_ = s.CreateNotificationDestination(ctx, &destination)
	route := models.NotificationRoute{ID: "route", Name: "all", Enabled: true, Priority: 1, Severities: []string{"MEDIUM"}, DestinationIDs: []string{"ops"}, CooldownSeconds: 3600}
	if err := s.CreateNotificationRoute(ctx, &route); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "r", Fingerprint: "fp", ServerID: "server", Severity: models.SeverityMedium, Category: "Locks", Title: "Blocked", StartedAt: now, UpdatedAt: now}
	if err := s.UpsertFindings(ctx, "server", []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	pending, _ := s.PendingFindingNotifications(ctx, 10)
	if len(pending) != 1 {
		t.Fatalf("medium pending=%#v", pending)
	}
	if err := s.RecordFindingNotification(ctx, pending[0].EventID, "ops", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertFindings(ctx, "server", nil); err != nil {
		t.Fatal(err)
	}
	if pending, _ := s.PendingFindingNotifications(ctx, 10); len(pending) != 0 {
		t.Fatalf("cooldown delivery pending=%#v", pending)
	}
	history, err := s.ListNotificationDeliveryHistory(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Status != "cooldown" {
		t.Fatalf("history=%#v", history)
	}
}

func TestDeliveryHistoryShowsBoundedRetryWithoutTargetSecrets(t *testing.T) {
	s, ctx := testMonitoringStore(t, "history")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	_ = s.CreateServer(ctx, &server)
	destination := models.NotificationDestination{ID: "d", Name: "pager", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": "https://example.com"}}
	_ = s.CreateNotificationDestination(ctx, &destination)
	now := time.Now().UTC()
	finding := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityHigh, Category: "WAL", Title: "Pressure", StartedAt: now, UpdatedAt: now}
	_ = s.UpsertFindings(ctx, "s", []models.Finding{finding})
	pending, _ := s.PendingFindingNotifications(ctx, 10)
	secret := "token-secret"
	if err := s.RecordFindingNotification(ctx, pending[0].EventID, "d", errors.New("POST https://user:"+secret+"@hooks.example/path failed")); err != nil {
		t.Fatal(err)
	}
	_, err := s.ListNotificationDeliveryHistory(ctx, 201, 0)
	if err == nil {
		t.Fatal("unbounded limit accepted")
	}
	history, err := s.ListNotificationDeliveryHistory(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Status != "retry" || history[0].NextAttemptAt == nil || strings.Contains(history[0].LastError, secret) {
		t.Fatalf("history=%#v", history)
	}
}
