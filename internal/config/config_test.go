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
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-encryption-key-32-chars-min")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "a-secure-test-password")
	t.Setenv("PGSENTINEL_TRUSTED_PROXY_CIDRS", "10.0.0.0/8, 192.0.2.1/32")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddr != ":8080" || c.FastInterval.Seconds() != 5 || len(c.TrustedProxyCIDRs) != 2 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if c.FanoutDatabaseLimit != 32 {
		t.Fatalf("default fanout database limit = %d, want 32", c.FanoutDatabaseLimit)
	}
}

func TestLoadParsesFanoutDatabaseLimit(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-encryption-key-32-chars-min")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "a-secure-test-password")
	t.Setenv("PGSENTINEL_FANOUT_DATABASE_LIMIT", "8")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.FanoutDatabaseLimit != 8 {
		t.Fatalf("fanout database limit = %d, want 8", c.FanoutDatabaseLimit)
	}
	t.Setenv("PGSENTINEL_FANOUT_DATABASE_LIMIT", "invalid")
	c, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.FanoutDatabaseLimit != 32 {
		t.Fatalf("invalid fanout value must fall back, got %d", c.FanoutDatabaseLimit)
	}
}

func TestLoadRequiresStrongAdminPassword(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-encryption-key-32-chars-min")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "short")
	if _, err := Load(); err == nil {
		t.Fatal("expected short admin password error")
	}
}

func TestLoadRejectsInvalidBooleanConfiguration(t *testing.T) {
	for _, key := range []string{"PGSENTINEL_SECURE_COOKIES", "PGSENTINEL_ALLOW_PRIVATE_NOTIFICATION_TARGETS"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
			t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-encryption-key-32-chars-min")
			t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "a-secure-test-password")
			t.Setenv(key, "treu")
			if _, err := Load(); err == nil {
				t.Fatalf("invalid %s value was silently accepted", key)
			}
		})
	}
}
