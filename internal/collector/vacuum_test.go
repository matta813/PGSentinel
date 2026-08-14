package collector

import "testing"

func TestVacuumProgress(t *testing.T) {
	dead, threshold := 1820000.0, 1950000.0
	got := dead / threshold * 100
	if got < 93 || got > 94 {
		t.Fatalf("got %f", got)
	}
}
