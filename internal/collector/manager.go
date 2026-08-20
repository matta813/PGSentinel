package collector

import (
	"context"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	pg "github.com/matta813/pgsentinel/internal/postgres"
	"github.com/matta813/pgsentinel/internal/storage"
)

type Schedule struct {
	Fast, Standard, Slow, Metadata, Retention time.Duration
	FanoutLimit                               int
}

type collectionCycle uint8

const (
	cycleFast collectionCycle = 1 << iota
	cycleStandard
	cycleSlow
	cycleMetadata
	cycleAll = cycleFast | cycleStandard | cycleSlow | cycleMetadata
)

type Manager struct {
	store    *storage.Store
	engine   *analyzer.Engine
	log      *slog.Logger
	schedule Schedule
	wg       sync.WaitGroup
}

func NewManager(store *storage.Store, log *slog.Logger, schedule Schedule) *Manager {
	return &Manager{store: store, engine: analyzer.New(analyzer.DefaultThresholds()), log: log, schedule: schedule.normalized()}
}

func (s Schedule) normalized() Schedule {
	if s.Standard <= 0 {
		s.Standard = 30 * time.Second
	}
	if s.Fast <= 0 {
		s.Fast = s.Standard
	}
	if s.Slow <= 0 {
		s.Slow = 5 * time.Minute
	}
	if s.Metadata <= 0 {
		s.Metadata = 30 * time.Minute
	}
	if s.Retention <= 0 {
		s.Retention = 30 * 24 * time.Hour
	}
	if s.FanoutLimit <= 0 {
		s.FanoutLimit = 32
	}
	return s
}

func (m *Manager) Run(ctx context.Context) {
	m.wg.Add(1)
	defer m.wg.Done()
	m.collectAll(ctx, cycleAll)
	fast := time.NewTicker(m.schedule.Fast)
	standard := time.NewTicker(m.schedule.Standard)
	slow := time.NewTicker(m.schedule.Slow)
	metadata := time.NewTicker(m.schedule.Metadata)
	prune := time.NewTicker(time.Hour)
	defer fast.Stop()
	defer standard.Stop()
	defer slow.Stop()
	defer metadata.Stop()
	defer prune.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-fast.C:
			m.collectAll(ctx, cycleFast)
		case <-standard.C:
			m.collectAll(ctx, cycleStandard)
		case <-slow.C:
			m.collectAll(ctx, cycleSlow)
		case <-metadata.C:
			m.collectAll(ctx, cycleMetadata)
		case <-prune.C:
			if err := m.prune(ctx, time.Now()); err != nil {
				m.log.Warn("prune monitoring history", "error", err)
			}
		}
	}
}

func (m *Manager) prune(ctx context.Context, now time.Time) error {
	return m.store.Prune(ctx, now.Add(-m.schedule.Retention))
}

func (m *Manager) collectAll(ctx context.Context, cycle collectionCycle) {
	servers, err := m.store.ListServers(ctx)
	if err != nil {
		m.log.Error("list monitoring targets", "error", err)
		return
	}
	for _, server := range servers {
		if ctx.Err() != nil {
			return
		}
		m.collect(ctx, server, cycle)
	}
}

func (m *Manager) collect(ctx context.Context, server models.Server, cycle collectionCycle) {
	target, err := m.store.GetServer(ctx, server.ID, true)
	if err != nil {
		return
	}
	client, err := pg.Connect(ctx, target)
	if err != nil {
		m.log.Warn("postgres collection failed", "server_id", server.ID, "error", err)
		_ = m.store.UpdateServerStatus(ctx, server.ID, "unreachable", server.Version, err.Error(), false)
		return
	}
	defer client.Close()
	collector := NewCore(client.Pool())
	snapshot := models.Snapshot{ServerID: server.ID, CollectedAt: time.Now().UTC(), Capabilities: map[string]bool{}}
	coreFresh := cycle&(cycleFast|cycleStandard) != 0
	if coreFresh {
		snapshot, err = collector.Collect(ctx, server.ID)
		if err != nil {
			m.log.Warn("core collection failed", "server_id", server.ID, "error", err)
			return
		}
		if cycle&cycleStandard == 0 {
			restoreCapabilities(ctx, m.store, server.ID, &snapshot)
		}
	} else if err := m.store.LatestSnapshot(ctx, server.ID, "core", &snapshot); err != nil {
		m.log.Debug("waiting for initial core snapshot", "server_id", server.ID)
		return
	}
	if cycle&cycleStandard != 0 {
		queries, available, queryErr := collector.CollectQueries(ctx)
		snapshot.Capabilities["pg_stat_statements"] = available
		if queryErr == nil {
			snapshot.Queries = queries
			_ = m.store.SaveSnapshot(ctx, server.ID, "queries", queries, snapshot.CollectedAt)
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "queries", &snapshot.Queries)
	}
	if cycle&cycleFast != 0 {
		snapshot.Locks, _ = collector.CollectLocks(ctx)
		_ = m.store.SaveSnapshot(ctx, server.ID, "locks", snapshot.Locks, snapshot.CollectedAt)
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "locks", &snapshot.Locks)
	}
	if cycle&cycleSlow != 0 {
		snapshot.Tables, snapshot.Indexes = m.collectPerDatabase(ctx, target, snapshot.Databases, snapshot.CollectedAt)
		_ = m.store.SaveSnapshot(ctx, server.ID, "tables", snapshot.Tables, snapshot.CollectedAt)
		_ = m.store.SaveSnapshot(ctx, server.ID, "indexes", snapshot.Indexes, snapshot.CollectedAt)
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "tables", &snapshot.Tables)
		_ = m.store.LatestSnapshot(ctx, server.ID, "indexes", &snapshot.Indexes)
	}
	if cycle&cycleMetadata != 0 {
		snapshot.Settings, _ = collector.CollectConfiguration(ctx)
		_ = m.store.SaveSnapshot(ctx, server.ID, "configuration", snapshot.Settings, snapshot.CollectedAt)
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "configuration", &snapshot.Settings)
	}
	if coreFresh {
		_ = m.store.SaveSnapshot(ctx, server.ID, "core", snapshot, snapshot.CollectedAt)
		_ = m.store.SaveMetrics(ctx, snapshotMetrics(snapshot))
	}
	if cycle == cycleMetadata {
		m.log.Info("metadata collection complete", "server_id", server.ID)
		return
	}
	findings := m.engine.Analyze(snapshot)
	findings = append(findings, analyzer.IndexFindings(server.ID, snapshot.Indexes)...)
	_ = m.store.UpsertFindings(ctx, server.ID, findings)
	_ = m.store.UpdateServerStatus(ctx, server.ID, "healthy", snapshot.Version, "", true)
	m.log.Info("monitoring cycle complete", "server_id", server.ID, "cycle", cycle, "findings", len(findings))
}

func collectibleDatabases(stats []models.DatabaseStat, limit int) []models.DatabaseStat {
	out := make([]models.DatabaseStat, 0, len(stats))
	for _, d := range stats {
		if d.Name == "" || d.Name == "template0" || d.Name == "template1" {
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SizeBytes > out[j].SizeBytes })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (m *Manager) collectPerDatabase(ctx context.Context, server models.Server, dbStats []models.DatabaseStat, at time.Time) (tables []models.TableStat, indexes []models.IndexStat) {
	targets := collectibleDatabases(dbStats, m.schedule.FanoutLimit)
	if len(targets) == 0 {
		targets = []models.DatabaseStat{{Name: "postgres"}}
	}
	for _, db := range targets {
		if ctx.Err() != nil {
			return
		}
		client, err := pg.ConnectDatabase(ctx, server, db.Name)
		if err != nil {
			m.log.Warn("per-database collection failed", "server_id", server.ID, "database", db.Name, "error", err)
			continue
		}
		core := NewCore(client.Pool())
		dbTables, err := core.CollectTables(ctx, db.Name)
		if err != nil {
			m.log.Warn("collect tables", "server_id", server.ID, "database", db.Name, "error", err)
		} else {
			tables = append(tables, dbTables...)
		}
		dbIndexes, err := core.CollectIndexes(ctx)
		if err != nil {
			m.log.Warn("collect indexes", "server_id", server.ID, "database", db.Name, "error", err)
		} else {
			indexes = append(indexes, dbIndexes...)
		}
		client.Close()
	}
	return
}

func restoreCapabilities(ctx context.Context, store *storage.Store, serverID string, snapshot *models.Snapshot) {
	var prev models.Snapshot
	if err := store.LatestSnapshot(ctx, serverID, "core", &prev); err != nil || prev.Capabilities == nil {
		return
	}
	if snapshot.Capabilities == nil {
		snapshot.Capabilities = map[string]bool{}
	}
	for name, ok := range prev.Capabilities {
		snapshot.Capabilities[name] = ok
	}
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
