package storage

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/matta813/pgsentinel/internal/models"
)

func TestNotificationDestinationCRUDEncryptsConfig(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "notifications.db"), "long enough encryption key")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	v := models.NotificationDestination{ID: "destination", Provider: "ntfy", Name: "Operations", Enabled: true, Config: map[string]string{"serverUrl": "https://ntfy.sh", "topic": "ops", "token": "top-secret"}}
	if err := s.CreateNotificationDestination(ctx, &v); err != nil {
		t.Fatal(err)
	}
	var encrypted []byte
	if err := s.DB.QueryRowContext(ctx, `SELECT config_cipher FROM notification_configs WHERE id=?`, v.ID).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("top-secret")) {
		t.Fatal("notification secret was stored in plaintext")
	}
	got, err := s.GetNotificationDestination(ctx, v.ID, true)
	if err != nil || got.Config["token"] != "top-secret" {
		t.Fatalf("decrypted destination=%#v err=%v", got, err)
	}
	v.Name, v.Config["token"] = "Platform", "rotated-secret"
	if err := s.UpdateNotificationDestination(ctx, &v); err != nil {
		t.Fatal(err)
	}
	listed, err := s.ListNotificationDestinations(ctx)
	if err != nil || len(listed) != 1 || listed[0].Name != "Platform" || listed[0].Config != nil {
		t.Fatalf("listed=%#v err=%v", listed, err)
	}
	if err := s.DeleteNotificationDestination(ctx, v.ID); err != nil {
		t.Fatal(err)
	}
	listed, err = s.ListNotificationDestinations(ctx)
	if err != nil || len(listed) != 0 {
		t.Fatalf("after delete=%#v err=%v", listed, err)
	}
}
