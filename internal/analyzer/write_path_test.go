package analyzer

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
	"time"
)

func TestEnrichWritePathUsesCompatibleIntervals(t *testing.T) {
	reset := time.Now().Add(-time.Hour)
	start := time.Now()
	previousWAL := models.WALStats{CollectedAt: start, WALBytes: 1000, WALBuffersFull: 2, WALStatsReset: &reset}
	currentWAL := models.WALStats{CollectedAt: start.Add(time.Minute), WALBytes: 7000, WALBuffersFull: 8, WALStatsReset: &reset}
	previousReplication := models.ReplicationStats{CollectedAt: start, Standbys: []models.ReplicationStandby{{Application: "r", PendingReplayBytes: 100}}, Slots: []models.ReplicationSlot{{Name: "s", RetainedBytes: 1000}}}
	currentReplication := models.ReplicationStats{CollectedAt: start.Add(time.Minute), Standbys: []models.ReplicationStandby{{Application: "r", PendingReplayBytes: 700}}, Slots: []models.ReplicationSlot{{Name: "s", RetainedBytes: 7000}}}
	EnrichWritePath(previousWAL, previousReplication, &currentWAL, &currentReplication)
	if currentWAL.GenerationBytesPerSecond != 100 || currentReplication.Standbys[0].PendingReplayGrowthBytesPerSecond != 10 || currentReplication.Slots[0].RetainedGrowthBytesPerSecond != 100 {
		t.Fatalf("unexpected rates: %#v %#v", currentWAL, currentReplication)
	}
}
func TestEnrichWritePathRejectsCounterReset(t *testing.T) {
	left, right := time.Now().Add(-time.Hour), time.Now()
	start := time.Now()
	previous := models.WALStats{CollectedAt: start, WALBytes: 1000, WALStatsReset: &left}
	current := models.WALStats{CollectedAt: start.Add(time.Minute), WALBytes: 2000, WALStatsReset: &right}
	EnrichWritePath(previous, models.ReplicationStats{}, &current, &models.ReplicationStats{})
	if current.GenerationBytesPerSecond != 0 {
		t.Fatal("rate derived across reset")
	}
}
