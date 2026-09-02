package notifications

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

func TestDispatcherSendsFindingContextAndMarksDelivery(t *testing.T) {
	var message Message
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &message)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	store, err := storage.Open(filepath.Join(t.TempDir(), "dispatcher.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "Production primary", Host: "db", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	destination := models.NotificationDestination{ID: "destination", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": target.URL}}
	if err := store.CreateNotificationDestination(ctx, &destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "replica", Fingerprint: "fingerprint", ServerID: server.ID, Severity: models.SeverityHigh, Category: "Replication", Title: "Replica lag", Summary: "Replay is delayed.", Evidence: []models.Evidence{{Label: "Lag", Value: "90 seconds"}}, Suggestions: []models.Suggestion{{Title: "Check receiver state."}}, StartedAt: now, UpdatedAt: now}
	if err := store.UpsertFindings(ctx, server.ID, []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}
	NewDispatcher(store, NewTargetPolicy(true, nil), slog.Default()).DispatchPending(ctx)
	if message.Title != "PGSentinel: Replica lag" || !strings.Contains(message.Body, "Server: Production primary") || !strings.Contains(message.Body, "Evidence: Lag = 90 seconds") {
		t.Fatalf("unexpected message: %#v", message)
	}
	if pending, err := store.PendingFindingNotifications(ctx, 10); err != nil || len(pending) != 0 {
		t.Fatalf("pending=%#v err=%v", pending, err)
	}
}

func TestDispatcherDoesNotSendPendingDeliveryTwiceConcurrently(t *testing.T) {
	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		once.Do(func() { close(started) })
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()

	store, err := storage.Open(filepath.Join(t.TempDir(), "concurrent-dispatcher.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "Primary", Host: "db", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	destination := models.NotificationDestination{ID: "destination", Name: "ops", Provider: "webhook", Enabled: true, Config: map[string]string{"webhookUrl": target.URL}}
	if err := store.CreateNotificationDestination(ctx, &destination); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	finding := models.Finding{ID: "finding", RuleID: "replica", Fingerprint: "concurrent-fingerprint", ServerID: server.ID, Severity: models.SeverityHigh, Category: "Replication", Title: "Replica lag", Summary: "Replay is delayed.", StartedAt: now, UpdatedAt: now}
	if err := store.UpsertFindings(ctx, server.ID, []models.Finding{finding}); err != nil {
		t.Fatal(err)
	}

	dispatcher := NewDispatcher(store, NewTargetPolicy(true, nil), slog.Default())
	done := make(chan struct{}, 2)
	go func() { dispatcher.DispatchPending(ctx); done <- struct{}{} }()
	<-started
	go func() { dispatcher.DispatchPending(ctx); done <- struct{}{} }()
	close(release)
	<-done
	<-done
	if got := calls.Load(); got != 1 {
		t.Fatalf("webhook calls = %d, want 1", got)
	}
}
