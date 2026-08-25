package collector

import (
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestSnapshotMetricsExtractsCoreTimeSeries(t *testing.T) {
	at := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	snapshot := models.Snapshot{ServerID: "server", CollectedAt: at, UptimeSeconds: 1234, Connections: models.ConnectionStats{Active: 4, Total: 12, Waiting: 2, Utilization: 24.5}, WAL: models.WALStats{WALBytes: 2048, GenerationBytesPerSecond: 100}, Replication: models.ReplicationStats{Standbys: []models.ReplicationStandby{{ReplayLagSeconds: 20, PendingReplayBytes: 4096}}, Slots: []models.ReplicationSlot{{RetainedBytes: 8192}}}}
	metrics := snapshotMetrics(snapshot)
	got := map[string]float64{}
	for _, metric := range metrics {
		if metric.ServerID != snapshot.ServerID || !metric.CollectedAt.Equal(at) {
			t.Fatalf("metric lost identity or timestamp: %#v", metric)
		}
		got[metric.Name] = metric.Value
	}
	want := map[string]float64{"connections.active": 4, "connections.total": 12, "connections.waiting": 2, "connections.utilization": 24.5, "server.uptime_seconds": 1234, "wal.bytes_total": 2048, "wal.generation_bytes_per_second": 100, "replication.max_replay_lag_seconds": 20, "replication.max_pending_replay_bytes": 4096, "replication.slot_retained_bytes": 8192}
	for name, value := range want {
		if got[name] != value {
			t.Fatalf("%s=%v, want %v", name, got[name], value)
		}
	}
}
