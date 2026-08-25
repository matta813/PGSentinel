package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/matta813/pgsentinel/migrations"
)

func TestMigration003UpgradesReleasedSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"001_initial.sql", "002_users.sql"} {
		body, readErr := migrations.FS.ReadFile(name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err := db.ExecContext(context.Background(), string(body)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(1,'2026-01-01T00:00:00Z'),(2,'2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path, "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, table := range []string{"finding_notification_events", "finding_notification_deliveries"} {
		var count int
		if err := store.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s count=%d err=%v", table, count, err)
		}
	}
}

func TestMigration004UpgradesVersion060Schema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "upgrade-060.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"001_initial.sql", "002_users.sql", "003_finding_notifications.sql"} {
		body, readErr := migrations.FS.ReadFile(name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if _, err := db.ExecContext(context.Background(), string(body)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(1,'2026-01-01T00:00:00Z'),(2,'2026-01-01T00:00:00Z'),(3,'2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path, "long-enough-encryption-key-32-chars")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	for _, table := range []string{"notification_routes", "notification_route_destinations"} {
		var count int
		if err := store.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("table %s count=%d err=%v", table, count, err)
		}
	}
	var version int
	if err := store.DB.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil || version != 5 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	var freshnessTable int
	if err := store.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='collection_resource_status'`).Scan(&freshnessTable); err != nil || freshnessTable != 1 {
		t.Fatalf("freshness table count=%d err=%v", freshnessTable, err)
	}
}
