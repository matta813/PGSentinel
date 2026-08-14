package analyzer

import "testing"

func TestBaselineDetectsRegression(t *testing.T) {
	b := CompareBaseline([]float64{80, 90, 95, 100, 105, 110, 120}, 640, 100)
	if !b.Anomalous || b.ChangePercent < 400 {
		t.Fatalf("unexpected baseline: %+v", b)
	}
}
func TestBaselineRequiresHistory(t *testing.T) {
	if CompareBaseline([]float64{1, 2}, 100, 50).Anomalous {
		t.Fatal("insufficient evidence must not alert")
	}
}
