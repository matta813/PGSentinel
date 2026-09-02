package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func testMonitoringStore(t *testing.T, name string) (*Store, context.Context) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), name+".db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, context.Background()
}
