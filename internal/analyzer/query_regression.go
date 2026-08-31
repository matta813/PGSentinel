package analyzer

import (
	"fmt"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

type queryKey struct{ database, id string }
type queryInterval struct {
	start, end                              time.Time
	calls, runtime, mean, rows, sharedReads float64
}

func queryMap(sample []models.QueryStat) map[queryKey]models.QueryStat {
	out := make(map[queryKey]models.QueryStat, len(sample))
	for _, query := range sample {
		out[queryKey{query.Database, query.QueryID}] = query
	}
	return out
}

// QueryRegressionFindings compares reset-aware interval deltas. A finding needs
// six compatible baseline intervals and two consecutive anomalous intervals.
func QueryRegressionFindings(serverID string, history []models.QueryObservation, current models.QueryObservation, changeHistory ...[]models.ChangeEvent) []models.Finding {
	if len(history) < 8 {
		return nil
	}
	observations := append(append([]models.QueryObservation(nil), history...), current)
	maps := make([]map[queryKey]models.QueryStat, len(observations))
	for i := range observations {
		maps[i] = queryMap(observations[i].Queries)
	}
	var out []models.Finding
	for key := range maps[len(maps)-1] {
		intervals := compatibleQueryIntervals(key, observations, maps)
		if len(intervals) < 8 {
			continue
		}
		previous, latest := intervals[len(intervals)-2], intervals[len(intervals)-1]
		baselineIntervals := intervals[:len(intervals)-2]
		baseline := make([]float64, 0, len(baselineIntervals))
		for _, interval := range baselineIntervals {
			baseline = append(baseline, interval.mean)
		}
		previousComparison := CompareBaseline(baseline, previous.mean, 50)
		comparison := CompareBaseline(baseline, latest.mean, 50)
		absolute := latest.mean - comparison.Median
		if !previousComparison.Anomalous || !comparison.Anomalous || comparison.Median < 5 || absolute < 5 {
			continue
		}
		confidence := models.ConfidenceMedium
		if len(baseline) >= 7 && latest.calls >= 50 && previous.calls >= 50 {
			confidence = models.ConfidenceHigh
		}
		finding := newFinding("query-regression", serverID, key.database, key.id, models.SeverityHigh, "Queries", "Persistent query latency regression detected",
			fmt.Sprintf("Query %s exceeded its reset-aware baseline in two consecutive observation intervals with sufficient calls and runtime.", key.id),
			"A persistent increase in execution latency can increase application response time and database load. The overlap is evidence of a meaningful change, not proof of a specific cause.", confidence,
			[]models.Evidence{
				{Label: "Current window", Value: formatWindow(latest.start, latest.end)},
				{Label: "Baseline window", Value: formatWindow(baselineIntervals[0].start, baselineIntervals[len(baselineIntervals)-1].end)},
				{Label: "Baseline samples", Value: fmt.Sprintf("%d intervals", len(baseline))},
				{Label: "Previous interval mean", Value: fmt.Sprintf("%.2f ms", previous.mean)},
				{Label: "Current interval mean", Value: fmt.Sprintf("%.2f ms", latest.mean)},
				{Label: "Baseline median", Value: fmt.Sprintf("%.2f ms", comparison.Median)},
				{Label: "Absolute difference", Value: fmt.Sprintf("+%.2f ms", absolute)},
				{Label: "Relative difference", Value: fmt.Sprintf("+%.1f%%", comparison.ChangePercent)},
				{Label: "Current calls", Value: fmt.Sprintf("%.0f", latest.calls)},
				{Label: "Current total runtime", Value: fmt.Sprintf("%.0f ms", latest.runtime)},
				{Label: "Current rows", Value: fmt.Sprintf("%.0f", latest.rows)},
				{Label: "Current shared reads", Value: fmt.Sprintf("%.0f blocks", latest.sharedReads)},
				{Label: "Significance", Value: "Above median + 3 MAD, at least 50% and 5 ms slower, persistent for 2 intervals"},
			})
		finding.Suggestions = []models.Suggestion{
			{Title: "Correlate the regression window", Detail: "Compare deployments, parameter changes, data growth, cache pressure, call volume and row volume during the displayed windows."},
			{Title: "Inspect a safe plan separately", Detail: "Use guarded read-only EXPLAIN only after reviewing the captured statement and parameters. PGSentinel never runs EXPLAIN ANALYZE automatically."},
		}
		if len(changeHistory) > 0 {
			matched := 0
			for _, event := range changeHistory[0] {
				if event.OccurredAt.Before(previous.start) || event.OccurredAt.After(latest.end) {
					continue
				}
				finding.Evidence = append(finding.Evidence, models.Evidence{Label: "Correlated change", Value: fmt.Sprintf("%s at %s: %s", event.Kind, event.OccurredAt.UTC().Format(time.RFC3339), event.Summary)})
				matched++
				if matched == 5 {
					break
				}
			}
			if matched > 0 {
				finding.Confidence = models.ConfidenceMedium
			}
		}
		out = append(out, finding)
	}
	return out
}

// QueryRegressionFindingReady reports whether resolving one existing regression is safe.
// Resets, restarts, missing query IDs and low-sample gaps require a full compatible window.
func QueryRegressionFindingReady(finding models.Finding, history []models.QueryObservation, current models.QueryObservation) bool {
	observations := append(append([]models.QueryObservation(nil), history...), current)
	maps := make([]map[queryKey]models.QueryStat, len(observations))
	for i := range observations {
		maps[i] = queryMap(observations[i].Queries)
	}
	return len(compatibleQueryIntervals(queryKey{database: finding.Database, id: finding.Resource}, observations, maps)) >= 8
}

func compatibleQueryIntervals(key queryKey, observations []models.QueryObservation, maps []map[queryKey]models.QueryStat) []queryInterval {
	intervals := make([]queryInterval, 0, len(observations)-1)
	for i := 1; i < len(observations); i++ {
		before, beforeOK := maps[i-1][key]
		after, afterOK := maps[i][key]
		if !beforeOK || !afterOK || !sameBoundary(observations[i-1].StatsResetAt, observations[i].StatsResetAt) || !sameBoundary(observations[i-1].PostmasterStartAt, observations[i].PostmasterStartAt) {
			intervals = intervals[:0]
			continue
		}
		calls, runtime := after.Calls-before.Calls, after.TotalExecMS-before.TotalExecMS
		rows, reads := after.Rows-before.Rows, after.SharedRead-before.SharedRead
		if calls < 20 || runtime < 1000 || rows < 0 || reads < 0 {
			intervals = intervals[:0]
			continue
		}
		intervals = append(intervals, queryInterval{start: observations[i-1].CollectedAt, end: observations[i].CollectedAt, calls: calls, runtime: runtime, mean: runtime / calls, rows: rows, sharedReads: reads})
	}
	return intervals
}

func sameBoundary(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func formatWindow(start, end time.Time) string {
	return start.UTC().Format(time.RFC3339) + " to " + end.UTC().Format(time.RFC3339)
}
