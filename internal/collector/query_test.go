package collector

import (
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
)

func TestImpactScoreRewardsTotalWork(t *testing.T) {
	frequent := ImpactScore(models.QueryStat{Calls: 2e7, MeanExecMS: 1, TotalExecMS: 2e7})
	rare := ImpactScore(models.QueryStat{Calls: 3, MeanExecMS: 4000, TotalExecMS: 12000})
	if frequent <= rare {
		t.Fatalf("frequent=%f rare=%f", frequent, rare)
	}
}
