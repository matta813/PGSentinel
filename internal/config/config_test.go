package config

import (
	"testing"
	"time"
)

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
	if c.MetricRawRetention != 24*time.Hour || c.MetricMediumRetention != 30*24*time.Hour || c.MetricLongRetention != 365*24*time.Hour {
		t.Fatalf("unexpected metric retention defaults: %+v", c)
	}
}

func TestLoadValidatesMetricRetentionTiers(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-encryption-key-32-chars-min")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "a-secure-test-password")
	t.Setenv("PGSENTINEL_METRIC_RAW_RETENTION", "48h")
	t.Setenv("PGSENTINEL_METRIC_MEDIUM_RETENTION", "24h")
	if _, err := Load(); err == nil {
		t.Fatal("expected descending metric retention tiers to be rejected")
	}
	t.Setenv("PGSENTINEL_METRIC_MEDIUM_RETENTION", "720h")
	t.Setenv("PGSENTINEL_METRIC_LONG_RETENTION", "8760h")
	c, err := Load()
	if err != nil || c.MetricRawRetention != 48*time.Hour {
		t.Fatalf("valid metric retention rejected: %+v, %v", c, err)
	}
}

func TestLoadRejectsInvalidMetricRetentionDuration(t *testing.T) {
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "test-encryption-key-32-chars-min")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "a-secure-test-password")
	t.Setenv("PGSENTINEL_METRIC_RAW_RETENTION", "forever")
	if _, err := Load(); err == nil {
		t.Fatal("invalid metric retention duration was silently accepted")
	}
}

func TestLoadValidatesSnapshotSampleLimit(t *testing.T) {
	t.Setenv("PGSENTINEL_ENCRYPTION_KEY", "long-enough-encryption-key-32-chars")
	t.Setenv("PGSENTINEL_ADMIN_PASSWORD", "long-enough-admin-password")
	t.Setenv("PGSENTINEL_DATA_DIR", t.TempDir())
	t.Setenv("PGSENTINEL_MAX_SNAPSHOTS_PER_RESOURCE", "9")
	if _, err := Load(); err == nil {
		t.Fatal("unsafe snapshot sample limit was accepted")
	}
	t.Setenv("PGSENTINEL_MAX_SNAPSHOTS_PER_RESOURCE", "250")
	c, err := Load()
	if err != nil || c.MaxSnapshotsPerResource != 250 {
		t.Fatalf("valid snapshot sample limit rejected: %+v, %v", c, err)
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
