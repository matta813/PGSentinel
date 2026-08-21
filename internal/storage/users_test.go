package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestUserPasswordLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "users.db"), "a sufficiently long user storage key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	user := models.User{ID: "user-1", Username: "admin", PasswordHash: []byte("initial-hash"), PasswordSalt: []byte("initial-salt"), MustChangePassword: true}
	if err := store.CreateUser(ctx, &user); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetUserByUsername(ctx, "ADMIN")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Username != "admin" || !stored.MustChangePassword || string(stored.PasswordHash) != "initial-hash" {
		t.Fatalf("unexpected stored user: %#v", stored)
	}
	if err := store.UpdateUserPassword(ctx, stored.ID, []byte("new-hash"), []byte("new-salt")); err != nil {
		t.Fatal(err)
	}
	stored, err = store.GetUserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if stored.MustChangePassword || string(stored.PasswordHash) != "new-hash" || string(stored.PasswordSalt) != "new-salt" {
		t.Fatalf("password was not updated: %#v", stored)
	}
}

func TestUserStorageRejectsDuplicateUsername(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "users.db"), "a sufficiently long user storage key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := context.Background()
	first := models.User{ID: "user-1", Username: "admin", PasswordHash: []byte("hash"), PasswordSalt: []byte("salt"), MustChangePassword: true}
	second := models.User{ID: "user-2", Username: "ADMIN", PasswordHash: []byte("hash"), PasswordSalt: []byte("salt"), MustChangePassword: true}
	if err := store.CreateUser(ctx, &first); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateUser(ctx, &second); err == nil {
		t.Fatal("case-insensitive duplicate username was accepted")
	}
	if _, err := store.GetUserByUsername(ctx, "missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing user error=%v", err)
	}
}
