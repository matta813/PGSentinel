package collector

import (
	"context"
	"sort"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	pg "github.com/matta813/pgsentinel/internal/postgres"
)

func collectibleDatabases(stats []models.DatabaseStat, limit int) []models.DatabaseStat {
	out := make([]models.DatabaseStat, 0, len(stats))
	for _, database := range stats {
		if database.Name == "" || database.Name == "template0" || database.Name == "template1" {
			continue
		}
		out = append(out, database)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SizeBytes > out[j].SizeBytes })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (m *Manager) collectPerDatabase(ctx context.Context, server models.Server, databaseStats []models.DatabaseStat) (tables []models.TableStat, indexes []models.IndexStat, tablesComplete bool, indexesComplete bool) {
	tablesComplete, indexesComplete = true, true
	targets := collectibleDatabases(databaseStats, m.schedule.FanoutLimit)
	if len(targets) == 0 {
		targets = []models.DatabaseStat{{Name: "postgres"}}
	}
	for _, database := range targets {
		if ctx.Err() != nil {
			return
		}
		connectStarted := time.Now()
		client, err := pg.ConnectDatabase(ctx, server, database.Name)
		if err != nil {
			tablesComplete, indexesComplete = false, false
			duration := time.Since(connectStarted)
			m.diagnostics.failed(server.ID, database.Name, "tables", "connect", err, duration, time.Now())
			m.diagnostics.failed(server.ID, database.Name, "indexes", "connect", err, duration, time.Now())
			continue
		}
		core := NewCore(client.Pool())
		tableStarted := time.Now()
		databaseTables, err := core.CollectTables(ctx, database.Name)
		tableDuration := time.Since(tableStarted)
		if err != nil {
			tablesComplete = false
			m.diagnostics.failed(server.ID, database.Name, "tables", "tables", err, tableDuration, time.Now())
		} else {
			m.diagnostics.succeeded(server.ID, database.Name, "tables", "tables", tableDuration)
			tables = append(tables, databaseTables...)
		}
		indexStarted := time.Now()
		databaseIndexes, err := core.CollectIndexes(ctx)
		indexDuration := time.Since(indexStarted)
		if err != nil {
			indexesComplete = false
			m.diagnostics.failed(server.ID, database.Name, "indexes", "indexes", err, indexDuration, time.Now())
		} else {
			m.diagnostics.succeeded(server.ID, database.Name, "indexes", "indexes", indexDuration)
			indexes = append(indexes, databaseIndexes...)
		}
		client.Close()
	}
	return
}
