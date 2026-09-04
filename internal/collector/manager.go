package collector

import (
	"log/slog"
	"sync"
	"time"

	"github.com/matta813/pgsentinel/internal/analyzer"
	"github.com/matta813/pgsentinel/internal/notifications"
	"github.com/matta813/pgsentinel/internal/storage"
)

type Schedule struct {
	Fast, Standard, Slow, Metadata, Retention                      time.Duration
	DiagnosticInterval                                             time.Duration
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
	store       *storage.Store
	engine      *analyzer.Engine
	log         *slog.Logger
	schedule    Schedule
	wg          sync.WaitGroup
	dispatcher  *notifications.Dispatcher
	diagnostics *collectorDiagnostics
}

func (m *Manager) SetNotificationDispatcher(dispatcher *notifications.Dispatcher) {
	m.dispatcher = dispatcher
}

func NewManager(store *storage.Store, log *slog.Logger, schedule Schedule) *Manager {
	schedule = schedule.normalized()
	return &Manager{store: store, engine: analyzer.New(analyzer.DefaultThresholds()), log: log, schedule: schedule, diagnostics: newCollectorDiagnostics(log, schedule.DiagnosticInterval)}
}
