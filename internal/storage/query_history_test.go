package storage

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestRecentQueryObservationsAreChronologicalAndBounded(t *testing.T) {
	s, ctx := testMonitoringStore(t, "query-observations")
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	for index := 0; index < 55; index++ {
		at := start.Add(time.Duration(index) * time.Minute)
		observation := models.QueryObservation{CollectedAt: at, Queries: []models.QueryStat{{QueryID: "42", Calls: float64(index)}}}
		if err := s.SaveSnapshot(ctx, server.ID, "query-regression", observation, at); err != nil {
			t.Fatal(err)
		}
	}
	items, err := s.RecentQueryObservations(ctx, server.ID, 1000)
	if err != nil || len(items) != 50 {
		t.Fatalf("len=%d err=%v", len(items), err)
	}
	if !items[0].CollectedAt.Before(items[len(items)-1].CollectedAt) || items[0].Queries[0].Calls != 5 {
		t.Fatalf("order=%#v", items)
	}
}
