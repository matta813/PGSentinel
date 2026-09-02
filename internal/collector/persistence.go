package collector

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/internal/storage"
)

func (m *Manager) saveSnapshot(ctx context.Context, serverID, resource string, value any, at time.Time, interval time.Duration) bool {
	if err := m.store.SaveSnapshot(ctx, serverID, resource, value, at); err != nil {
		m.log.Warn("save snapshot", "server_id", serverID, "resource", resource, "error", err)
		m.recordUnavailable(ctx, serverID, resource, interval, at)
		return false
	}
	m.recordFresh(ctx, serverID, resource, interval, at)
	return true
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
	var previous models.Snapshot
	if err := store.LatestSnapshot(ctx, serverID, "core", &previous); err != nil || previous.Capabilities == nil {
		return
	}
	if snapshot.Capabilities == nil {
		snapshot.Capabilities = map[string]bool{}
	}
	for name, available := range previous.Capabilities {
		snapshot.Capabilities[name] = available
	}
}
