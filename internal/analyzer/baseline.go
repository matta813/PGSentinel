package analyzer

import (
	"math"
	"sort"
)

type Baseline struct {
	Median        float64 `json:"median"`
	Lower         float64 `json:"lower"`
	Upper         float64 `json:"upper"`
	ChangePercent float64 `json:"changePercent"`
	Anomalous     bool    `json:"anomalous"`
}

// CompareBaseline uses median and median absolute deviation, which tolerate workload spikes better than mean/stddev.
func CompareBaseline(history []float64, current float64, minChangePercent float64) Baseline {
	if len(history) < 6 {
		return Baseline{Median: median(history)}
	}
	mid := median(history)
	dev := make([]float64, len(history))
	for i, v := range history {
		dev[i] = math.Abs(v - mid)
	}
	mad := median(dev)
	if mad == 0 {
		mad = math.Max(math.Abs(mid)*.05, .001)
	}
	change := 0.0
	if mid != 0 {
		change = (current - mid) / math.Abs(mid) * 100
	}
	return Baseline{Median: mid, Lower: math.Max(0, mid-3*mad), Upper: mid + 3*mad, ChangePercent: change, Anomalous: current > mid+3*mad && change >= minChangePercent}
}
func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	m := len(c) / 2
	if len(c)%2 == 0 {
		return (c[m-1] + c[m]) / 2
	}
	return c[m]
}
