package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab.scruzzi.com/root/postgresqlui/internal/api"
	"gitlab.scruzzi.com/root/postgresqlui/internal/buildinfo"
	"gitlab.scruzzi.com/root/postgresqlui/internal/collector"
	"gitlab.scruzzi.com/root/postgresqlui/internal/config"
	"gitlab.scruzzi.com/root/postgresqlui/internal/storage"
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
	manager := collector.NewManager(store, log, cfg.StatsInterval)
	go manager.Run(ctx)
	app := api.New(store, log)
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
