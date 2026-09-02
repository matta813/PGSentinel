package analyzer

import (
	"fmt"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeWALArchive(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	if (s.WAL.ArchiveMode == "on" || s.WAL.ArchiveMode == "always") && !s.WAL.ArchiveConfigured {
		out = append(out, newFinding("archive-not-configured", s.ServerID, "", "", models.SeverityHigh, "WAL", "WAL archiving is enabled but has no destination", "archive_mode is enabled while both archive_command and archive_library are empty.", "Completed WAL segments can accumulate locally and the expected archive recovery chain is not being produced.", models.ConfidenceHigh, []models.Evidence{{Label: "Archive mode", Value: s.WAL.ArchiveMode}, {Label: "Archive destination", Value: "not configured"}}))
	}
	if s.WAL.ArchiveMode != "off" && s.WAL.ArchiveMode != "" && s.WAL.LastFailedAt != nil && (s.WAL.LastArchivedAt == nil || s.WAL.LastFailedAt.After(*s.WAL.LastArchivedAt)) {
		out = append(out, newFinding("archive-failure-current", s.ServerID, "", "", models.SeverityHigh, "WAL", "Latest WAL archive attempt failed", "The newest archive status is a failure with no later successful archive recorded.", "Repeated failures can retain WAL on the database server and weaken archive-based recovery objectives.", models.ConfidenceHigh, []models.Evidence{{Label: "Failed WAL", Value: s.WAL.LastFailedWAL}, {Label: "Last failure", Value: s.WAL.LastFailedAt.UTC().Format(time.RFC3339)}, {Label: "Failed attempts", Value: fmt.Sprintf("%.0f", s.WAL.FailedArchiveCount)}, {Label: "Last archived WAL", Value: s.WAL.LastArchivedWAL}}))
	}
	return out
}

func (e *Engine) analyzeWALCheckpoints(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	t := e.Thresholds
	restartpointTotal := s.WAL.RestartpointsTimed + s.WAL.RestartpointsRequested
	if s.Replication.InRecovery && restartpointTotal >= 10 && s.WAL.RestartpointsRequested/restartpointTotal*100 >= t.RequestedCheckpointRatio {
		out = append(out, newFinding("requested-restartpoints", s.ServerID, "", "", models.SeverityMedium, "WAL", "Requested restartpoints are frequent", fmt.Sprintf("%.1f%% of %.0f observed restartpoint attempts were requested; %.0f completed.", s.WAL.RestartpointsRequested/restartpointTotal*100, restartpointTotal, s.WAL.RestartpointsDone), "Frequent requested restartpoints can indicate WAL pressure during recovery and contribute to storage write bursts.", models.ConfidenceMedium, []models.Evidence{{Label: "Requested restartpoints", Value: fmt.Sprintf("%.0f", s.WAL.RestartpointsRequested)}, {Label: "Timed restartpoints", Value: fmt.Sprintf("%.0f", s.WAL.RestartpointsTimed)}, {Label: "Completed restartpoints", Value: fmt.Sprintf("%.0f", s.WAL.RestartpointsDone)}}))
	}
	checkpointTotal := s.WAL.TimedCheckpoints + s.WAL.RequestedCheckpoints
	if !s.Replication.InRecovery && checkpointTotal >= 10 {
		requestedRatio := s.WAL.RequestedCheckpoints / checkpointTotal * 100
		if requestedRatio >= t.RequestedCheckpointRatio {
			out = append(out, newFinding("requested-checkpoints", s.ServerID, "", "", models.SeverityMedium, "WAL", "Requested checkpoints are frequent", fmt.Sprintf("%.1f%% of %.0f checkpoints were requested rather than timed.", requestedRatio, checkpointTotal), "Frequent requested checkpoints can increase write bursts and indicate WAL pressure or undersized max_wal_size.", models.ConfidenceMedium, []models.Evidence{{Label: "Requested", Value: fmt.Sprintf("%.0f", s.WAL.RequestedCheckpoints)}, {Label: "Timed", Value: fmt.Sprintf("%.0f", s.WAL.TimedCheckpoints)}, {Label: "Requested ratio", Value: fmt.Sprintf("%.1f%%", requestedRatio)}}))
		}
		if s.WAL.StatsReset != nil {
			interval := time.Since(*s.WAL.StatsReset).Seconds() / checkpointTotal
			if interval > 0 && interval < t.CheckpointIntervalSeconds {
				out = append(out, newFinding("checkpoint-frequency", s.ServerID, "", "", models.SeverityMedium, "WAL", "Checkpoints are occurring frequently", fmt.Sprintf("The average interval since the statistics reset is %.0f seconds.", interval), "Frequent checkpoints can create avoidable write pressure and latency variability.", models.ConfidenceMedium, []models.Evidence{{Label: "Average interval", Value: fmt.Sprintf("%.0f seconds", interval)}, {Label: "Write time", Value: fmt.Sprintf("%.0f ms", s.WAL.WriteTimeMS)}, {Label: "Sync time", Value: fmt.Sprintf("%.0f ms", s.WAL.SyncTimeMS)}}))
			}
		}
	}
	return out
}
