package analyzer

import (
	"fmt"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeTables(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	for _, table := range s.Tables {
		ratio := 0.0
		if table.LiveTuples+table.DeadTuples > 0 {
			ratio = table.DeadTuples / (table.LiveTuples + table.DeadTuples) * 100
		}
		resource := table.Schema + "." + table.Table
		if ratio >= e.Thresholds.DeadTupleRatio {
			out = append(out, newFinding("dead-tuples", s.ServerID, table.Database, resource, models.SeverityHigh, "Vacuum", "Dead tuple ratio is high", fmt.Sprintf("%s has %.0f dead tuples (%.1f%%).", resource, table.DeadTuples, ratio), "Table and index bloat may increase disk IO and query latency.", models.ConfidenceHigh, []models.Evidence{{Label: "Dead tuples", Value: fmt.Sprintf("%.0f", table.DeadTuples)}, {Label: "Dead tuple ratio", Value: fmt.Sprintf("%.1f%%", ratio)}}))
		}
		if table.VacuumProgress >= e.Thresholds.VacuumProgress && table.DeadTuples > 10000 {
			out = append(out, newFinding("vacuum-behind", s.ServerID, table.Database, resource, models.SeverityHigh, "Vacuum", "Autovacuum may not be keeping up", fmt.Sprintf("Dead tuples reached %.0f%% of the estimated autovacuum trigger.", table.VacuumProgress), "Continued churn may cause bloat and additional IO.", models.ConfidenceMedium, []models.Evidence{{Label: "Dead tuples", Value: fmt.Sprintf("%.0f", table.DeadTuples)}, {Label: "Estimated trigger", Value: fmt.Sprintf("%.0f", table.VacuumThreshold)}}))
		}
		if table.EstimatedRows > 1e6 && table.SeqScans > 1000 && table.IndexScans < table.SeqScans/10 {
			out = append(out, newFinding("large-seq-scans", s.ServerID, table.Database, resource, models.SeverityLow, "Queries", "Large table receives many sequential scans", fmt.Sprintf("%.0f sequential scans were observed on approximately %.0f rows.", table.SeqScans, table.EstimatedRows), "This is a potential improvement only; sequential scans can be optimal. Correlate with frequent costly query filters and existing indexes.", models.ConfidenceLow, nil))
		}
		if table.EstimatedRows > 100000 && stale(table.LastAutoanalyze, 7*24*time.Hour) && stale(table.LastAnalyze, 7*24*time.Hour) {
			out = append(out, newFinding("stale-analyze", s.ServerID, table.Database, resource, models.SeverityMedium, "Queries", "Table statistics may be stale", fmt.Sprintf("%s has not been analyzed in the last 7 days.", resource), "Stale planner statistics can lead to poor row estimates and inefficient plans.", models.ConfidenceMedium, nil))
		}
	}
	return out
}

func stale(t *time.Time, duration time.Duration) bool { return t == nil || time.Since(*t) > duration }
