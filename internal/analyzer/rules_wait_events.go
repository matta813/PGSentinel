package analyzer

import (
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

const (
	minimumConcurrentLockWaits = 3
	minimumConcentratedWaits   = 5
	meaningfulWaitQueryAge     = 60
	concentratedWaitQueryAge   = 120
)

func (e *Engine) analyzeWaitEvents(s models.Snapshot) []models.Finding {
	out := []models.Finding{}
	waitClasses := map[string][]models.WaitEventSample{}
	for _, sample := range s.WaitEvents {
		waitClasses[sample.WaitEventType] = append(waitClasses[sample.WaitEventType], sample)
	}
	if lockWaits := waitClasses["Lock"]; len(lockWaits) >= minimumConcurrentLockWaits {
		oldest := oldestQueryAge(lockWaits)
		if oldest >= meaningfulWaitQueryAge || len(s.Locks) > 0 {
			databases := affectedWaitDatabases(lockWaits)
			evidence := []models.Evidence{{Label: "Sessions reporting Lock waits", Value: fmt.Sprint(len(lockWaits))}, {Label: "Affected databases", Value: fmt.Sprint(databases)}, {Label: "Oldest active query age", Value: fmt.Sprintf("%.0f seconds", oldest)}, {Label: "Blocked sessions observed", Value: fmt.Sprint(len(s.Locks))}}
			confidence := models.ConfidenceMedium
			if len(s.Locks) > 0 {
				confidence = models.ConfidenceHigh
			}
			f := newFinding("wait-lock-pressure", s.ServerID, "", "Lock", models.SeverityHigh, "Wait Events", "Concurrent lock wait pressure detected", fmt.Sprintf("%d sessions currently report PostgreSQL Lock waits; %d blocked sessions are independently visible in the Locks snapshot.", len(lockWaits), len(s.Locks)), "Application requests may experience increased latency while sessions wait for lock acquisition.", confidence, evidence)
			f.Cause = "PostgreSQL currently reports concurrent Lock waits. This correlation does not by itself identify a root cause or prove which blocker is responsible."
			f.Suggestions = []models.Suggestion{{Title: "Inspect the blocking chain in Locks", Detail: "Compare blocking PIDs and long-lived transactions before taking action."}}
			out = append(out, f)
		}
	}
	for _, class := range []string{"IO", "LWLock"} {
		waits := waitClasses[class]
		if len(waits) < minimumConcentratedWaits || oldestQueryAge(waits) < concentratedWaitQueryAge {
			continue
		}
		f := newFinding("wait-class-concentration", s.ServerID, "", class, models.SeverityMedium, "Wait Events", fmt.Sprintf("Significant %s wait concentration detected", class), fmt.Sprintf("%d sessions currently report PostgreSQL %s waits; the oldest active query is %.0f seconds old.", len(waits), class, oldestQueryAge(waits)), "Concurrent waits may increase workload latency, but this snapshot alone does not establish resource saturation or root cause.", models.ConfidenceMedium, []models.Evidence{{Label: "Wait class", Value: class}, {Label: "Waiting sessions", Value: fmt.Sprint(len(waits))}, {Label: "Affected databases", Value: fmt.Sprint(affectedWaitDatabases(waits))}, {Label: "Oldest active query age", Value: fmt.Sprintf("%.0f seconds", oldestQueryAge(waits))}})
		f.Cause = "PostgreSQL currently reports a concentration in this wait class. The observation is correlation evidence, not proof of the underlying cause."
		f.Suggestions = []models.Suggestion{{Title: "Inspect the affected queries and wait events", Detail: "Compare query, lock, host, and workload evidence before changing production."}}
		out = append(out, f)
	}
	return out
}

func oldestQueryAge(samples []models.WaitEventSample) float64 {
	oldest := 0.0
	for _, sample := range samples {
		if sample.QueryAgeSeconds > oldest {
			oldest = sample.QueryAgeSeconds
		}
	}
	return oldest
}

func affectedWaitDatabases(samples []models.WaitEventSample) int {
	databases := map[string]bool{}
	for _, sample := range samples {
		if sample.Database != "" {
			databases[sample.Database] = true
		}
	}
	return len(databases)
}
