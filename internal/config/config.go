package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr                      string
	DataDir                         string
	DatabasePath                    string
	EncryptionKey                   string
	BootstrapAdminPassword          string
	SecureCookies                   bool
	AllowPrivateNotificationTargets bool
	NotificationAllowedHosts        []string
	TrustedProxyCIDRs               []string
	LogLevel                        string
	FastInterval                    time.Duration
	StatsInterval                   time.Duration
	SlowInterval                    time.Duration
	MetaInterval                    time.Duration
	Retention                       time.Duration
	FanoutDatabaseLimit             int
	FrontendDir                     string
}

func Load() (Config, error) {
	dataDir := env("PGSENTINEL_DATA_DIR", "/data")
	c := Config{
		ListenAddr: env("PGSENTINEL_LISTEN_ADDR", ":8080"), DataDir: dataDir,
		DatabasePath:  filepath.Join(dataDir, "pgsentinel.db"),
		EncryptionKey: os.Getenv("PGSENTINEL_ENCRYPTION_KEY"), BootstrapAdminPassword: os.Getenv("PGSENTINEL_ADMIN_PASSWORD"), LogLevel: env("PGSENTINEL_LOG_LEVEL", "info"),
		SecureCookies: boolean("PGSENTINEL_SECURE_COOKIES", false), AllowPrivateNotificationTargets: boolean("PGSENTINEL_ALLOW_PRIVATE_NOTIFICATION_TARGETS", false),
		NotificationAllowedHosts: list("PGSENTINEL_NOTIFICATION_ALLOWED_HOSTS"),
		TrustedProxyCIDRs:        list("PGSENTINEL_TRUSTED_PROXY_CIDRS"),
		FastInterval:             duration("PGSENTINEL_FAST_INTERVAL", 5*time.Second), StatsInterval: duration("PGSENTINEL_STATS_INTERVAL", 30*time.Second),
		SlowInterval: duration("PGSENTINEL_SLOW_INTERVAL", 5*time.Minute), MetaInterval: duration("PGSENTINEL_META_INTERVAL", 30*time.Minute),
		Retention:           duration("PGSENTINEL_RETENTION", 30*24*time.Hour),
		FanoutDatabaseLimit: positiveInt("PGSENTINEL_FANOUT_DATABASE_LIMIT", 32),
		FrontendDir:         env("PGSENTINEL_FRONTEND_DIR", "./frontend/dist"),
	}
	if c.EncryptionKey == "" {
		return Config{}, fmt.Errorf("PGSENTINEL_ENCRYPTION_KEY is required; generate one with openssl rand -base64 32")
	}
	if len(c.EncryptionKey) < 32 {
		return Config{}, fmt.Errorf("PGSENTINEL_ENCRYPTION_KEY must contain at least 32 characters")
	}
	if len(c.BootstrapAdminPassword) < 12 {
		return Config{}, fmt.Errorf("PGSENTINEL_ADMIN_PASSWORD must contain at least 12 characters")
	}
	if err := os.MkdirAll(c.DataDir, 0o750); err != nil {
		return Config{}, fmt.Errorf("create data directory: %w", err)
	}
	return c, nil
}

func boolean(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	return err == nil && parsed
}

func list(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func duration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	return fallback
}
func positiveInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
