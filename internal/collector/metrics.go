package collector

import "github.com/matta813/pgsentinel/internal/models"

func snapshotMetrics(snapshot models.Snapshot) []models.Metric {
	maxReplayLag, maxReplayBytes, retainedSlotBytes := 0.0, 0.0, 0.0
	for _, standby := range snapshot.Replication.Standbys {
		if standby.ReplayLagSeconds > maxReplayLag {
			maxReplayLag = standby.ReplayLagSeconds
		}
		if standby.PendingReplayBytes > maxReplayBytes {
			maxReplayBytes = standby.PendingReplayBytes
		}
	}
	for _, slot := range snapshot.Replication.Slots {
		retainedSlotBytes += slot.RetainedBytes
	}
	values := map[string]float64{
		"connections.active": float64(snapshot.Connections.Active), "connections.total": float64(snapshot.Connections.Total),
		"connections.utilization": snapshot.Connections.Utilization, "connections.waiting": float64(snapshot.Connections.Waiting),
		"server.uptime_seconds": snapshot.UptimeSeconds, "wal.generation_bytes_per_second": snapshot.WAL.GenerationBytesPerSecond,
		"wal.bytes_total": snapshot.WAL.WALBytes, "replication.max_replay_lag_seconds": maxReplayLag,
		"replication.max_pending_replay_bytes": maxReplayBytes, "replication.slot_retained_bytes": retainedSlotBytes,
	}
	out := make([]models.Metric, 0, len(values))
	for name, value := range values {
		out = append(out, models.Metric{ServerID: snapshot.ServerID, Name: name, Value: value, CollectedAt: snapshot.CollectedAt})
	}
	return out
}
