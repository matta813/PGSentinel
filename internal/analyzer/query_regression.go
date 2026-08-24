package analyzer

import (
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

type queryKey struct{ database, id string }

func queryMap(sample []models.QueryStat) map[queryKey]models.QueryStat {
	out := make(map[queryKey]models.QueryStat, len(sample))
	for _, query := range sample {
		out[queryKey{query.Database, query.QueryID}] = query
	}
	return out
}

// QueryRegressionFindings compares interval deltas, not cumulative pg_stat_statements means.
// Counter resets or evictions make an interval incompatible and exclude it from the baseline.
func QueryRegressionFindings(serverID string, history [][]models.QueryStat, current []models.QueryStat) []models.Finding {
	if len(history) < 7 {
		return nil
	}
	samples := append(append([][]models.QueryStat(nil), history...), current)
	maps := make([]map[queryKey]models.QueryStat, len(samples))
	for i, sample := range samples {
		maps[i] = queryMap(sample)
	}
	var out []models.Finding
	for key, now := range maps[len(maps)-1] {
		previous, ok := maps[len(maps)-2][key]
		if !ok {
			continue
		}
		calls := now.Calls - previous.Calls
		runtime := now.TotalExecMS - previous.TotalExecMS
		if calls < 20 || runtime < 1000 || calls <= 0 || runtime <= 0 {
			continue
		}
		currentMean := runtime / calls
		var baseline []float64
		for i := 1; i < len(maps)-1; i++ {
			before, beforeOK := maps[i-1][key]
			after, afterOK := maps[i][key]
			if !beforeOK || !afterOK {
				continue
			}
			deltaCalls, deltaRuntime := after.Calls-before.Calls, after.TotalExecMS-before.TotalExecMS
			if deltaCalls >= 20 && deltaRuntime > 0 {
				baseline = append(baseline, deltaRuntime/deltaCalls)
			}
		}
		comparison := CompareBaseline(baseline, currentMean, 75)
		if !comparison.Anomalous || comparison.Median < 5 {
			continue
		}
		finding := newFinding("query-regression", serverID, key.database, key.id, models.SeverityHigh, "Queries", "Query latency regressed against its baseline", fmt.Sprintf("Query %s averaged %.2f ms in the latest interval versus a %.2f ms median baseline.", key.id, currentMean, comparison.Median), "A sustained regression in an important query can increase application latency and database load.", models.ConfidenceHigh, []models.Evidence{{Label: "Current interval mean", Value: fmt.Sprintf("%.2f ms", currentMean)}, {Label: "Baseline median", Value: fmt.Sprintf("%.2f ms", comparison.Median)}, {Label: "Change", Value: fmt.Sprintf("%.1f%%", comparison.ChangePercent)}, {Label: "Interval calls", Value: fmt.Sprintf("%.0f", calls)}})
		finding.Suggestions = []models.Suggestion{{Title: "Compare recent deployments, parameter changes, data growth, and execution plans without running EXPLAIN ANALYZE automatically."}}
		out = append(out, finding)
	}
	return out
}
