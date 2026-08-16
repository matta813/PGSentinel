package collector

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

func TestManagerUsesConfiguredRetention(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "manager.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err = store.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, at := range []time.Time{now.Add(-3 * time.Hour), now.Add(-30 * time.Minute)} {
		if err = store.SaveSnapshot(ctx, server.ID, "core", map[string]bool{"ok": true}, at); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Minute, time.Hour)
	if err = manager.prune(ctx, now); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM snapshots`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("configured retention kept %d snapshots", count)
	}
}
