package analyzer

import (
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

func (e *Engine) analyzeQueries(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	for _, query := range s.Queries {
		if query.ImpactScore >= e.Thresholds.QueryImpactHigh {
			out = append(out, newFinding("query-impact", s.ServerID, query.Database, query.QueryID, models.SeverityMedium, "Queries", "Query has unusually high total impact", fmt.Sprintf("Query %s has an impact score of %.1f from total runtime, latency, reads, temp I/O and WAL.", query.QueryID, query.ImpactScore), "A high cumulative workload can consume substantial database capacity even when individual executions appear fast.", models.ConfidenceMedium, []models.Evidence{{Label: "Calls", Value: fmt.Sprintf("%.0f", query.Calls)}, {Label: "Total runtime", Value: fmt.Sprintf("%.0f ms", query.TotalExecMS)}, {Label: "Mean latency", Value: fmt.Sprintf("%.2f ms", query.MeanExecMS)}}))
		}
	}
	return out
}
