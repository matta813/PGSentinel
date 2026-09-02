package collector

import (
	"context"
	"sort"

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

func (m *Manager) collectPerDatabase(ctx context.Context, server models.Server, databaseStats []models.DatabaseStat) (tables []models.TableStat, indexes []models.IndexStat, complete bool) {
	complete = true
	targets := collectibleDatabases(databaseStats, m.schedule.FanoutLimit)
	if len(targets) == 0 {
		targets = []models.DatabaseStat{{Name: "postgres"}}
	}
	for _, database := range targets {
		if ctx.Err() != nil {
			return
		}
		client, err := pg.ConnectDatabase(ctx, server, database.Name)
		if err != nil {
			complete = false
			m.log.Warn("per-database collection failed", "server_id", server.ID, "database", database.Name, "error", err)
			continue
		}
		core := NewCore(client.Pool())
		databaseTables, err := core.CollectTables(ctx, database.Name)
		if err != nil {
			complete = false
			m.log.Warn("collect tables", "server_id", server.ID, "database", database.Name, "error", err)
		} else {
			tables = append(tables, databaseTables...)
		}
		databaseIndexes, err := core.CollectIndexes(ctx)
		if err != nil {
			complete = false
			m.log.Warn("collect indexes", "server_id", server.ID, "database", database.Name, "error", err)
		} else {
			indexes = append(indexes, databaseIndexes...)
		}
		client.Close()
	}
	return
}
