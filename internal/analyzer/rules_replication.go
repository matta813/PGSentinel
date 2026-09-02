package analyzer

import (
	"fmt"
	"strings"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeReplicationState(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	delayedReplica := hasTag(s.ServerTags, "delayed-replica") || hasTag(s.ServerTags, "allow-delayed-replicas") || s.Replication.ReplayDelaySeconds > 0
	if s.Replication.InRecovery {
		if s.Replication.Receiver == nil {
			out = append(out, newFinding("wal-receiver-disconnected", s.ServerID, "", "", models.SeverityHigh, "Replication", "Replica is not receiving WAL", "PostgreSQL reports recovery mode but no WAL receiver process is visible.", "The replica may stop replaying new changes and fall behind its upstream.", models.ConfidenceHigh, []models.Evidence{{Label: "Server role", Value: "replica"}, {Label: "WAL receiver", Value: "not running"}}))
		} else if s.Replication.Receiver.Status != "streaming" {
			out = append(out, newFinding("wal-receiver-state", s.ServerID, "", "", models.SeverityHigh, "Replication", "Replica WAL receiver is not streaming", fmt.Sprintf("The WAL receiver state is %s.", s.Replication.Receiver.Status), "Replication progress may be interrupted until streaming resumes.", models.ConfidenceHigh, []models.Evidence{{Label: "Receiver state", Value: s.Replication.Receiver.Status}, {Label: "Last message age", Value: fmt.Sprintf("%.0f seconds", s.Replication.Receiver.LastMessageSeconds)}}))
		}
		if s.Replication.RecoveryPaused {
			f := newFinding("recovery-paused", s.ServerID, "", "", models.SeverityMedium, "Replication", "WAL replay is paused", "PostgreSQL reports that recovery replay is paused.", "Queries on this standby will not see newer changes until replay resumes.", models.ConfidenceHigh, []models.Evidence{{Label: "Replay state", Value: "paused"}, {Label: "Replay LSN", Value: s.Replication.ReplayLSN}, {Label: "Timeline", Value: fmt.Sprint(s.Replication.TimelineID)}})
			f.Suggestions = []models.Suggestion{{Title: "Confirm whether replay was paused intentionally", Detail: "Inspect the standby recovery procedure and owning change record before asking an administrator to resume replay."}}
			out = append(out, f)
		}
	}
	for _, standby := range s.Replication.Standbys {
		resource := standby.Application
		if resource == "" {
			resource = standby.ClientAddress
		}
		if standby.State != "streaming" {
			out = append(out, newFinding("standby-state", s.ServerID, "", resource, models.SeverityHigh, "Replication", "Connected standby is not streaming", fmt.Sprintf("Standby %s reports replication state %s.", resource, standby.State), "The standby may not be receiving current WAL from this primary.", models.ConfidenceHigh, []models.Evidence{{Label: "State", Value: standby.State}, {Label: "Sync mode", Value: standby.SyncState}}))
		} else if standby.ReplayLagSeconds >= e.Thresholds.ReplicaLagSeconds && !delayedReplica {
			out = append(out, newFinding("standby-replay-lag", s.ServerID, "", resource, models.SeverityMedium, "Replication", "Replica replay lag is elevated", fmt.Sprintf("Standby %s reports %.1f seconds of replay lag and %.0f bytes pending replay.", resource, standby.ReplayLagSeconds, standby.PendingReplayBytes), "Reads from the replica may observe older data and recovery objectives may be at risk if lag continues.", models.ConfidenceMedium, []models.Evidence{{Label: "Replay lag", Value: fmt.Sprintf("%.1f seconds", standby.ReplayLagSeconds)}, {Label: "Pending replay", Value: fmt.Sprintf("%.0f bytes", standby.PendingReplayBytes)}, {Label: "Pending network send", Value: fmt.Sprintf("%.0f bytes", standby.PendingSendBytes)}, {Label: "Reply age", Value: fmt.Sprintf("%.1f seconds", standby.ReplyAgeSeconds)}, {Label: "Sync mode", Value: standby.SyncState}}))
		}
	}
	return out
}

func (e *Engine) analyzeReplicationSlots(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	for _, slot := range s.Replication.Slots {
		if slot.WALStatus == "lost" {
			out = append(out, newFinding("replication-slot-lost", s.ServerID, slot.Database, slot.Name, models.SeverityCritical, "Replication", "Replication slot has lost required WAL", fmt.Sprintf("Slot %s reports wal_status=lost; its consumer can no longer continue from the retained position.", slot.Name), "The replication consumer may require deliberate reinitialization before it can resume safely.", models.ConfidenceHigh, []models.Evidence{{Label: "Slot", Value: slot.Name}, {Label: "Slot type", Value: slot.Type}, {Label: "WAL status", Value: slot.WALStatus}, {Label: "Active", Value: fmt.Sprint(slot.Active)}}))
			continue
		}
		if !slot.Active && slot.RetainedBytes >= e.Thresholds.SlotRetainedBytes {
			out = append(out, newFinding("inactive-slot-wal", s.ServerID, slot.Database, slot.Name, models.SeverityHigh, "Replication", "Inactive replication slot is retaining WAL", fmt.Sprintf("Slot %s is inactive and retains approximately %.0f bytes of WAL.", slot.Name, slot.RetainedBytes), "Retained WAL can continue consuming disk until the slot advances or is deliberately removed.", models.ConfidenceHigh, []models.Evidence{{Label: "Slot", Value: slot.Name}, {Label: "Retained WAL", Value: fmt.Sprintf("%.0f bytes", slot.RetainedBytes)}, {Label: "Retention growth", Value: fmt.Sprintf("%.1f bytes/second", slot.RetainedGrowthBytesPerSecond)}, {Label: "Inactive age", Value: fmt.Sprintf("%.0f seconds", slot.InactiveSeconds)}, {Label: "WAL status", Value: slot.WALStatus}}))
		}
	}
	return out
}

func hasTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, wanted) {
			return true
		}
	}
	return false
}
