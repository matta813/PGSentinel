package analyzer

import (
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestWaitEventsIgnoreTrivialNormalWaits(t *testing.T) {
	snapshot := models.Snapshot{ServerID: "server", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, WaitEvents: []models.WaitEventSample{{PID: 1, WaitEventType: "Client", WaitEvent: "ClientRead", QueryAgeSeconds: 3600}}}
	for _, finding := range New(DefaultThresholds()).Analyze(snapshot) {
		if finding.Category == "Wait Events" {
			t.Fatalf("unexpected finding: %s", finding.Title)
		}
	}
}

func TestWaitEventsFindLockPressureAndCorrelateLocks(t *testing.T) {
	waits := []models.WaitEventSample{
		{PID: 1, Database: "app", WaitEventType: "Lock", WaitEvent: "transactionid", QueryAgeSeconds: 70},
		{PID: 2, Database: "app", WaitEventType: "Lock", WaitEvent: "transactionid", QueryAgeSeconds: 65},
		{PID: 3, Database: "audit", WaitEventType: "Lock", WaitEvent: "tuple", QueryAgeSeconds: 61},
	}
	snapshot := models.Snapshot{ServerID: "server", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, WaitEvents: waits, Locks: []models.LockInfo{{BlockedPID: 1, BlockingPID: 9}}}
	for _, finding := range New(DefaultThresholds()).Analyze(snapshot) {
		if finding.RuleID == "wait-lock-pressure" {
			if finding.Confidence != models.ConfidenceHigh {
				t.Fatalf("confidence=%s", finding.Confidence)
			}
			if finding.Fingerprint == "" || finding.Resource != "Lock" {
				t.Fatalf("unstable identity: %#v", finding)
			}
			return
		}
	}
	t.Fatal("expected lock wait pressure finding")
}

func TestWaitEventsSupportUnknownClassWithoutFinding(t *testing.T) {
	snapshot := models.Snapshot{ServerID: "server", Capabilities: map[string]bool{"pg_stat_statements": true}, Settings: map[string]string{"track_io_timing": "on"}, WaitEvents: []models.WaitEventSample{{PID: 1, WaitEventType: "FutureClass", QueryAgeSeconds: 999}}}
	for _, finding := range New(DefaultThresholds()).Analyze(snapshot) {
		if finding.Category == "Wait Events" {
			t.Fatalf("unexpected finding: %s", finding.Title)
		}
	}
}
