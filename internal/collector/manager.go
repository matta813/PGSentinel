package collector

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/notifications"
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
	store      *storage.Store
	engine     *analyzer.Engine
	log        *slog.Logger
	schedule   Schedule
	wg         sync.WaitGroup
	dispatcher *notifications.Dispatcher
}

func (m *Manager) SetNotificationDispatcher(dispatcher *notifications.Dispatcher) {
	m.dispatcher = dispatcher
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
	complete := true
	var regressionFindings []models.Finding
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
			m.recordCollectionFailure(ctx, server, err)
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
		replication, replicationErr := collector.CollectReplication(ctx)
		if replicationErr == nil {
			snapshot.Replication = replication
			_ = m.store.SaveSnapshot(ctx, server.ID, "replication", replication, snapshot.CollectedAt)
		} else {
			m.log.Warn("collect replication", "server_id", server.ID, "error", replicationErr)
			_ = m.restoreSnapshot(ctx, server.ID, "replication", &snapshot.Replication)
			complete = false
		}
		wal, walErr := collector.CollectWAL(ctx)
		if walErr == nil {
			snapshot.WAL = wal
			_ = m.store.SaveSnapshot(ctx, server.ID, "wal", wal, snapshot.CollectedAt)
		} else {
			m.log.Warn("collect WAL statistics", "server_id", server.ID, "error", walErr)
			_ = m.restoreSnapshot(ctx, server.ID, "wal", &snapshot.WAL)
			complete = false
		}
		queries, available, queryErr := collector.CollectQueries(ctx)
		snapshot.Capabilities["pg_stat_statements"] = available
		if queryErr == nil {
			history, historyErr := m.store.RecentQuerySnapshots(ctx, server.ID, 7)
			if historyErr == nil {
				regressionFindings = analyzer.QueryRegressionFindings(server.ID, history, queries)
			}
			snapshot.Queries = queries
			_ = m.store.SaveSnapshot(ctx, server.ID, "queries", queries, snapshot.CollectedAt)
		} else {
			m.log.Warn("collect queries", "server_id", server.ID, "error", queryErr)
			_ = m.restoreSnapshot(ctx, server.ID, "queries", &snapshot.Queries)
			complete = false
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "replication", &snapshot.Replication)
		_ = m.store.LatestSnapshot(ctx, server.ID, "wal", &snapshot.WAL)
		_ = m.store.LatestSnapshot(ctx, server.ID, "queries", &snapshot.Queries)
	}
	if cycle&cycleFast != 0 {
		locks, lockErr := collector.CollectLocks(ctx)
		if lockErr == nil {
			snapshot.Locks = locks
			_ = m.store.SaveSnapshot(ctx, server.ID, "locks", snapshot.Locks, snapshot.CollectedAt)
		} else {
			m.log.Warn("collect locks", "server_id", server.ID, "error", lockErr)
			_ = m.restoreSnapshot(ctx, server.ID, "locks", &snapshot.Locks)
			complete = false
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "locks", &snapshot.Locks)
	}
	if cycle&cycleSlow != 0 {
		tables, indexes, collectionComplete := m.collectPerDatabase(ctx, target, snapshot.Databases)
		if collectionComplete {
			snapshot.Tables, snapshot.Indexes = tables, indexes
			_ = m.store.SaveSnapshot(ctx, server.ID, "tables", snapshot.Tables, snapshot.CollectedAt)
			_ = m.store.SaveSnapshot(ctx, server.ID, "indexes", snapshot.Indexes, snapshot.CollectedAt)
		} else {
			_ = m.restoreSnapshot(ctx, server.ID, "tables", &snapshot.Tables)
			_ = m.restoreSnapshot(ctx, server.ID, "indexes", &snapshot.Indexes)
			complete = false
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "tables", &snapshot.Tables)
		_ = m.store.LatestSnapshot(ctx, server.ID, "indexes", &snapshot.Indexes)
	}
	if cycle&cycleMetadata != 0 {
		settings, settingsErr := collector.CollectConfiguration(ctx)
		if settingsErr == nil {
			snapshot.Settings = settings
			_ = m.store.SaveSnapshot(ctx, server.ID, "configuration", snapshot.Settings, snapshot.CollectedAt)
		} else {
			m.log.Warn("collect configuration", "server_id", server.ID, "error", settingsErr)
			_ = m.restoreSnapshot(ctx, server.ID, "configuration", &snapshot.Settings)
			complete = false
		}
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
	engine := m.engine
	if overrides, overrideErr := m.store.EffectiveThresholdOverrides(ctx, server); overrideErr == nil {
		engine = analyzer.New(analyzer.ApplyThresholdOverrides(analyzer.DefaultThresholds(), overrides))
	} else {
		complete = false
		m.log.Warn("resolve analyzer thresholds", "server_id", server.ID, "error", overrideErr)
	}
	findings := engine.Analyze(snapshot)
	findings = append(findings, regressionFindings...)
	findings = append(findings, analyzer.IndexFindings(server.ID, snapshot.Indexes)...)
	if err := m.reconcileFindings(ctx, server.ID, findings, complete); err != nil {
		m.log.Warn("reconcile findings", "server_id", server.ID, "error", err)
	}
	if m.dispatcher != nil {
		m.dispatcher.DispatchPending(ctx)
	}
	status, lastError := collectionOutcome(complete)
	_ = m.store.UpdateServerStatus(ctx, server.ID, status, snapshot.Version, lastError, true)
	m.log.Info("monitoring cycle complete", "server_id", server.ID, "cycle", cycle, "findings", len(findings))
}

func collectionOutcome(complete bool) (string, string) {
	if complete {
		return "healthy", ""
	}
	return "degraded", "One or more collector sections failed; cached evidence is being preserved."
}

func (m *Manager) recordCollectionFailure(ctx context.Context, server models.Server, collectionErr error) {
	m.log.Warn("core collection failed", "server_id", server.ID, "error", collectionErr)
	if err := m.store.UpdateServerStatus(ctx, server.ID, "error", server.Version, collectionErr.Error(), false); err != nil {
		m.log.Warn("update server collection status", "server_id", server.ID, "error", err)
	}
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

func (m *Manager) collectPerDatabase(ctx context.Context, server models.Server, dbStats []models.DatabaseStat) (tables []models.TableStat, indexes []models.IndexStat, complete bool) {
	complete = true
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
			complete = false
			m.log.Warn("per-database collection failed", "server_id", server.ID, "database", db.Name, "error", err)
			continue
		}
		core := NewCore(client.Pool())
		dbTables, err := core.CollectTables(ctx, db.Name)
		if err != nil {
			complete = false
			m.log.Warn("collect tables", "server_id", server.ID, "database", db.Name, "error", err)
		} else {
			tables = append(tables, dbTables...)
		}
		dbIndexes, err := core.CollectIndexes(ctx)
		if err != nil {
			complete = false
			m.log.Warn("collect indexes", "server_id", server.ID, "database", db.Name, "error", err)
		} else {
			indexes = append(indexes, dbIndexes...)
		}
		client.Close()
	}
	return
}

func (m *Manager) reconcileFindings(ctx context.Context, serverID string, findings []models.Finding, complete bool) error {
	if !complete {
		return nil
	}
	return m.store.UpsertFindings(ctx, serverID, findings)
}

func (m *Manager) restoreSnapshot(ctx context.Context, serverID, kind string, value any) bool {
	err := m.store.LatestSnapshot(ctx, serverID, kind, value)
	if err == nil {
		return true
	}
	if errors.Is(err, sql.ErrNoRows) {
		m.log.Warn("collector section unavailable and has no cached snapshot", "server_id", serverID, "kind", kind)
		return false
	}
	m.log.Warn("restore last complete snapshot", "server_id", serverID, "kind", kind, "error", err)
	return false
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
