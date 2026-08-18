package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matta813/pgsentinel/internal/api"
	"github.com/matta813/pgsentinel/internal/auth"
	"github.com/matta813/pgsentinel/internal/buildinfo"
	"github.com/matta813/pgsentinel/internal/collector"
	"github.com/matta813/pgsentinel/internal/config"
	"github.com/matta813/pgsentinel/internal/notifications"
	"github.com/matta813/pgsentinel/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).With("service", "pgsentinel", "version", buildinfo.Version, "commit", buildinfo.CommitSHA)
	store, err := storage.Open(cfg.DatabasePath, cfg.EncryptionKey)
	if err != nil {
		log.Error("open storage", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	manager := collector.NewManager(store, log, collector.Schedule{Fast: cfg.FastInterval, Standard: cfg.StatsInterval, Slow: cfg.SlowInterval, Metadata: cfg.MetaInterval, Retention: cfg.Retention, FanoutLimit: cfg.FanoutDatabaseLimit})
	go manager.Run(ctx)
	authentication, err := auth.New(auth.Config{Password: cfg.AdminPassword, SecureCookies: cfg.SecureCookies, TrustedProxies: cfg.TrustedProxyCIDRs})
	if err != nil {
		log.Error("configure authentication", "error", err)
		os.Exit(1)
	}
	app := api.New(store, log, api.Options{
		Auth:               authentication,
		NotificationPolicy: notifications.NewTargetPolicy(cfg.AllowPrivateNotificationTargets, cfg.NotificationAllowedHosts),
	})
	app.ServeFrontend(cfg.FrontendDir)
	server := &http.Server{Addr: cfg.ListenAddr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Info("server listening", "address", cfg.ListenAddr)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("http server failed", "error", err)
		os.Exit(1)
	}
}
