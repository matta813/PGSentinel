package analyzer

import (
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeConnections(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	t := e.Thresholds
	if s.Connections.Utilization >= t.ConnectionHigh {
		sev := models.SeverityHigh
		if s.Connections.Utilization >= t.ConnectionCritical {
			sev = models.SeverityCritical
		}
		out = append(out, newFinding("connection-utilization", s.ServerID, "", "", sev, "Connections", "Connection capacity is running low", fmt.Sprintf("%.1f%% of max_connections is currently used.", s.Connections.Utilization), "New application connections may be rejected and existing requests can queue.", models.ConfidenceHigh, []models.Evidence{{Label: "Utilization", Value: fmt.Sprintf("%.1f%%", s.Connections.Utilization)}, {Label: "Connections", Value: fmt.Sprintf("%d / %d", s.Connections.Total, s.Connections.Max)}}))
	}
	if s.Connections.IdleInTransaction > 0 && s.Connections.LongestIdleTransactionSeconds >= t.IdleTransactionSeconds {
		out = append(out, newFinding("idle-in-transaction", s.ServerID, "", "", models.SeverityHigh, "Connections", "Sessions are idle in transaction", fmt.Sprintf("%d sessions are idle in transaction; the oldest is %.0f seconds.", s.Connections.IdleInTransaction, s.Connections.LongestIdleTransactionSeconds), "Open transactions may prevent vacuum cleanup, retain locks and increase table bloat.", models.ConfidenceHigh, []models.Evidence{{Label: "Sessions", Value: fmt.Sprint(s.Connections.IdleInTransaction)}, {Label: "Longest", Value: fmt.Sprintf("%.0f seconds", s.Connections.LongestIdleTransactionSeconds)}}))
	}
	if s.Connections.LongestTransactionSeconds >= t.LongTransactionSeconds {
		out = append(out, newFinding("long-transaction", s.ServerID, "", "", models.SeverityHigh, "Transactions", "A transaction has been open unusually long", fmt.Sprintf("The longest transaction is %.0f seconds old.", s.Connections.LongestTransactionSeconds), "Long snapshots can delay vacuum cleanup and contribute to lock contention.", models.ConfidenceHigh, []models.Evidence{{Label: "Transaction age", Value: fmt.Sprintf("%.0f seconds", s.Connections.LongestTransactionSeconds)}}))
	}
	return out
}
