package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	user := models.User{ID: "user-1", Username: "admin", Role: "administrator", PasswordHash: []byte("initial-hash"), PasswordSalt: []byte("initial-salt"), MustChangePassword: true}
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
	first := models.User{ID: "user-1", Username: "admin", Role: "administrator", PasswordHash: []byte("hash"), PasswordSalt: []byte("salt"), MustChangePassword: true}
	second := models.User{ID: "user-2", Username: "ADMIN", Role: "viewer", PasswordHash: []byte("hash"), PasswordSalt: []byte("salt"), MustChangePassword: true}
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

func TestUserStorageIsBounded(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "bounded-users.db"), "a sufficiently long user storage key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	for i := 0; i < 100; i++ {
		user := models.User{ID: fmt.Sprintf("user-%d", i), Username: fmt.Sprintf("user-%d", i), Role: "viewer", PasswordHash: []byte("hash"), PasswordSalt: []byte("salt"), MustChangePassword: true}
		if err := store.CreateUser(context.Background(), &user); err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
	}
	extra := models.User{ID: "extra", Username: "extra", Role: "viewer", PasswordHash: []byte("hash"), PasswordSalt: []byte("salt"), MustChangePassword: true}
	if err := store.CreateUser(context.Background(), &extra); err == nil {
		t.Fatal("101st local user was accepted")
	}
}
