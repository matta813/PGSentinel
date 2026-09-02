package analyzer

import (
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeLocks(s models.Snapshot) []models.Finding {
	longBlocked := []models.LockInfo{}
	for _, lock := range s.Locks {
		if lock.DurationSeconds >= e.Thresholds.LongQuerySeconds {
			longBlocked = append(longBlocked, lock)
		}
	}
	if len(longBlocked) == 0 {
		return nil
	}
	max := 0.0
	for _, lock := range longBlocked {
		if lock.DurationSeconds > max {
			max = lock.DurationSeconds
		}
	}
	return []models.Finding{newFinding("blocking-queries", s.ServerID, "", "", models.SeverityHigh, "Locks", "Queries are blocked by other sessions", fmt.Sprintf("%d blocked sessions detected; longest wait %.0f seconds.", len(longBlocked), max), "Blocked work can increase application latency and cause cascading connection pressure.", models.ConfidenceHigh, []models.Evidence{{Label: "Affected sessions", Value: fmt.Sprint(len(longBlocked))}, {Label: "Longest wait", Value: fmt.Sprintf("%.0f seconds", max)}})}
}
