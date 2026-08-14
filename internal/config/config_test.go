package config

import "testing"

func TestLoadRequiresEncryptionKey(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected missing encryption key error")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-key")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "a-secure-test-password")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddr != ":8080" || c.FastInterval.Seconds() != 5 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}

func TestLoadRequiresStrongAdminPassword(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-key")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "short")
	if _, err := Load(); err == nil {
		t.Fatal("expected short admin password error")
	}
}
