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

func TestTablesSQLTreatsMissingIndexAggregateAsZero(t *testing.T) {
	if !strings.Contains(tablesSQL, "COALESCE(s.idx_scan,0)") {
		t.Fatal("table query must normalize PostgreSQL's NULL idx_scan for tables without indexes")
	}
}

func TestPartialTableCollectionUsesCacheAndRecovers(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "tables.db"), "long-enough-encryption-key-32-chars")
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
	cached := []models.TableStat{{Database: "failed", Table: "cached"}}
	if err := store.SaveSnapshot(ctx, "server", "tables", cached, at.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	partial, fresh := manager.persistTableCollection(ctx, "server", []models.TableStat{{Database: "healthy", Table: "fresh"}}, false, at)
	if fresh {
		t.Fatal("partial collection reported fresh")
	}
	if len(partial) != 1 || partial[0].Table != "cached" {
		t.Fatalf("partial collection did not preserve cache: %#v", partial)
	}
	status, err := store.ListCollectionResources(ctx, "server", at)
	if err != nil || len(status) != 1 || status[0].State != "partial" || status[0].ConsecutiveFailures != 1 {
		t.Fatalf("partial status=%#v err=%v", status, err)
	}
	recovered := []models.TableStat{{Database: "healthy", Table: "recovered"}}
	got, collectionFresh := manager.persistTableCollection(ctx, "server", recovered, true, at.Add(time.Minute))
	if !collectionFresh {
		t.Fatal("successful collection did not report fresh")
	}
	if len(got) != 1 || got[0].Table != "recovered" {
		t.Fatalf("successful collection did not recover: %#v", got)
	}
	status, err = store.ListCollectionResources(ctx, "server", at.Add(time.Minute))
	if err != nil || status[0].State != "fresh" || status[0].ConsecutiveFailures != 0 {
		t.Fatalf("recovered status=%#v err=%v", status, err)
	}
}

func TestPartialTableCollectionWithoutCacheKeepsSuccessfulDatabases(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "tables.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.CreateServer(context.Background(), &models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	want := []models.TableStat{{Database: "healthy", Table: "events"}}
	got, fresh := manager.persistTableCollection(context.Background(), "server", want, false, time.Now().UTC())
	if fresh {
		t.Fatal("partial collection reported fresh")
	}
	if len(got) != 1 || got[0].Database != "healthy" {
		t.Fatalf("successful database evidence was discarded: %#v", got)
	}
	var persisted []models.TableStat
	if err := store.LatestSnapshot(context.Background(), "server", "tables", &persisted); err != nil || len(persisted) != 1 {
		t.Fatalf("partial evidence not persisted: %#v err=%v", persisted, err)
	}
}

func TestCompleteEmptyTableCollectionIsFresh(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "tables.db"), "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.CreateServer(context.Background(), &models.Server{ID: "server", Name: "server", Host: "localhost", Port: 5432, User: "monitor", Password: "secret", SSLMode: "disable"}); err != nil {
		t.Fatal(err)
	}
	manager := NewManager(store, slog.New(slog.NewTextHandler(io.Discard, nil)), Schedule{})
	got, fresh := manager.persistTableCollection(context.Background(), "server", []models.TableStat{}, true, time.Now().UTC())
	if !fresh {
		t.Fatal("complete empty collection did not report fresh")
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("complete empty result must remain a non-nil empty snapshot: %#v", got)
	}
}
