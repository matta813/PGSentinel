package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestMetricHistoryFiltersOrdersAndPrunes(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "metrics.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "db", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	metrics := []models.Metric{
		{ServerID: server.ID, Name: "connections.total", Value: 12, CollectedAt: base.Add(2 * time.Minute)},
		{ServerID: server.ID, Name: "connections.active", Value: 3, CollectedAt: base},
		{ServerID: server.ID, Name: "connections.total", Value: 11, Labels: map[string]string{"source": "postgres"}, CollectedAt: base.Add(time.Minute)},
	}
	if err := s.SaveMetrics(ctx, metrics); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListMetrics(ctx, server.ID, "connections.total", base.Add(30*time.Second), 10)
	if err != nil || len(got) != 2 || got[0].Value != 11 || got[1].Value != 12 || got[0].Labels["source"] != "postgres" {
		t.Fatalf("history=%#v err=%v", got, err)
	}
	if err := s.Prune(ctx, base.Add(90*time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err = s.ListMetrics(ctx, server.ID, "connections.total", time.Time{}, 10)
	if err != nil || len(got) != 1 || got[0].Value != 12 {
		t.Fatalf("after prune=%#v err=%v", got, err)
	}
}

func TestMetricHistoryLimitKeepsNewestSamplesInChronologicalOrder(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "metrics-limit.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "db", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	metrics := []models.Metric{
		{ServerID: server.ID, Name: "connections.total", Value: 10, CollectedAt: base},
		{ServerID: server.ID, Name: "connections.total", Value: 11, CollectedAt: base.Add(time.Minute)},
		{ServerID: server.ID, Name: "connections.total", Value: 12, CollectedAt: base.Add(2 * time.Minute)},
	}
	if err := s.SaveMetrics(ctx, metrics); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListMetrics(ctx, server.ID, "connections.total", time.Time{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Value != 11 || got[1].Value != 12 {
		t.Fatalf("limited history=%#v, want newest values [11, 12]", got)
	}
}
