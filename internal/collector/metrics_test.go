package collector

import (
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestSnapshotMetricsExtractsCoreTimeSeries(t *testing.T) {
	at := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	snapshot := models.Snapshot{ServerID: "server", CollectedAt: at, UptimeSeconds: 1234, Connections: models.ConnectionStats{Active: 4, Total: 12, Waiting: 2, Utilization: 24.5}}
	metrics := snapshotMetrics(snapshot)
	got := map[string]float64{}
	for _, metric := range metrics {
		if metric.ServerID != snapshot.ServerID || !metric.CollectedAt.Equal(at) {
			t.Fatalf("metric lost identity or timestamp: %#v", metric)
		}
		got[metric.Name] = metric.Value
	}
	want := map[string]float64{"connections.active": 4, "connections.total": 12, "connections.waiting": 2, "connections.utilization": 24.5, "server.uptime_seconds": 1234}
	for name, value := range want {
		if got[name] != value {
			t.Fatalf("%s=%v, want %v", name, got[name], value)
		}
	}
}
