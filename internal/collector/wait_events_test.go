package collector

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestWaitEventCollectionIsBoundedAndReadOnly(t *testing.T) {
	query := strings.ToLower(waitEventsSQL)
	for _, wanted := range []string{"pg_stat_activity", "wait_event is not null", "pid <> pg_backend_pid()", "limit 500", "left(coalesce(query, ''), 2000)"} {
		if !strings.Contains(query, wanted) {
			t.Fatalf("missing %q", wanted)
		}
	}
	for _, forbidden := range []string{"pg_cancel_backend", "pg_terminate_backend", "explain", "update ", "delete ", "insert ", "alter ", "create ", "drop "} {
		if strings.Contains(query, forbidden) {
			t.Fatalf("wait collection contains %q", forbidden)
		}
	}
}

func TestBoundWaitEventQueryTrimsAndPreservesUTF8(t *testing.T) {
	value := boundWaitEventQuery("  " + strings.Repeat("é", 1500) + "  ")
	if len(value) > maxWaitEventQueryBytes {
		t.Fatalf("query has %d bytes", len(value))
	}
	if !utf8.ValidString(value) {
		t.Fatal("query is not valid UTF-8")
	}
}
