package analyzer

import (
	"fmt"
	"github.com/matta813/pgsentinel/internal/models"
	"strings"
	"time"
)

type Thresholds struct{ ConnectionHigh, ConnectionCritical, IdleTransactionSeconds, LongTransactionSeconds, LongQuerySeconds, DeadTupleRatio, VacuumProgress, CacheHitLow, RollbackRatio, QueryImpactHigh, ReplicaLagSeconds, SlotRetainedBytes, RequestedCheckpointRatio, CheckpointIntervalSeconds float64 }

func DefaultThresholds() Thresholds {
	return Thresholds{80, 95, 300, 900, 60, 20, 100, 95, 5, 250, 60, 1024 * 1024 * 1024, 20, 300}
}

type Engine struct{ Thresholds Thresholds }

func New(t Thresholds) *Engine { return &Engine{Thresholds: t} }
func (e *Engine) Analyze(s models.Snapshot) []models.Finding {
	t := e.Thresholds
	out := []models.Finding{}
	add := func(f models.Finding) { out = append(out, f) }
	delayedReplica := hasTag(s.ServerTags, "delayed-replica") || hasTag(s.ServerTags, "allow-delayed-replicas") || s.Replication.ReplayDelaySeconds > 0
	if s.Replication.InRecovery {
		if s.Replication.Receiver == nil {
			add(newFinding("wal-receiver-disconnected", s.ServerID, "", "", models.SeverityHigh, "Replication", "Replica is not receiving WAL", "PostgreSQL reports recovery mode but no WAL receiver process is visible.", "The replica may stop replaying new changes and fall behind its upstream.", models.ConfidenceHigh, []models.Evidence{{Label: "Server role", Value: "replica"}, {Label: "WAL receiver", Value: "not running"}}))
		} else if s.Replication.Receiver.Status != "streaming" {
			add(newFinding("wal-receiver-state", s.ServerID, "", "", models.SeverityHigh, "Replication", "Replica WAL receiver is not streaming", fmt.Sprintf("The WAL receiver state is %s.", s.Replication.Receiver.Status), "Replication progress may be interrupted until streaming resumes.", models.ConfidenceHigh, []models.Evidence{{Label: "Receiver state", Value: s.Replication.Receiver.Status}, {Label: "Last message age", Value: fmt.Sprintf("%.0f seconds", s.Replication.Receiver.LastMessageSeconds)}}))
		}
		if s.Replication.RecoveryPaused {
			f := newFinding("recovery-paused", s.ServerID, "", "", models.SeverityMedium, "Replication", "WAL replay is paused", "PostgreSQL reports that recovery replay is paused.", "Queries on this standby will not see newer changes until replay resumes.", models.ConfidenceHigh, []models.Evidence{{Label: "Replay state", Value: "paused"}, {Label: "Replay LSN", Value: s.Replication.ReplayLSN}, {Label: "Timeline", Value: fmt.Sprint(s.Replication.TimelineID)}})
			f.Suggestions = []models.Suggestion{{Title: "Confirm whether replay was paused intentionally", Detail: "Inspect the standby recovery procedure and owning change record before asking an administrator to resume replay."}}
			add(f)
		}
	}
	for _, standby := range s.Replication.Standbys {
		resource := standby.Application
		if resource == "" {
			resource = standby.ClientAddress
		}
		if standby.State != "streaming" {
			add(newFinding("standby-state", s.ServerID, "", resource, models.SeverityHigh, "Replication", "Connected standby is not streaming", fmt.Sprintf("Standby %s reports replication state %s.", resource, standby.State), "The standby may not be receiving current WAL from this primary.", models.ConfidenceHigh, []models.Evidence{{Label: "State", Value: standby.State}, {Label: "Sync mode", Value: standby.SyncState}}))
		} else if standby.ReplayLagSeconds >= t.ReplicaLagSeconds && !delayedReplica {
			add(newFinding("standby-replay-lag", s.ServerID, "", resource, models.SeverityMedium, "Replication", "Replica replay lag is elevated", fmt.Sprintf("Standby %s reports %.1f seconds of replay lag and %.0f bytes pending replay.", resource, standby.ReplayLagSeconds, standby.PendingReplayBytes), "Reads from the replica may observe older data and recovery objectives may be at risk if lag continues.", models.ConfidenceMedium, []models.Evidence{{Label: "Replay lag", Value: fmt.Sprintf("%.1f seconds", standby.ReplayLagSeconds)}, {Label: "Pending replay", Value: fmt.Sprintf("%.0f bytes", standby.PendingReplayBytes)}, {Label: "Pending network send", Value: fmt.Sprintf("%.0f bytes", standby.PendingSendBytes)}, {Label: "Reply age", Value: fmt.Sprintf("%.1f seconds", standby.ReplyAgeSeconds)}, {Label: "Sync mode", Value: standby.SyncState}}))
		}
	}
	if (s.WAL.ArchiveMode == "on" || s.WAL.ArchiveMode == "always") && !s.WAL.ArchiveConfigured {
		add(newFinding("archive-not-configured", s.ServerID, "", "", models.SeverityHigh, "WAL", "WAL archiving is enabled but has no destination", "archive_mode is enabled while both archive_command and archive_library are empty.", "Completed WAL segments can accumulate locally and the expected archive recovery chain is not being produced.", models.ConfidenceHigh, []models.Evidence{{Label: "Archive mode", Value: s.WAL.ArchiveMode}, {Label: "Archive destination", Value: "not configured"}}))
	}
	if s.WAL.ArchiveMode != "off" && s.WAL.ArchiveMode != "" && s.WAL.LastFailedAt != nil && (s.WAL.LastArchivedAt == nil || s.WAL.LastFailedAt.After(*s.WAL.LastArchivedAt)) {
		add(newFinding("archive-failure-current", s.ServerID, "", "", models.SeverityHigh, "WAL", "Latest WAL archive attempt failed", "The newest archive status is a failure with no later successful archive recorded.", "Repeated failures can retain WAL on the database server and weaken archive-based recovery objectives.", models.ConfidenceHigh, []models.Evidence{{Label: "Failed WAL", Value: s.WAL.LastFailedWAL}, {Label: "Last failure", Value: s.WAL.LastFailedAt.UTC().Format(time.RFC3339)}, {Label: "Failed attempts", Value: fmt.Sprintf("%.0f", s.WAL.FailedArchiveCount)}, {Label: "Last archived WAL", Value: s.WAL.LastArchivedWAL}}))
	}
	for _, slot := range s.Replication.Slots {
		if slot.WALStatus == "lost" {
			add(newFinding("replication-slot-lost", s.ServerID, slot.Database, slot.Name, models.SeverityCritical, "Replication", "Replication slot has lost required WAL", fmt.Sprintf("Slot %s reports wal_status=lost; its consumer can no longer continue from the retained position.", slot.Name), "The replication consumer may require deliberate reinitialization before it can resume safely.", models.ConfidenceHigh, []models.Evidence{{Label: "Slot", Value: slot.Name}, {Label: "Slot type", Value: slot.Type}, {Label: "WAL status", Value: slot.WALStatus}, {Label: "Active", Value: fmt.Sprint(slot.Active)}}))
			continue
		}
		if !slot.Active && slot.RetainedBytes >= t.SlotRetainedBytes {
			add(newFinding("inactive-slot-wal", s.ServerID, slot.Database, slot.Name, models.SeverityHigh, "Replication", "Inactive replication slot is retaining WAL", fmt.Sprintf("Slot %s is inactive and retains approximately %.0f bytes of WAL.", slot.Name, slot.RetainedBytes), "Retained WAL can continue consuming disk until the slot advances or is deliberately removed.", models.ConfidenceHigh, []models.Evidence{{Label: "Slot", Value: slot.Name}, {Label: "Retained WAL", Value: fmt.Sprintf("%.0f bytes", slot.RetainedBytes)}, {Label: "Retention growth", Value: fmt.Sprintf("%.1f bytes/second", slot.RetainedGrowthBytesPerSecond)}, {Label: "Inactive age", Value: fmt.Sprintf("%.0f seconds", slot.InactiveSeconds)}, {Label: "WAL status", Value: slot.WALStatus}}))
		}
	}
	restartpointTotal := s.WAL.RestartpointsTimed + s.WAL.RestartpointsRequested
	if s.Replication.InRecovery && restartpointTotal >= 10 && s.WAL.RestartpointsRequested/restartpointTotal*100 >= t.RequestedCheckpointRatio {
		add(newFinding("requested-restartpoints", s.ServerID, "", "", models.SeverityMedium, "WAL", "Requested restartpoints are frequent", fmt.Sprintf("%.1f%% of %.0f observed restartpoint attempts were requested; %.0f completed.", s.WAL.RestartpointsRequested/restartpointTotal*100, restartpointTotal, s.WAL.RestartpointsDone), "Frequent requested restartpoints can indicate WAL pressure during recovery and contribute to storage write bursts.", models.ConfidenceMedium, []models.Evidence{{Label: "Requested restartpoints", Value: fmt.Sprintf("%.0f", s.WAL.RestartpointsRequested)}, {Label: "Timed restartpoints", Value: fmt.Sprintf("%.0f", s.WAL.RestartpointsTimed)}, {Label: "Completed restartpoints", Value: fmt.Sprintf("%.0f", s.WAL.RestartpointsDone)}}))
	}
	checkpointTotal := s.WAL.TimedCheckpoints + s.WAL.RequestedCheckpoints
	if !s.Replication.InRecovery && checkpointTotal >= 10 {
		requestedRatio := s.WAL.RequestedCheckpoints / checkpointTotal * 100
		if requestedRatio >= t.RequestedCheckpointRatio {
			add(newFinding("requested-checkpoints", s.ServerID, "", "", models.SeverityMedium, "WAL", "Requested checkpoints are frequent", fmt.Sprintf("%.1f%% of %.0f checkpoints were requested rather than timed.", requestedRatio, checkpointTotal), "Frequent requested checkpoints can increase write bursts and indicate WAL pressure or undersized max_wal_size.", models.ConfidenceMedium, []models.Evidence{{Label: "Requested", Value: fmt.Sprintf("%.0f", s.WAL.RequestedCheckpoints)}, {Label: "Timed", Value: fmt.Sprintf("%.0f", s.WAL.TimedCheckpoints)}, {Label: "Requested ratio", Value: fmt.Sprintf("%.1f%%", requestedRatio)}}))
		}
		if s.WAL.StatsReset != nil {
			interval := time.Since(*s.WAL.StatsReset).Seconds() / checkpointTotal
			if interval > 0 && interval < t.CheckpointIntervalSeconds {
				add(newFinding("checkpoint-frequency", s.ServerID, "", "", models.SeverityMedium, "WAL", "Checkpoints are occurring frequently", fmt.Sprintf("The average interval since the statistics reset is %.0f seconds.", interval), "Frequent checkpoints can create avoidable write pressure and latency variability.", models.ConfidenceMedium, []models.Evidence{{Label: "Average interval", Value: fmt.Sprintf("%.0f seconds", interval)}, {Label: "Write time", Value: fmt.Sprintf("%.0f ms", s.WAL.WriteTimeMS)}, {Label: "Sync time", Value: fmt.Sprintf("%.0f ms", s.WAL.SyncTimeMS)}}))
			}
		}
	}
	if s.Connections.Utilization >= t.ConnectionHigh {
		sev := models.SeverityHigh
		if s.Connections.Utilization >= t.ConnectionCritical {
			sev = models.SeverityCritical
		}
		add(newFinding("connection-utilization", s.ServerID, "", "", sev, "Connections", "Connection capacity is running low", fmt.Sprintf("%.1f%% of max_connections is currently used.", s.Connections.Utilization), "New application connections may be rejected and existing requests can queue.", models.ConfidenceHigh, []models.Evidence{{Label: "Utilization", Value: fmt.Sprintf("%.1f%%", s.Connections.Utilization)}, {Label: "Connections", Value: fmt.Sprintf("%d / %d", s.Connections.Total, s.Connections.Max)}}))
	}
	if s.Connections.IdleInTransaction > 0 && s.Connections.LongestIdleTransactionSeconds >= t.IdleTransactionSeconds {
		add(newFinding("idle-in-transaction", s.ServerID, "", "", models.SeverityHigh, "Connections", "Sessions are idle in transaction", fmt.Sprintf("%d sessions are idle in transaction; the oldest is %.0f seconds.", s.Connections.IdleInTransaction, s.Connections.LongestIdleTransactionSeconds), "Open transactions may prevent vacuum cleanup, retain locks and increase table bloat.", models.ConfidenceHigh, []models.Evidence{{Label: "Sessions", Value: fmt.Sprint(s.Connections.IdleInTransaction)}, {Label: "Longest", Value: fmt.Sprintf("%.0f seconds", s.Connections.LongestIdleTransactionSeconds)}}))
	}
	if s.Connections.LongestTransactionSeconds >= t.LongTransactionSeconds {
		add(newFinding("long-transaction", s.ServerID, "", "", models.SeverityHigh, "Transactions", "A transaction has been open unusually long", fmt.Sprintf("The longest transaction is %.0f seconds old.", s.Connections.LongestTransactionSeconds), "Long snapshots can delay vacuum cleanup and contribute to lock contention.", models.ConfidenceHigh, []models.Evidence{{Label: "Transaction age", Value: fmt.Sprintf("%.0f seconds", s.Connections.LongestTransactionSeconds)}}))
	}
	var longBlocked []models.LockInfo
	for _, l := range s.Locks {
		if l.DurationSeconds >= t.LongQuerySeconds {
			longBlocked = append(longBlocked, l)
		}
	}
	if len(longBlocked) > 0 {
		max := 0.0
		for _, l := range longBlocked {
			if l.DurationSeconds > max {
				max = l.DurationSeconds
			}
		}
		add(newFinding("blocking-queries", s.ServerID, "", "", models.SeverityHigh, "Locks", "Queries are blocked by other sessions", fmt.Sprintf("%d blocked sessions detected; longest wait %.0f seconds.", len(longBlocked), max), "Blocked work can increase application latency and cause cascading connection pressure.", models.ConfidenceHigh, []models.Evidence{{Label: "Affected sessions", Value: fmt.Sprint(len(longBlocked))}, {Label: "Longest wait", Value: fmt.Sprintf("%.0f seconds", max)}}))
	}
	for _, d := range s.Databases {
		if d.Deadlocks > 0 {
			add(newFinding("deadlocks", s.ServerID, d.Name, "", models.SeverityHigh, "Locks", "Deadlocks were detected", fmt.Sprintf("PostgreSQL recorded %.0f deadlocks since statistics reset.", d.Deadlocks), "Deadlocks abort transactions and can surface as application errors. Compare counter deltas to determine whether this is current.", models.ConfidenceMedium, []models.Evidence{{Label: "Deadlocks", Value: fmt.Sprintf("%.0f", d.Deadlocks)}}))
		}
		total := d.Commits + d.Rollbacks
		if total > 100 && d.Rollbacks/total*100 >= t.RollbackRatio {
			add(newFinding("rollback-ratio", s.ServerID, d.Name, "", models.SeverityMedium, "Transactions", "Rollback ratio is elevated", fmt.Sprintf("%.1f%% of transactions rolled back.", d.Rollbacks/total*100), "Frequent rollbacks waste database work and may indicate application errors or contention.", models.ConfidenceMedium, nil))
		}
		hit := d.BlocksHit + d.BlocksRead
		if hit > 0 && d.BlocksHit/hit*100 < t.CacheHitLow {
			add(newFinding("cache-hit", s.ServerID, d.Name, "", models.SeverityMedium, "Performance", "Cache hit ratio is degraded", fmt.Sprintf("Observed database buffer cache hit ratio is %.1f%%.", d.BlocksHit/hit*100), "Additional physical reads may increase query latency. Review trends and available memory before tuning.", models.ConfidenceMedium, nil))
		}
	}
	for _, table := range s.Tables {
		ratio := 0.0
		if table.LiveTuples+table.DeadTuples > 0 {
			ratio = table.DeadTuples / (table.LiveTuples + table.DeadTuples) * 100
		}
		resource := table.Schema + "." + table.Table
		if ratio >= t.DeadTupleRatio {
			add(newFinding("dead-tuples", s.ServerID, table.Database, resource, models.SeverityHigh, "Vacuum", "Dead tuple ratio is high", fmt.Sprintf("%s has %.0f dead tuples (%.1f%%).", resource, table.DeadTuples, ratio), "Table and index bloat may increase disk IO and query latency.", models.ConfidenceHigh, []models.Evidence{{Label: "Dead tuples", Value: fmt.Sprintf("%.0f", table.DeadTuples)}, {Label: "Dead tuple ratio", Value: fmt.Sprintf("%.1f%%", ratio)}}))
		}
		if table.VacuumProgress >= t.VacuumProgress && table.DeadTuples > 10000 {
			add(newFinding("vacuum-behind", s.ServerID, table.Database, resource, models.SeverityHigh, "Vacuum", "Autovacuum may not be keeping up", fmt.Sprintf("Dead tuples reached %.0f%% of the estimated autovacuum trigger.", table.VacuumProgress), "Continued churn may cause bloat and additional IO.", models.ConfidenceMedium, []models.Evidence{{Label: "Dead tuples", Value: fmt.Sprintf("%.0f", table.DeadTuples)}, {Label: "Estimated trigger", Value: fmt.Sprintf("%.0f", table.VacuumThreshold)}}))
		}
		if table.EstimatedRows > 1e6 && table.SeqScans > 1000 && table.IndexScans < table.SeqScans/10 {
			add(newFinding("large-seq-scans", s.ServerID, table.Database, resource, models.SeverityLow, "Queries", "Large table receives many sequential scans", fmt.Sprintf("%.0f sequential scans were observed on approximately %.0f rows.", table.SeqScans, table.EstimatedRows), "This is a potential improvement only; sequential scans can be optimal. Correlate with frequent costly query filters and existing indexes.", models.ConfidenceLow, nil))
		}
		if table.EstimatedRows > 100000 && stale(table.LastAutoanalyze, 7*24*time.Hour) && stale(table.LastAnalyze, 7*24*time.Hour) {
			add(newFinding("stale-analyze", s.ServerID, table.Database, resource, models.SeverityMedium, "Queries", "Table statistics may be stale", fmt.Sprintf("%s has not been analyzed in the last 7 days.", resource), "Stale planner statistics can lead to poor row estimates and inefficient plans.", models.ConfidenceMedium, nil))
		}
	}
	for _, q := range s.Queries {
		if q.ImpactScore >= t.QueryImpactHigh {
			add(newFinding("query-impact", s.ServerID, q.Database, q.QueryID, models.SeverityMedium, "Queries", "Query has unusually high total impact", fmt.Sprintf("Query %s has an impact score of %.1f from total runtime, latency, reads, temp I/O and WAL.", q.QueryID, q.ImpactScore), "A high cumulative workload can consume substantial database capacity even when individual executions appear fast.", models.ConfidenceMedium, []models.Evidence{{Label: "Calls", Value: fmt.Sprintf("%.0f", q.Calls)}, {Label: "Total runtime", Value: fmt.Sprintf("%.0f ms", q.TotalExecMS)}, {Label: "Mean latency", Value: fmt.Sprintf("%.2f ms", q.MeanExecMS)}}))
		}
	}
	if !s.Capabilities["pg_stat_statements"] {
		add(newFinding("pgss-unavailable", s.ServerID, "", "", models.SeverityInfo, "Configuration", "Query monitoring is unavailable", "pg_stat_statements is not enabled on this PostgreSQL server.", "Query-level load, latency and regression analysis cannot be performed.", models.ConfidenceHigh, nil))
	}
	if s.Settings["track_io_timing"] == "off" {
		add(newFinding("io-timing-disabled", s.ServerID, "", "", models.SeverityInfo, "Configuration", "I/O timing is disabled", "track_io_timing is off, limiting diagnosis of storage latency.", "Read and write time attribution will be unavailable. Enabling it has overhead that should be evaluated.", models.ConfidenceHigh, nil))
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
func stale(t *time.Time, d time.Duration) bool { return t == nil || time.Since(*t) > d }
