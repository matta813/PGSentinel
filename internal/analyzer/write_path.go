package analyzer

import (
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func EnrichWritePath(previousWAL models.WALStats, previousReplication models.ReplicationStats, currentWAL *models.WALStats, currentReplication *models.ReplicationStats) {
	seconds := currentWAL.CollectedAt.Sub(previousWAL.CollectedAt).Seconds()
	if seconds >= 10 && seconds <= time.Hour.Seconds() && sameReset(previousWAL.WALStatsReset, currentWAL.WALStatsReset) && currentWAL.WALBytes >= previousWAL.WALBytes {
		currentWAL.GenerationBytesPerSecond = (currentWAL.WALBytes - previousWAL.WALBytes) / seconds
		if currentWAL.WALBuffersFull >= previousWAL.WALBuffersFull {
			currentWAL.BufferFullEventsPerSecond = (currentWAL.WALBuffersFull - previousWAL.WALBuffersFull) / seconds
		}
	}
	replicationSeconds := currentReplication.CollectedAt.Sub(previousReplication.CollectedAt).Seconds()
	if replicationSeconds < 10 || replicationSeconds > time.Hour.Seconds() {
		return
	}
	standbys := map[string]models.ReplicationStandby{}
	for _, standby := range previousReplication.Standbys {
		standbys[standby.Application+"|"+standby.ClientAddress] = standby
	}
	for i := range currentReplication.Standbys {
		current := &currentReplication.Standbys[i]
		if previous, ok := standbys[current.Application+"|"+current.ClientAddress]; ok {
			current.PendingReplayGrowthBytesPerSecond = (current.PendingReplayBytes - previous.PendingReplayBytes) / replicationSeconds
		}
	}
	slots := map[string]models.ReplicationSlot{}
	for _, slot := range previousReplication.Slots {
		slots[slot.Name] = slot
	}
	for i := range currentReplication.Slots {
		current := &currentReplication.Slots[i]
		if previous, ok := slots[current.Name]; ok {
			current.RetainedGrowthBytesPerSecond = (current.RetainedBytes - previous.RetainedBytes) / replicationSeconds
		}
	}
}
func sameReset(left, right *time.Time) bool { return left != nil && right != nil && left.Equal(*right) }
