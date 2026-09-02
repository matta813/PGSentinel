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
	Fast, Standard, Slow, Metadata, Retention                      time.Duration
	MetricRawRetention, MetricMediumRetention, MetricLongRetention time.Duration
	MaxSnapshotsPerResource                                        int
	FanoutLimit                                                    int
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
	if s.MetricRawRetention <= 0 {
		s.MetricRawRetention = 24 * time.Hour
	}
	if s.MetricMediumRetention < s.MetricRawRetention {
		s.MetricMediumRetention = 30 * 24 * time.Hour
	}
	if s.MetricLongRetention < s.MetricMediumRetention {
		s.MetricLongRetention = 365 * 24 * time.Hour
	}
	if s.FanoutLimit <= 0 {
		s.FanoutLimit = 32
	}
	if s.MaxSnapshotsPerResource < 10 {
		s.MaxSnapshotsPerResource = 120
	}
	return s
}

func (m *Manager) Run(ctx context.Context) {
	m.wg.Add(1)
	defer m.wg.Done()
	if err := m.prune(ctx, time.Now()); err != nil {
		m.log.Warn("prune monitoring history at startup", "error", err)
	}
	m.collectAll(ctx, cycleAll)
	var workers sync.WaitGroup
	for _, worker := range []struct {
		interval time.Duration
		cycle    collectionCycle
	}{{m.schedule.Fast, cycleFast}, {m.schedule.Standard, cycleStandard}, {m.schedule.Slow, cycleSlow}, {m.schedule.Metadata, cycleMetadata}} {
		workers.Add(1)
		go func(interval time.Duration, cycle collectionCycle) {
			defer workers.Done()
			m.runCollectionSchedule(ctx, interval, cycle)
		}(worker.interval, worker.cycle)
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if err := m.prune(ctx, now); err != nil {
					m.log.Warn("prune monitoring history", "error", err)
				}
			}
		}
	}()
	workers.Wait()
}

func (m *Manager) runCollectionSchedule(ctx context.Context, interval time.Duration, cycle collectionCycle) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectAll(ctx, cycle)
		}
	}
}

func (m *Manager) prune(ctx context.Context, now time.Time) error {
	return m.store.PruneMonitoringHistory(ctx, now, m.schedule.Retention, m.schedule.MaxSnapshotsPerResource, storage.MetricRetentionPolicy{
		Raw: m.schedule.MetricRawRetention, Medium: m.schedule.MetricMediumRetention, Long: m.schedule.MetricLongRetention,
	})
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
		m.recordCycleUnavailable(ctx, server.ID, cycle)
		return
	}
	defer client.Close()
	collector := NewCore(client.Pool())
	snapshot := models.Snapshot{ServerID: server.ID, CollectedAt: time.Now().UTC(), Capabilities: map[string]bool{}}
	snapshot.ServerTags = append([]string(nil), server.Tags...)
	coreFresh := cycle&(cycleFast|cycleStandard) != 0
	if coreFresh {
		snapshot, err = collector.Collect(ctx, server.ID)
		if err != nil {
			m.recordCollectionFailure(ctx, server, err)
			m.recordCycleUnavailable(ctx, server.ID, cycle)
			return
		}
		m.recordFresh(ctx, server.ID, "connections", m.schedule.Fast, snapshot.CollectedAt)
		m.recordFresh(ctx, server.ID, "database-statistics", m.schedule.Fast, snapshot.CollectedAt)
		if cycle&cycleStandard == 0 {
			restoreCapabilities(ctx, m.store, server.ID, &snapshot)
		}
	} else if err := m.store.LatestSnapshot(ctx, server.ID, "core", &snapshot); err != nil {
		m.log.Debug("waiting for initial core snapshot", "server_id", server.ID)
		return
	}
	if cycle&cycleStandard != 0 {
		var previousReplication models.ReplicationStats
		var previousWAL models.WALStats
		_ = m.store.LatestSnapshot(ctx, server.ID, "replication", &previousReplication)
		_ = m.store.LatestSnapshot(ctx, server.ID, "wal", &previousWAL)
		replication, replicationErr := collector.CollectReplication(ctx)
		if replicationErr == nil {
			snapshot.Replication = replication
			if !m.saveSnapshot(ctx, server.ID, "replication", replication, snapshot.CollectedAt, m.schedule.Standard) {
				complete = false
			}
		} else {
			m.log.Warn("collect replication", "server_id", server.ID, "error", replicationErr)
			_ = m.restoreSnapshot(ctx, server.ID, "replication", &snapshot.Replication)
			complete = false
			m.recordUnavailable(ctx, server.ID, "replication", m.schedule.Standard, snapshot.CollectedAt)
		}
		wal, walErr := collector.CollectWAL(ctx)
		if walErr == nil {
			if replicationErr == nil {
				analyzer.EnrichWritePath(previousWAL, previousReplication, &wal, &snapshot.Replication)
			}
			snapshot.WAL = wal
			if !m.saveSnapshot(ctx, server.ID, "wal", wal, snapshot.CollectedAt, m.schedule.Standard) {
				complete = false
			}
		} else {
			m.log.Warn("collect WAL statistics", "server_id", server.ID, "error", walErr)
			_ = m.restoreSnapshot(ctx, server.ID, "wal", &snapshot.WAL)
			complete = false
			m.recordUnavailable(ctx, server.ID, "wal", m.schedule.Standard, snapshot.CollectedAt)
		}
		if replicationErr == nil {
			if err := m.store.SaveSnapshot(ctx, server.ID, "replication", snapshot.Replication, snapshot.CollectedAt); err != nil {
				complete = false
				m.log.Warn("save enriched replication snapshot", "server_id", server.ID, "error", err)
			}
		}
		queries, available, statsReset, postmasterStart, queryErr := collector.CollectQueries(ctx)
		snapshot.Capabilities["pg_stat_statements"] = available
		if queryErr == nil {
			history, historyErr := m.store.RecentQueryObservations(ctx, server.ID, 10)
			if historyErr == nil {
				current := models.QueryObservation{CollectedAt: snapshot.CollectedAt, StatsResetAt: statsReset, PostmasterStartAt: postmasterStart, Queries: queries}
				var changes []models.ChangeEvent
				if len(history) > 0 {
					changes, _ = m.store.ListChangeEvents(ctx, server.ID, history[0].CollectedAt, current.CollectedAt, 50)
				}
				regressionFindings = analyzer.QueryRegressionFindings(server.ID, history, current, changes)
				preserved, preserveErr := m.store.OpenFindingsByRule(ctx, server.ID, "query-regression")
				if preserveErr != nil {
					complete = false
				} else {
					active := map[string]bool{}
					for _, finding := range regressionFindings {
						active[finding.Fingerprint] = true
					}
					for _, finding := range preserved {
						if !active[finding.Fingerprint] && !analyzer.QueryRegressionFindingReady(finding, history, current) {
							regressionFindings = append(regressionFindings, finding)
						}
					}
				}
				if saveErr := m.store.SaveSnapshot(ctx, server.ID, "query-regression", current, snapshot.CollectedAt); saveErr != nil {
					complete = false
					m.log.Warn("save query regression observation", "server_id", server.ID, "error", saveErr)
				}
			} else {
				complete = false
				m.log.Warn("load query regression history", "server_id", server.ID, "error", historyErr)
			}
			snapshot.Queries = queries
			if !m.saveSnapshot(ctx, server.ID, "queries", queries, snapshot.CollectedAt, m.schedule.Standard) {
				complete = false
			}
		} else {
			m.log.Warn("collect queries", "server_id", server.ID, "error", queryErr)
			_ = m.restoreSnapshot(ctx, server.ID, "queries", &snapshot.Queries)
			complete = false
			m.recordUnavailable(ctx, server.ID, "queries", m.schedule.Standard, snapshot.CollectedAt)
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "replication", &snapshot.Replication)
		_ = m.store.LatestSnapshot(ctx, server.ID, "wal", &snapshot.WAL)
		_ = m.store.LatestSnapshot(ctx, server.ID, "queries", &snapshot.Queries)
	}
	if cycle&cycleFast != 0 {
		waitEvents, waitEventErr := collector.CollectWaitEvents(ctx)
		if waitEventErr == nil {
			snapshot.WaitEvents = waitEvents
			if saveErr := m.store.SaveSnapshot(ctx, server.ID, "wait-events", snapshot.WaitEvents, snapshot.CollectedAt); saveErr == nil {
				m.recordFresh(ctx, server.ID, "wait-events", m.schedule.Fast, snapshot.CollectedAt)
			} else {
				m.log.Warn("save wait-event snapshot", "server_id", server.ID, "error", saveErr)
				complete = false
				m.recordUnavailable(ctx, server.ID, "wait-events", m.schedule.Fast, snapshot.CollectedAt)
			}
		} else {
			m.log.Warn("collect wait events", "server_id", server.ID, "error", waitEventErr)
			_ = m.restoreSnapshot(ctx, server.ID, "wait-events", &snapshot.WaitEvents)
			complete = false
			m.recordUnavailable(ctx, server.ID, "wait-events", m.schedule.Fast, snapshot.CollectedAt)
		}
		locks, lockErr := collector.CollectLocks(ctx)
		if lockErr == nil {
			snapshot.Locks = locks
			if !m.saveSnapshot(ctx, server.ID, "locks", snapshot.Locks, snapshot.CollectedAt, m.schedule.Fast) {
				complete = false
			}
		} else {
			m.log.Warn("collect locks", "server_id", server.ID, "error", lockErr)
			_ = m.restoreSnapshot(ctx, server.ID, "locks", &snapshot.Locks)
			complete = false
			m.recordUnavailable(ctx, server.ID, "locks", m.schedule.Fast, snapshot.CollectedAt)
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "wait-events", &snapshot.WaitEvents)
		_ = m.store.LatestSnapshot(ctx, server.ID, "locks", &snapshot.Locks)
	}
	if cycle&cycleSlow != 0 {
		tables, indexes, collectionComplete := m.collectPerDatabase(ctx, target, snapshot.Databases)
		if collectionComplete {
			snapshot.Tables, snapshot.Indexes = tables, indexes
			tablesSaved := m.saveSnapshot(ctx, server.ID, "tables", snapshot.Tables, snapshot.CollectedAt, m.schedule.Slow)
			indexesSaved := m.saveSnapshot(ctx, server.ID, "indexes", snapshot.Indexes, snapshot.CollectedAt, m.schedule.Slow)
			if tablesSaved && indexesSaved {
				m.recordFresh(ctx, server.ID, "vacuum", m.schedule.Slow, snapshot.CollectedAt)
			} else {
				complete = false
				m.recordUnavailable(ctx, server.ID, "vacuum", m.schedule.Slow, snapshot.CollectedAt)
			}
		} else {
			_ = m.restoreSnapshot(ctx, server.ID, "tables", &snapshot.Tables)
			_ = m.restoreSnapshot(ctx, server.ID, "indexes", &snapshot.Indexes)
			complete = false
			for _, resource := range []string{"tables", "indexes", "vacuum"} {
				m.recordPartial(ctx, server.ID, resource, m.schedule.Slow, snapshot.CollectedAt)
			}
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "tables", &snapshot.Tables)
		_ = m.store.LatestSnapshot(ctx, server.ID, "indexes", &snapshot.Indexes)
	}
	if cycle&cycleMetadata != 0 {
		previousSettings := map[string]string{}
		_ = m.store.LatestSnapshot(ctx, server.ID, "configuration", &previousSettings)
		settings, settingsErr := collector.CollectConfiguration(ctx)
		if settingsErr == nil {
			snapshot.Settings = settings
			if details := changedSettings(previousSettings, settings); len(previousSettings) > 0 && len(details) > 0 {
				event := models.ChangeEvent{ServerID: server.ID, Kind: "configuration", Summary: "Monitored PostgreSQL settings changed", Details: details, OccurredAt: snapshot.CollectedAt}
				if err := m.store.RecordChangeEvent(ctx, &event); err != nil {
					m.log.Warn("record configuration change", "server_id", server.ID, "error", err)
				}
			}
			if !m.saveSnapshot(ctx, server.ID, "configuration", snapshot.Settings, snapshot.CollectedAt, m.schedule.Metadata) {
				complete = false
			}
		} else {
			m.log.Warn("collect configuration", "server_id", server.ID, "error", settingsErr)
			_ = m.restoreSnapshot(ctx, server.ID, "configuration", &snapshot.Settings)
			complete = false
			m.recordUnavailable(ctx, server.ID, "configuration", m.schedule.Metadata, snapshot.CollectedAt)
		}
	} else {
		_ = m.store.LatestSnapshot(ctx, server.ID, "configuration", &snapshot.Settings)
	}
	if coreFresh {
		if err := m.store.SaveSnapshot(ctx, server.ID, "core", snapshot, snapshot.CollectedAt); err != nil {
			complete = false
			m.log.Warn("save core snapshot", "server_id", server.ID, "error", err)
		}
		if err := m.store.SaveMetrics(ctx, snapshotMetrics(snapshot)); err != nil {
			complete = false
			m.log.Warn("save metrics", "server_id", server.ID, "error", err)
		}
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

func (m *Manager) saveSnapshot(ctx context.Context, serverID, resource string, value any, at time.Time, interval time.Duration) bool {
	if err := m.store.SaveSnapshot(ctx, serverID, resource, value, at); err != nil {
		m.log.Warn("save snapshot", "server_id", serverID, "resource", resource, "error", err)
		m.recordUnavailable(ctx, serverID, resource, interval, at)
		return false
	}
	m.recordFresh(ctx, serverID, resource, interval, at)
	return true
}

func changedSettings(before, after map[string]string) []string {
	keys := make([]string, 0)
	for key, value := range after {
		if old, ok := before[key]; ok && old != value {
			keys = append(keys, key+": "+old+" → "+value)
		}
	}
	sort.Strings(keys)
	if len(keys) > 50 {
		keys = keys[:50]
	}
	return keys
}

func (m *Manager) recordFresh(ctx context.Context, serverID, resource string, interval time.Duration, at time.Time) {
	if err := m.store.RecordCollectionResource(ctx, serverID, resource, "fresh", interval, at, ""); err != nil {
		m.log.Warn("record resource freshness", "server_id", serverID, "resource", resource, "error", err)
	}
}

func (m *Manager) recordUnavailable(ctx context.Context, serverID, resource string, interval time.Duration, at time.Time) {
	if err := m.store.RecordCollectionResource(ctx, serverID, resource, "unavailable", interval, at, "Collection failed; the last successful evidence is preserved."); err != nil {
		m.log.Warn("record resource freshness", "server_id", serverID, "resource", resource, "error", err)
	}
}

func (m *Manager) recordPartial(ctx context.Context, serverID, resource string, interval time.Duration, at time.Time) {
	if err := m.store.RecordCollectionResource(ctx, serverID, resource, "partial", interval, at, "Some databases could not be collected; cached evidence is preserved."); err != nil {
		m.log.Warn("record resource freshness", "server_id", serverID, "resource", resource, "error", err)
	}
}

func (m *Manager) recordCycleUnavailable(ctx context.Context, serverID string, cycle collectionCycle) {
	now := time.Now().UTC()
	if cycle&(cycleFast|cycleStandard) != 0 {
		m.recordUnavailable(ctx, serverID, "connections", m.schedule.Fast, now)
		m.recordUnavailable(ctx, serverID, "database-statistics", m.schedule.Fast, now)
	}
	if cycle&cycleFast != 0 {
		m.recordUnavailable(ctx, serverID, "locks", m.schedule.Fast, now)
		m.recordUnavailable(ctx, serverID, "wait-events", m.schedule.Fast, now)
	}
	if cycle&cycleStandard != 0 {
		for _, resource := range []string{"queries", "replication", "wal"} {
			m.recordUnavailable(ctx, serverID, resource, m.schedule.Standard, now)
		}
	}
	if cycle&cycleSlow != 0 {
		for _, resource := range []string{"tables", "indexes", "vacuum"} {
			m.recordUnavailable(ctx, serverID, resource, m.schedule.Slow, now)
		}
	}
	if cycle&cycleMetadata != 0 {
		m.recordUnavailable(ctx, serverID, "configuration", m.schedule.Metadata, now)
	}
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
	if err := m.store.UpsertFindings(ctx, serverID, findings); err != nil {
		return err
	}
	return m.store.RebuildIncidents(ctx, serverID, time.Now().UTC())
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
	maxReplayLag, maxReplayBytes, retainedSlotBytes := 0.0, 0.0, 0.0
	for _, standby := range s.Replication.Standbys {
		if standby.ReplayLagSeconds > maxReplayLag {
			maxReplayLag = standby.ReplayLagSeconds
		}
		if standby.PendingReplayBytes > maxReplayBytes {
			maxReplayBytes = standby.PendingReplayBytes
		}
	}
	for _, slot := range s.Replication.Slots {
		retainedSlotBytes += slot.RetainedBytes
	}
	values := map[string]float64{
		"connections.active":                   float64(s.Connections.Active),
		"connections.total":                    float64(s.Connections.Total),
		"connections.utilization":              s.Connections.Utilization,
		"connections.waiting":                  float64(s.Connections.Waiting),
		"server.uptime_seconds":                s.UptimeSeconds,
		"wal.generation_bytes_per_second":      s.WAL.GenerationBytesPerSecond,
		"wal.bytes_total":                      s.WAL.WALBytes,
		"replication.max_replay_lag_seconds":   maxReplayLag,
		"replication.max_pending_replay_bytes": maxReplayBytes,
		"replication.slot_retained_bytes":      retainedSlotBytes,
	}
	out := make([]models.Metric, 0, len(values))
	for name, value := range values {
		out = append(out, models.Metric{ServerID: s.ServerID, Name: name, Value: value, CollectedAt: s.CollectedAt})
	}
	return out
}
