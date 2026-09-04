package collector

import (
	"log/slog"
	"regexp"
	"sync"
	"time"
)

type collectorFailureKey struct {
	serverID string
	database string
	kind     string
}

type collectorFailureState struct {
	errorText      string
	failures       int
	sameErrorCount int
	lastReported   time.Time
}

type collectorDiagnostics struct {
	log          *slog.Logger
	interval     time.Duration
	mu           sync.Mutex
	failures     map[collectorFailureKey]collectorFailureState
	cacheReports map[collectorFailureKey]time.Time
}

var collectorSecret = regexp.MustCompile(`(?i)(password\s*=\s*|postgres(?:ql)?://[^:/\s]+:)([^\s@]+)`)

func newCollectorDiagnostics(log *slog.Logger, interval time.Duration) *collectorDiagnostics {
	return &collectorDiagnostics{log: log, interval: interval, failures: make(map[collectorFailureKey]collectorFailureState), cacheReports: make(map[collectorFailureKey]time.Time)}
}

func (d *collectorDiagnostics) failed(serverID, database, kind, collector string, err error, duration time.Duration, at time.Time) {
	key := collectorFailureKey{serverID: serverID, database: database, kind: kind}
	d.mu.Lock()
	state := d.failures[key]
	errorText := collectorSecret.ReplaceAllString(err.Error(), "${1}[REDACTED]")
	state.failures++
	changed := state.errorText != "" && state.errorText != errorText
	if state.errorText == errorText {
		state.sameErrorCount++
	} else {
		state.errorText = errorText
		state.sameErrorCount = 1
	}
	report := state.failures == 1 || changed || at.Sub(state.lastReported) >= d.interval
	if report {
		state.lastReported = at
	}
	d.failures[key] = state
	d.mu.Unlock()
	if !report {
		return
	}
	message := "collector failed"
	if state.failures > 1 && !changed {
		message = "collector still failing"
	}
	d.log.Warn(message, "server_id", serverID, "database", database, "kind", kind, "collector", collector, "failure_count", state.failures, "retry_count", state.failures-1, "same_error_count", state.sameErrorCount, "duration", duration, "error", errorText)
}

func (d *collectorDiagnostics) cached(serverID, kind string, at time.Time) {
	key := collectorFailureKey{serverID: serverID, kind: kind}
	d.mu.Lock()
	last := d.cacheReports[key]
	report := last.IsZero() || at.Sub(last) >= d.interval
	if report {
		d.cacheReports[key] = at
	}
	d.mu.Unlock()
	if report {
		d.log.Info("collector cached evidence restored", "server_id", serverID, "database", "", "kind", kind, "collector", kind, "cached", true)
	}
}

func (d *collectorDiagnostics) succeeded(serverID, database, kind, collector string, duration time.Duration) {
	key := collectorFailureKey{serverID: serverID, database: database, kind: kind}
	d.mu.Lock()
	state, failed := d.failures[key]
	if failed {
		delete(d.failures, key)
	}
	d.mu.Unlock()
	if failed {
		d.log.Info("collector recovered", "server_id", serverID, "database", database, "kind", kind, "collector", collector, "previous_failures", state.failures, "duration", duration)
	}
}
