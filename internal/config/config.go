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
	CollectorDiagnosticInterval     time.Duration
	Retention                       time.Duration
	MetricRawRetention              time.Duration
	MetricMediumRetention           time.Duration
	MetricLongRetention             time.Duration
	FanoutDatabaseLimit             int
	MaxSnapshotsPerResource         int
	FrontendDir                     string
}

func Load() (Config, error) {
	dataDir := env("PGSENTINEL_DATA_DIR", "/data")
	secureCookies, err := boolean("PGSENTINEL_SECURE_COOKIES", false)
	if err != nil {
		return Config{}, err
	}
	allowPrivateTargets, err := boolean("PGSENTINEL_ALLOW_PRIVATE_NOTIFICATION_TARGETS", false)
	if err != nil {
		return Config{}, err
	}
	metricRawRetention, err := strictDuration("PGSENTINEL_METRIC_RAW_RETENTION", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	metricMediumRetention, err := strictDuration("PGSENTINEL_METRIC_MEDIUM_RETENTION", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	metricLongRetention, err := strictDuration("PGSENTINEL_METRIC_LONG_RETENTION", 365*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	collectorDiagnosticInterval, err := strictDuration("PGSENTINEL_COLLECTOR_DIAGNOSTIC_INTERVAL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	c := Config{
		ListenAddr: env("PGSENTINEL_LISTEN_ADDR", ":8080"), DataDir: dataDir,
		DatabasePath:  filepath.Join(dataDir, "pgsentinel.db"),
		EncryptionKey: os.Getenv("PGSENTINEL_ENCRYPTION_KEY"), BootstrapAdminPassword: os.Getenv("PGSENTINEL_ADMIN_PASSWORD"), LogLevel: env("PGSENTINEL_LOG_LEVEL", "info"),
		SecureCookies: secureCookies, AllowPrivateNotificationTargets: allowPrivateTargets,
		NotificationAllowedHosts: list("PGSENTINEL_NOTIFICATION_ALLOWED_HOSTS"),
		TrustedProxyCIDRs:        list("PGSENTINEL_TRUSTED_PROXY_CIDRS"),
		FastInterval:             duration("PGSENTINEL_FAST_INTERVAL", 5*time.Second), StatsInterval: duration("PGSENTINEL_STATS_INTERVAL", 30*time.Second),
		SlowInterval: duration("PGSENTINEL_SLOW_INTERVAL", 5*time.Minute), MetaInterval: duration("PGSENTINEL_META_INTERVAL", 30*time.Minute),
		CollectorDiagnosticInterval: collectorDiagnosticInterval,
		Retention:                   duration("PGSENTINEL_RETENTION", 30*24*time.Hour),
		MetricRawRetention:          metricRawRetention,
		MetricMediumRetention:       metricMediumRetention,
		MetricLongRetention:         metricLongRetention,
		FanoutDatabaseLimit:         positiveInt("PGSENTINEL_FANOUT_DATABASE_LIMIT", 32),
		MaxSnapshotsPerResource:     positiveInt("PGSENTINEL_MAX_SNAPSHOTS_PER_RESOURCE", 120),
		FrontendDir:                 env("PGSENTINEL_FRONTEND_DIR", "./frontend/dist"),
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
	if c.MetricRawRetention < time.Hour || c.MetricRawRetention > 30*24*time.Hour {
		return Config{}, fmt.Errorf("PGSENTINEL_METRIC_RAW_RETENTION must be between 1h and 720h")
	}
	if c.MetricMediumRetention < c.MetricRawRetention || c.MetricMediumRetention > 180*24*time.Hour {
		return Config{}, fmt.Errorf("PGSENTINEL_METRIC_MEDIUM_RETENTION must be at least the raw retention and no more than 4320h")
	}
	if c.MetricLongRetention < c.MetricMediumRetention || c.MetricLongRetention > 5*365*24*time.Hour {
		return Config{}, fmt.Errorf("PGSENTINEL_METRIC_LONG_RETENTION must be at least the medium retention and no more than 43800h")
	}
	if c.MaxSnapshotsPerResource < 10 || c.MaxSnapshotsPerResource > 10000 {
		return Config{}, fmt.Errorf("PGSENTINEL_MAX_SNAPSHOTS_PER_RESOURCE must be between 10 and 10000")
	}
	if c.CollectorDiagnosticInterval < 10*time.Second || c.CollectorDiagnosticInterval > time.Hour {
		return Config{}, fmt.Errorf("PGSENTINEL_COLLECTOR_DIAGNOSTIC_INTERVAL must be between 10s and 1h")
	}
	if err := os.MkdirAll(c.DataDir, 0o750); err != nil {
		return Config{}, fmt.Errorf("create data directory: %w", err)
	}
	return c, nil
}

func boolean(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s must be a valid boolean: %w", key, err)
	}
	return parsed, nil
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
func strictDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return d, nil
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
