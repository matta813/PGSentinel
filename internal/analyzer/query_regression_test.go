package analyzer

import (
	"strings"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func regressionSeries(means, calls []float64) ([]models.QueryObservation, models.QueryObservation) {
	reset := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	start := reset.Add(time.Hour)
	observations := []models.QueryObservation{{CollectedAt: start, StatsResetAt: &reset, PostmasterStartAt: &reset, Queries: []models.QueryStat{{QueryID: "42", Database: "app"}}}}
	var totalCalls, totalRuntime, totalRows, totalReads float64
	for i, mean := range means {
		totalCalls += calls[i]
		totalRuntime += calls[i] * mean
		totalRows += calls[i] * 3
		totalReads += calls[i] / 2
		observations = append(observations, models.QueryObservation{CollectedAt: start.Add(time.Duration(i+1) * time.Minute), StatsResetAt: &reset, PostmasterStartAt: &reset, Queries: []models.QueryStat{{QueryID: "42", Database: "app", Calls: totalCalls, TotalExecMS: totalRuntime, Rows: totalRows, SharedRead: totalReads}}})
	}
	return observations[:len(observations)-1], observations[len(observations)-1]
}

func TestQueryRegressionRequiresPersistentResetAwareSignal(t *testing.T) {
	history, current := regressionSeries([]float64{10, 10, 10, 10, 10, 10, 10, 30, 32}, []float64{100, 120, 100, 110, 130, 100, 100, 80, 85})
	got := QueryRegressionFindings("server", history, current)
	if len(got) != 1 || got[0].Confidence != models.ConfidenceHigh {
		t.Fatalf("findings=%#v", got)
	}
	evidence := ""
	for _, item := range got[0].Evidence {
		evidence += item.Label + ":" + item.Value + "\n"
	}
	for _, wanted := range []string{"Current window", "Baseline window", "Baseline samples:7 intervals", "Previous interval mean:30.00 ms", "Absolute difference:+22.00 ms", "Current total runtime:2720 ms", "Current rows:255"} {
		if !strings.Contains(evidence, wanted) {
			t.Fatalf("missing %q in %s", wanted, evidence)
		}
	}
}

func TestQueryRegressionRejectsResetRestartAndSingleSpike(t *testing.T) {
	means := []float64{10, 10, 10, 10, 10, 10, 10, 10, 35}
	calls := []float64{100, 100, 100, 100, 100, 100, 100, 100, 100}
	history, current := regressionSeries(means, calls)
	if got := QueryRegressionFindings("server", history, current); len(got) != 0 {
		t.Fatalf("single spike=%#v", got)
	}
	history, current = regressionSeries([]float64{10, 10, 10, 10, 10, 10, 10, 30, 32}, calls)
	changed := current.CollectedAt
	current.StatsResetAt = &changed
	if got := QueryRegressionFindings("server", history, current); len(got) != 0 {
		t.Fatalf("reset boundary=%#v", got)
	}
	finding := models.Finding{Database: "app", Resource: "42"}
	if QueryRegressionFindingReady(finding, history, current) {
		t.Fatal("reset boundary was considered safe for resolution")
	}
	history, current = regressionSeries([]float64{10, 10, 10, 10, 10, 10, 10, 30, 32}, calls)
	current.PostmasterStartAt = &changed
	if got := QueryRegressionFindings("server", history, current); len(got) != 0 {
		t.Fatalf("restart boundary=%#v", got)
	}
}

func TestQueryRegressionWindowBecomesReadyAfterContiguousWarmup(t *testing.T) {
	history, current := regressionSeries([]float64{10, 10, 10, 10, 10, 10, 10, 10}, []float64{100, 100, 100, 100, 100, 100, 100, 100})
	finding := models.Finding{Database: "app", Resource: "42"}
	if !QueryRegressionFindingReady(finding, history, current) {
		t.Fatal("contiguous window should permit recovery")
	}
	if QueryRegressionFindingReady(finding, history[:7], history[7]) {
		t.Fatal("short window should not permit recovery")
	}
	lowHistory, lowCurrent := regressionSeries([]float64{10, 10, 10, 10, 10, 10, 10, 10}, []float64{100, 100, 100, 100, 100, 100, 100, 10})
	if QueryRegressionFindingReady(finding, lowHistory, lowCurrent) {
		t.Fatal("low-sample interval should not resolve a finding")
	}
}

func TestQueryRegressionRejectsLowSamplesAndRecognizesRecovery(t *testing.T) {
	history, current := regressionSeries([]float64{100, 100, 100, 100, 100, 100, 100, 300, 320}, []float64{10, 10, 10, 10, 10, 10, 10, 10, 10})
	if got := QueryRegressionFindings("server", history, current); len(got) != 0 {
		t.Fatalf("low samples=%#v", got)
	}
	history, current = regressionSeries([]float64{10, 10, 10, 10, 10, 10, 10, 30, 10}, []float64{100, 100, 100, 100, 100, 100, 100, 100, 100})
	if got := QueryRegressionFindings("server", history, current); len(got) != 0 {
		t.Fatalf("recovery=%#v", got)
	}
}
