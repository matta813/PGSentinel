package collector

import (
	"context"
	"time"

	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/models"
	pg "github.com/matta813/pgsentinel/internal/postgres"
)

func (m *Manager) collect(ctx context.Context, server models.Server, cycle collectionCycle) {
	cycleStarted := time.Now()
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
		if replicationErr == nil && !m.saveSnapshot(ctx, server.ID, "replication", snapshot.Replication, snapshot.CollectedAt, m.schedule.Standard) {
			complete = false
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
		tables, indexes, tablesComplete, indexesComplete := m.collectPerDatabase(ctx, target, snapshot.Databases)
		var tablesFresh bool
		snapshot.Tables, tablesFresh = m.persistTableCollection(ctx, server.ID, tables, tablesComplete, snapshot.CollectedAt)
		if !tablesFresh {
			complete = false
		}
		if indexesComplete {
			snapshot.Indexes = indexes
			if !m.saveSnapshot(ctx, server.ID, "indexes", snapshot.Indexes, snapshot.CollectedAt, m.schedule.Slow) {
				complete = false
			}
		} else {
			_ = m.restoreSnapshot(ctx, server.ID, "indexes", &snapshot.Indexes)
			complete = false
			m.recordPartial(ctx, server.ID, "indexes", m.schedule.Slow, snapshot.CollectedAt)
		}
		if tablesFresh {
			m.recordFresh(ctx, server.ID, "vacuum", m.schedule.Slow, snapshot.CollectedAt)
		} else if !tablesComplete {
			m.recordPartial(ctx, server.ID, "vacuum", m.schedule.Slow, snapshot.CollectedAt)
		} else {
			m.recordUnavailable(ctx, server.ID, "vacuum", m.schedule.Slow, snapshot.CollectedAt)
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
	m.log.Info("monitoring cycle complete", "server_id", server.ID, "cycle", cycle, "findings", len(findings), "complete", complete, "duration", time.Since(cycleStarted))
}
