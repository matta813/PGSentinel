package collector

import (
	"context"
	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	pg "github.com/matta813/pgsentinel/internal/postgres"
	"github.com/matta813/pgsentinel/internal/storage"
	"log/slog"
	"sync"
	"time"
)

type Manager struct {
	store     *storage.Store
	engine    *analyzer.Engine
	log       *slog.Logger
	interval  time.Duration
	retention time.Duration
	wg        sync.WaitGroup
}

func NewManager(store *storage.Store, log *slog.Logger, interval, retention time.Duration) *Manager {
	return &Manager{store: store, engine: analyzer.New(analyzer.DefaultThresholds()), log: log, interval: interval, retention: retention}
}
func (m *Manager) Run(ctx context.Context) {
	m.wg.Add(1)
	defer m.wg.Done()
	m.collectAll(ctx)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	prune := time.NewTicker(time.Hour)
	defer prune.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectAll(ctx)
		case <-prune.C:
			if err := m.prune(ctx, time.Now()); err != nil {
				m.log.Warn("prune monitoring history", "error", err)
			}
		}
	}
}

func (m *Manager) prune(ctx context.Context, now time.Time) error {
	return m.store.Prune(ctx, now.Add(-m.retention))
}
func (m *Manager) collectAll(ctx context.Context) {
	servers, err := m.store.ListServers(ctx)
	if err != nil {
		m.log.Error("list monitoring targets", "error", err)
		return
	}
	for _, s := range servers {
		m.collect(ctx, s)
	}
}
func (m *Manager) collect(ctx context.Context, s models.Server) {
	target, err := m.store.GetServer(ctx, s.ID, true)
	if err != nil {
		return
	}
	client, err := pg.Connect(ctx, target)
	if err != nil {
		m.log.Warn("postgres collection failed", "server_id", s.ID, "error", err)
		_ = m.store.UpdateServerStatus(ctx, s.ID, "unreachable", s.Version, err.Error(), false)
		return
	}
	defer client.Close()
	c := NewCore(client.Pool())
	snap, err := c.Collect(ctx, s.ID)
	if err != nil {
		m.log.Warn("core collection failed", "server_id", s.ID, "error", err)
		return
	}
	queries, pgss, qerr := c.CollectQueries(ctx)
	snap.Capabilities["pg_stat_statements"] = pgss
	if qerr == nil {
		snap.Queries = queries
		_ = m.store.SaveSnapshot(ctx, s.ID, "queries", queries, snap.CollectedAt)
	}
	locks, _ := c.CollectLocks(ctx)
	snap.Locks = locks
	_ = m.store.SaveSnapshot(ctx, s.ID, "locks", locks, snap.CollectedAt)
	tables, _ := c.CollectTables(ctx, "postgres")
	snap.Tables = tables
	_ = m.store.SaveSnapshot(ctx, s.ID, "tables", tables, snap.CollectedAt)
	indexes, _ := c.CollectIndexes(ctx)
	snap.Indexes = indexes
	_ = m.store.SaveSnapshot(ctx, s.ID, "indexes", indexes, snap.CollectedAt)
	settings, _ := c.CollectConfiguration(ctx)
	snap.Settings = settings
	_ = m.store.SaveSnapshot(ctx, s.ID, "configuration", settings, snap.CollectedAt)
	_ = m.store.SaveSnapshot(ctx, s.ID, "core", snap, snap.CollectedAt)
	_ = m.store.SaveMetrics(ctx, snapshotMetrics(snap))
	findings := m.engine.Analyze(snap)
	findings = append(findings, analyzer.IndexFindings(s.ID, indexes)...)
	_ = m.store.UpsertFindings(ctx, s.ID, findings)
	_ = m.store.UpdateServerStatus(ctx, s.ID, "healthy", snap.Version, "", true)
	m.log.Info("monitoring cycle complete", "server_id", s.ID, "findings", len(findings))
}

func snapshotMetrics(s models.Snapshot) []models.Metric {
	values := map[string]float64{
		"connections.active":      float64(s.Connections.Active),
		"connections.total":       float64(s.Connections.Total),
		"connections.utilization": s.Connections.Utilization,
		"connections.waiting":     float64(s.Connections.Waiting),
		"server.uptime_seconds":   s.UptimeSeconds,
	}
	out := make([]models.Metric, 0, len(values))
	for name, value := range values {
		out = append(out, models.Metric{ServerID: s.ServerID, Name: name, Value: value, CollectedAt: s.CollectedAt})
	}
	return out
}
