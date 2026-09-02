package collector

import (
	"context"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

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
