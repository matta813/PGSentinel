package collector

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

func TestIndexesSQLNormalizesMissingStatisticsRow(t *testing.T) {
	if !strings.Contains(indexesSQL, "COALESCE(s.idx_scan,0)") {
		t.Fatal("index query must normalize a missing pg_stat_user_indexes row")
	}
}

func TestPartialIndexCollectionPreservesCacheAndRecovers(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "indexes.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.CreateServer(ctx, &models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	at := time.Now().UTC()
	if err := store.SaveSnapshot(ctx, "server", "indexes", []models.IndexStat{{Database: "cached", Index: "cached_idx"}}, at.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	partial, fresh := manager.persistIndexCollection(ctx, "server", []models.IndexStat{{Database: "healthy", Index: "fresh_idx"}}, false, at)
	if fresh || len(partial) != 1 || partial[0].Index != "cached_idx" {
		t.Fatalf("partial collection=%#v fresh=%v", partial, fresh)
	}
	recovered, fresh := manager.persistIndexCollection(ctx, "server", []models.IndexStat{{Database: "healthy", Index: "recovered_idx"}}, true, at.Add(time.Minute))
	if !fresh || len(recovered) != 1 || recovered[0].Index != "recovered_idx" {
		t.Fatalf("recovered collection=%#v fresh=%v", recovered, fresh)
	}
	status, err := store.ListCollectionResources(ctx, "server", at.Add(time.Minute))
	if err != nil || len(status) != 1 || status[0].State != "fresh" || status[0].ConsecutiveFailures != 0 {
		t.Fatalf("status=%#v err=%v", status, err)
	}
}

func TestPartialIndexCollectionWithoutCacheKeepsSuccessfulDatabase(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "indexes.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.CreateServer(ctx, &models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	got, fresh := manager.persistIndexCollection(ctx, "server", []models.IndexStat{{Database: "healthy", Index: "events_idx"}}, false, time.Now().UTC())
	if fresh || len(got) != 1 || got[0].Database != "healthy" {
		t.Fatalf("partial collection=%#v fresh=%v", got, fresh)
	}
}

func TestCompleteEmptyIndexCollectionIsFresh(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "indexes.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	if err := store.CreateServer(ctx, &models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	got, fresh := manager.persistIndexCollection(ctx, "server", []models.IndexStat{}, true, time.Now().UTC())
	if !fresh || got == nil || len(got) != 0 {
		t.Fatalf("complete empty collection=%#v fresh=%v", got, fresh)
	}
}
