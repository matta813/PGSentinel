package storage

import (
	"context"
	"github.com/matta813/pgsentinel/internal/models"
	"path/filepath"
	"testing"
	"time"
)

func TestFindingLifecycle(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "s", Name: "db", Host: "localhost", Port: 5432, User: "u", Password: "p", SSLMode: "disable"}
	if err = s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	f := models.Finding{ID: "f", RuleID: "r", Fingerprint: "fp", ServerID: "s", Severity: models.SeverityHigh, Category: "Vacuum", Title: "T", Status: "active", StartedAt: time.Now(), UpdatedAt: time.Now()}
	if err = s.UpsertFindings(ctx, "s", []models.Finding{f}); err != nil {
		t.Fatal(err)
	}
	if err = s.UpsertFindings(ctx, "s", nil); err != nil {
		t.Fatal(err)
	}
	got, err := s.ListFindings(ctx, "resolved", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d: %v", len(got), err)
	}
}

func TestUpdateServerPreservesAndRotatesPassword(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "servers.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	server := models.Server{ID: "server", Name: "old", Host: "old.example", Port: 5432, User: "old-user", Password: "old-password", SSLMode: "prefer"}
	if err = s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}

	update := models.Server{ID: server.ID, Name: "new", Host: "new.example", Port: 6432, User: "new-user", SSLMode: "require", Tags: []string{"production"}}
	if err = s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetServer(ctx, server.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "new" || got.Host != "new.example" || got.Port != 6432 || got.Password != "old-password" {
		t.Fatalf("unexpected preserved update: %#v", got)
	}

	update.Password = "rotated-password"
	if err = s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetServer(ctx, server.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "rotated-password" {
		t.Fatalf("password was not rotated: %q", got.Password)
	}
}
