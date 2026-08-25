package analyzer

import (
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestThresholdOverridesChangeOnlyAllowlistedRule(t *testing.T) {
	defaults := DefaultThresholds()
	got := ApplyThresholdOverrides(defaults, []models.ThresholdOverride{{RuleID: "standby-replay-lag", Value: 180}, {RuleID: "unknown", Value: 0}})
	if got.ReplicaLagSeconds != 180 || got.ConnectionHigh != defaults.ConnectionHigh {
		t.Fatalf("thresholds=%#v", got)
	}
	for rule, spec := range ThresholdSpecs() {
		if spec.Min <= 0 || spec.Max <= spec.Min || spec.Default < spec.Min || spec.Default > spec.Max {
			t.Fatalf("unsafe spec %s=%#v", rule, spec)
		}
	}
}
