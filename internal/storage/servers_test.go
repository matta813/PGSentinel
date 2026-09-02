package storage

import (
	"database/sql"
	"errors"
	"github.com/matta813/pgsentinel/internal/models"
	"testing"
)

func TestDeleteServerReportsMissingTarget(t *testing.T) {
	s, ctx := testMonitoringStore(t, "delete-missing-server")
	if err := s.DeleteServer(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("DeleteServer error = %v, want sql.ErrNoRows", err)
	}
}

func TestUpdateServerPreservesAndRotatesPassword(t *testing.T) {
	s, ctx := testMonitoringStore(t, "servers")
	server := models.Server{ID: "server", Name: "old", Host: "old.example", Port: 5432, User: "old-user", Password: "old-password", SSLMode: "prefer"}
	if err := s.CreateServer(ctx, &server); err != nil {
		t.Fatal(err)
	}
	update := models.Server{ID: server.ID, Name: "new", Host: "new.example", Port: 6432, User: "new-user", SSLMode: "require", Tags: []string{"production"}}
	if err := s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetServer(ctx, server.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "new" || got.Host != "new.example" || got.Port != 6432 || got.Password != "old-password" {
		t.Fatalf("unexpected update: %#v", got)
	}
	update.Password = "rotated-password"
	if err := s.UpdateServer(ctx, &update); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetServer(ctx, server.ID, true)
	if err != nil || got.Password != "rotated-password" {
		t.Fatalf("password=%q err=%v", got.Password, err)
	}
}
