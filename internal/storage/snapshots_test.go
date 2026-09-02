package storage

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestWaitEventSnapshotRoundTripReturnsLatestRealSample(t *testing.T) {
	s, ctx := testMonitoringStore(t, "wait-events")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	first := []models.WaitEventSample{{PID: 10, WaitEventType: "Lock", WaitEvent: "tuple", Query: "select first", QueryAgeSeconds: 5}}
	latest := []models.WaitEventSample{{PID: 11, WaitEventType: "FutureClass", WaitEvent: "FutureWait", Query: "select latest", QueryAgeSeconds: 7}}
	now := time.Now().UTC()
	if err := s.SaveSnapshot(ctx, server.ID, "wait-events", first, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveSnapshot(ctx, server.ID, "wait-events", latest, now); err != nil {
		t.Fatal(err)
	}
	var got []models.WaitEventSample
	if err := s.LatestSnapshot(ctx, server.ID, "wait-events", &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].PID != 11 || got[0].WaitEventType != "FutureClass" {
		t.Fatalf("got %#v", got)
	}
}
