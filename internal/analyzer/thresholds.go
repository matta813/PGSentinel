package analyzer

type Thresholds struct {
	ConnectionHigh, ConnectionCritical, IdleTransactionSeconds, LongTransactionSeconds float64
	LongQuerySeconds, DeadTupleRatio, VacuumProgress, CacheHitLow, RollbackRatio       float64
	QueryImpactHigh, ReplicaLagSeconds, SlotRetainedBytes                              float64
	RequestedCheckpointRatio, CheckpointIntervalSeconds                                float64
}

func DefaultThresholds() Thresholds {
	return Thresholds{80, 95, 300, 900, 60, 20, 100, 95, 5, 250, 60, 1024 * 1024 * 1024, 20, 300}
}
