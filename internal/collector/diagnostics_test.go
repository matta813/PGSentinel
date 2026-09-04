package collector

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestCollectorDiagnosticsFailureRateLimitErrorChangeAndRecovery(t *testing.T) {
	var output bytes.Buffer
	diagnostics := newCollectorDiagnostics(slog.New(slog.NewJSONHandler(&output, nil)), time.Minute)
	at := time.Date(2026, 9, 4, 8, 0, 0, 0, time.UTC)
	diagnostics.failed("server-1", "tracearr", "tables", "tables", errors.New("cannot scan NULL"), 12*time.Millisecond, at)
	diagnostics.failed("server-1", "tracearr", "tables", "tables", errors.New("cannot scan NULL"), 10*time.Millisecond, at.Add(10*time.Second))
	if got := strings.Count(output.String(), `"msg":"collector failed"`); got != 1 {
		t.Fatalf("first failure count=%d logs=%s", got, output.String())
	}
	diagnostics.failed("server-1", "tracearr", "tables", "tables", errors.New("cannot scan NULL"), 11*time.Millisecond, at.Add(time.Minute))
	diagnostics.failed("server-1", "tracearr", "tables", "tables", errors.New("permission denied"), 9*time.Millisecond, at.Add(time.Minute+time.Second))
	diagnostics.succeeded("server-1", "tracearr", "tables", "tables", 8*time.Millisecond)
	logs := output.String()
	for _, expected := range []string{`"msg":"collector still failing"`, `"failure_count":3`, `"retry_count":2`, `"same_error_count":3`, `"error":"permission denied"`, `"msg":"collector recovered"`, `"previous_failures":4`, `"server_id":"server-1"`, `"database":"tracearr"`, `"kind":"tables"`} {
		if !strings.Contains(logs, expected) {
			t.Fatalf("missing %s in %s", expected, logs)
		}
	}
}

func TestCollectorDiagnosticsSeparatesDatabasesAndKinds(t *testing.T) {
	var output bytes.Buffer
	diagnostics := newCollectorDiagnostics(slog.New(slog.NewJSONHandler(&output, nil)), time.Hour)
	at := time.Now()
	for _, item := range []struct{ database, kind string }{{"app", "tables"}, {"audit", "tables"}, {"app", "indexes"}} {
		diagnostics.failed("server", item.database, item.kind, item.kind, errors.New("failed"), time.Millisecond, at)
	}
	if got := strings.Count(output.String(), `"msg":"collector failed"`); got != 3 {
		t.Fatalf("independent failures=%d logs=%s", got, output.String())
	}
}

func TestCollectorDiagnosticsRateLimitsCacheAndRedactsCredentials(t *testing.T) {
	var output bytes.Buffer
	diagnostics := newCollectorDiagnostics(slog.New(slog.NewJSONHandler(&output, nil)), time.Minute)
	at := time.Now()
	diagnostics.cached("server", "tables", at)
	diagnostics.cached("server", "tables", at.Add(10*time.Second))
	diagnostics.cached("server", "tables", at.Add(time.Minute))
	diagnostics.failed("server", "app", "tables", "connect", errors.New("postgres://monitor:super-secret@db/app password=another-secret"), time.Millisecond, at)
	logs := output.String()
	if strings.Count(logs, `"cached":true`) != 2 {
		t.Fatalf("cache reports not rate limited: %s", logs)
	}
	if strings.Contains(logs, "super-secret") || strings.Contains(logs, "another-secret") || !strings.Contains(logs, "[REDACTED]") {
		t.Fatalf("credential was not redacted: %s", logs)
	}
}
