package analyzer

import (
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeDatabases(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	for _, database := range s.Databases {
		if database.Deadlocks > 0 {
			out = append(out, newFinding("deadlocks", s.ServerID, database.Name, "", models.SeverityHigh, "Locks", "Deadlocks were detected", fmt.Sprintf("PostgreSQL recorded %.0f deadlocks since statistics reset.", database.Deadlocks), "Deadlocks abort transactions and can surface as application errors. Compare counter deltas to determine whether this is current.", models.ConfidenceMedium, []models.Evidence{{Label: "Deadlocks", Value: fmt.Sprintf("%.0f", database.Deadlocks)}}))
		}
		total := database.Commits + database.Rollbacks
		if total > 100 && database.Rollbacks/total*100 >= e.Thresholds.RollbackRatio {
			out = append(out, newFinding("rollback-ratio", s.ServerID, database.Name, "", models.SeverityMedium, "Transactions", "Rollback ratio is elevated", fmt.Sprintf("%.1f%% of transactions rolled back.", database.Rollbacks/total*100), "Frequent rollbacks waste database work and may indicate application errors or contention.", models.ConfidenceMedium, nil))
		}
		hit := database.BlocksHit + database.BlocksRead
		if hit > 0 && database.BlocksHit/hit*100 < e.Thresholds.CacheHitLow {
			out = append(out, newFinding("cache-hit", s.ServerID, database.Name, "", models.SeverityMedium, "Performance", "Cache hit ratio is degraded", fmt.Sprintf("Observed database buffer cache hit ratio is %.1f%%.", database.BlocksHit/hit*100), "Additional physical reads may increase query latency. Review trends and available memory before tuning.", models.ConfidenceMedium, nil))
		}
	}
	return out
}
