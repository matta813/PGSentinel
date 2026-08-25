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

func TestTieredMetricRetentionAggregatesBeforeDeletingRawSamples(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "metric-tiers.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "db", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	metrics := make([]models.Metric, 0, 10002)
	for i := 0; i < 10000; i++ {
		metrics = append(metrics, models.Metric{ServerID: server.ID, Name: "connections.total", Value: float64(i % 100), CollectedAt: now.Add(-48*time.Hour + time.Duration(i)*time.Second)})
	}
	metrics = append(metrics,
		models.Metric{ServerID: server.ID, Name: "connections.total", Value: 80, CollectedAt: now.Add(-30 * time.Minute)},
		models.Metric{ServerID: server.ID, Name: "connections.total", Value: 90, CollectedAt: now.Add(-10 * time.Minute)},
	)
	if err := s.SaveMetrics(ctx, metrics); err != nil {
		t.Fatal(err)
	}
	policy := MetricRetentionPolicy{Raw: time.Hour, Medium: 30 * 24 * time.Hour, Long: 365 * 24 * time.Hour}
	if err := s.PruneMonitoringHistory(ctx, now, 7*24*time.Hour, policy); err != nil {
		t.Fatal(err)
	}
	var raw, mediumSamples, longSamples int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM metrics`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(sample_count),0) FROM metric_aggregates WHERE tier='medium'`).Scan(&mediumSamples); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(sample_count),0) FROM metric_aggregates WHERE tier='long'`).Scan(&longSamples); err != nil {
		t.Fatal(err)
	}
	if raw != 2 || mediumSamples != 10000 || longSamples != 10000 {
		t.Fatalf("raw=%d medium samples=%d long samples=%d", raw, mediumSamples, longSamples)
	}
	if err := s.PruneMonitoringHistory(ctx, now, 7*24*time.Hour, policy); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(sample_count),0) FROM metric_aggregates WHERE tier='medium'`).Scan(&mediumSamples); err != nil || mediumSamples != 10000 {
		t.Fatalf("retry duplicated samples: count=%d err=%v", mediumSamples, err)
	}
	history, err := s.ListMetrics(ctx, server.ID, "connections.total", time.Time{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) < 3 || history[len(history)-1].Value != 90 {
		t.Fatalf("tiered history did not retain newest raw points: %#v", history)
	}
	foundAggregate := false
	for _, point := range history {
		if point.Resolution == "15m" && point.Samples > 1 && point.Minimum != nil && point.Maximum != nil {
			foundAggregate = true
			break
		}
	}
	if !foundAggregate {
		t.Fatal("history did not expose aggregate resolution, sample count, and range")
	}
}

func TestMetricRetentionRejectsUnsafePolicy(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "invalid-retention.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.PruneMonitoringHistory(context.Background(), time.Now(), 24*time.Hour, MetricRetentionPolicy{Raw: 24 * time.Hour, Medium: time.Hour, Long: 365 * 24 * time.Hour}); err == nil {
		t.Fatal("descending retention policy was accepted")
	}
}
