package collector

import (
	"context"
	"time"

	"github.com/matta813/pgsentinel/internal/storage"
)

func (m *Manager) prune(ctx context.Context, now time.Time) error {
	return m.store.PruneMonitoringHistory(ctx, now, m.schedule.Retention, m.schedule.MaxSnapshotsPerResource, storage.MetricRetentionPolicy{
		Raw: m.schedule.MetricRawRetention, Medium: m.schedule.MetricMediumRetention, Long: m.schedule.MetricLongRetention,
	})
}
