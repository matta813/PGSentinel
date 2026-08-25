package analyzer

import "github.com/matta813/pgsentinel/internal/models"

type ThresholdSpec struct {
	Label   string  `json:"label"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Default float64 `json:"default"`
	Unit    string  `json:"unit"`
}

func ThresholdSpecs() map[string]ThresholdSpec {
	return map[string]ThresholdSpec{
		"connection-utilization-high":     {"Connection pressure", 50, 90, 80, "percent"},
		"connection-utilization-critical": {"Critical connection pressure", 91, 99, 95, "percent"},
		"idle-in-transaction":             {"Idle transaction age", 60, 86400, 300, "seconds"},
		"long-transaction":                {"Long transaction age", 60, 604800, 900, "seconds"},
		"blocking-queries":                {"Blocking duration", 5, 3600, 60, "seconds"},
		"dead-tuples":                     {"Dead tuple ratio", 5, 80, 20, "percent"},
		"vacuum-behind":                   {"Vacuum trigger progress", 50, 300, 100, "percent"},
		"cache-hit":                       {"Minimum cache hit ratio", 50, 99.9, 95, "percent"},
		"rollback-ratio":                  {"Rollback ratio", 1, 50, 5, "percent"},
		"query-impact":                    {"Query impact score", 50, 1000, 250, "score"},
		"standby-replay-lag":              {"Replica replay lag", 10, 86400, 60, "seconds"},
		"inactive-slot-wal":               {"Inactive slot retained WAL", 67108864, 1099511627776, 1073741824, "bytes"},
		"requested-checkpoints":           {"Requested checkpoint ratio", 5, 80, 20, "percent"},
		"checkpoint-frequency":            {"Minimum checkpoint interval", 30, 3600, 300, "seconds"},
	}
}

func ApplyThresholdOverrides(base Thresholds, overrides []models.ThresholdOverride) Thresholds {
	for _, item := range overrides {
		switch item.RuleID {
		case "connection-utilization-high":
			base.ConnectionHigh = item.Value
		case "connection-utilization-critical":
			base.ConnectionCritical = item.Value
		case "idle-in-transaction":
			base.IdleTransactionSeconds = item.Value
		case "long-transaction":
			base.LongTransactionSeconds = item.Value
		case "blocking-queries":
			base.LongQuerySeconds = item.Value
		case "dead-tuples":
			base.DeadTupleRatio = item.Value
		case "vacuum-behind":
			base.VacuumProgress = item.Value
		case "cache-hit":
			base.CacheHitLow = item.Value
		case "rollback-ratio":
			base.RollbackRatio = item.Value
		case "query-impact":
			base.QueryImpactHigh = item.Value
		case "standby-replay-lag":
			base.ReplicaLagSeconds = item.Value
		case "inactive-slot-wal":
			base.SlotRetainedBytes = item.Value
		case "requested-checkpoints":
			base.RequestedCheckpointRatio = item.Value
		case "checkpoint-frequency":
			base.CheckpointIntervalSeconds = item.Value
		}
	}
	return base
}
