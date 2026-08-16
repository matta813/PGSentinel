package collector

import (
	"testing"
	"time"
)

func TestScheduleNormalizesEachCollectorIndependently(t *testing.T) {
	schedule := (Schedule{Fast: 2 * time.Second, Standard: 15 * time.Second, Slow: 2 * time.Minute, Metadata: time.Hour}).normalized()
	if schedule.Fast != 2*time.Second || schedule.Standard != 15*time.Second || schedule.Slow != 2*time.Minute || schedule.Metadata != time.Hour {
		t.Fatalf("custom schedule changed: %#v", schedule)
	}
	fallback := (Schedule{}).normalized()
	if fallback.Fast != 30*time.Second || fallback.Standard != 30*time.Second || fallback.Slow != 5*time.Minute || fallback.Metadata != 30*time.Minute {
		t.Fatalf("unexpected fallback schedule: %#v", fallback)
	}
}

func TestCollectionCyclesSelectExpectedWork(t *testing.T) {
	tests := []struct {
		cycle                          collectionCycle
		fast, standard, slow, metadata bool
	}{
		{cycleFast, true, false, false, false},
		{cycleStandard, false, true, false, false},
		{cycleSlow, false, false, true, false},
		{cycleMetadata, false, false, false, true},
		{cycleAll, true, true, true, true},
	}
	for _, test := range tests {
		if (test.cycle&cycleFast != 0) != test.fast || (test.cycle&cycleStandard != 0) != test.standard || (test.cycle&cycleSlow != 0) != test.slow || (test.cycle&cycleMetadata != 0) != test.metadata {
			t.Fatalf("cycle %d selected unexpected work", test.cycle)
		}
	}
}
