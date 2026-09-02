package collector

import (
	"context"
	"sync"
	"time"
)

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
