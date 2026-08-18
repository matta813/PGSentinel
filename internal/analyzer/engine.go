package analyzer

import (
	"fmt"
	"github.com/matta813/pgsentinel/internal/models"
	"time"
)

type Thresholds struct{ ConnectionHigh, ConnectionCritical, IdleTransactionSeconds, LongTransactionSeconds, LongQuerySeconds, DeadTupleRatio, VacuumProgress, CacheHitLow, RollbackRatio, QueryImpactHigh float64 }

func DefaultThresholds() Thresholds { return Thresholds{80, 95, 300, 900, 60, 20, 100, 95, 5, 250} }

type Engine struct{ Thresholds Thresholds }

func New(t Thresholds) *Engine { return &Engine{Thresholds: t} }
func (e *Engine) Analyze(s models.Snapshot) []models.Finding {
	t := e.Thresholds
	out := []models.Finding{}
	add := func(f models.Finding) { out = append(out, f) }
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
func stale(t *time.Time, d time.Duration) bool { return t == nil || time.Since(*t) > d }
