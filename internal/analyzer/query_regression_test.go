package analyzer

import (
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestQueryRegressionUsesCompatibleIntervalDeltas(t *testing.T) {
	var history [][]models.QueryStat
	for i := 1; i <= 7; i++ {
		history = append(history, []models.QueryStat{{QueryID: "42", Database: "app", Calls: float64(i * 100), TotalExecMS: float64(i * 1000)}})
	}
	current := []models.QueryStat{{QueryID: "42", Database: "app", Calls: 800, TotalExecMS: 12000}}
	got := QueryRegressionFindings("server", history, current)
	if len(got) != 1 || got[0].RuleID != "query-regression" || got[0].Confidence != models.ConfidenceHigh {
		t.Fatalf("unexpected regression findings: %#v", got)
	}
}

func TestQueryRegressionRejectsResetsAndTinyIntervals(t *testing.T) {
	history := make([][]models.QueryStat, 7)
	for i := range history {
		history[i] = []models.QueryStat{{QueryID: "42", Database: "app", Calls: float64((i + 1) * 100), TotalExecMS: float64((i + 1) * 1000)}}
	}
	for _, current := range [][]models.QueryStat{
		{{QueryID: "42", Database: "app", Calls: 5, TotalExecMS: 50}},
		{{QueryID: "42", Database: "app", Calls: 705, TotalExecMS: 7500}},
	} {
		if got := QueryRegressionFindings("server", history, current); len(got) != 0 {
			t.Fatalf("noisy/reset interval produced %#v", got)
		}
	}
}
