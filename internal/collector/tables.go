package collector

import (
	"context"
	"fmt"

	"github.com/matta813/pgsentinel/internal/models"
)

const tablesSQL = `SELECT current_database(),s.schemaname,s.relname,c.reltuples,pg_total_relation_size(c.oid),pg_relation_size(c.oid),pg_indexes_size(c.oid),s.n_live_tup,s.n_dead_tup,s.seq_scan,COALESCE(s.idx_scan,0),s.n_tup_ins,s.n_tup_upd,s.n_tup_del,s.n_tup_hot_upd,s.last_vacuum,s.last_autovacuum,s.last_analyze,s.last_autoanalyze,s.vacuum_count,s.autovacuum_count,COALESCE((SELECT option_value::float8 FROM pg_options_to_table(c.reloptions) WHERE option_name='autovacuum_vacuum_threshold'),current_setting('autovacuum_vacuum_threshold')::float8)+COALESCE((SELECT option_value::float8 FROM pg_options_to_table(c.reloptions) WHERE option_name='autovacuum_vacuum_scale_factor'),current_setting('autovacuum_vacuum_scale_factor')::float8)*GREATEST(c.reltuples,0) FROM pg_stat_user_tables s JOIN pg_class c ON c.oid=s.relid ORDER BY pg_total_relation_size(c.oid) DESC LIMIT 500`

func (c *Core) CollectTables(ctx context.Context, database string) ([]models.TableStat, error) {
	rows, err := c.db.Query(ctx, tablesSQL)
	if err != nil {
		return nil, fmt.Errorf("collect tables for %s: %w", database, err)
	}
	defer rows.Close()
	out := []models.TableStat{}
	for rows.Next() {
		var t models.TableStat
		if err := rows.Scan(&t.Database, &t.Schema, &t.Table, &t.EstimatedRows, &t.TotalSize, &t.TableSize, &t.IndexSize, &t.LiveTuples, &t.DeadTuples, &t.SeqScans, &t.IndexScans, &t.Inserts, &t.Updates, &t.Deletes, &t.HotUpdates, &t.LastVacuum, &t.LastAutovacuum, &t.LastAnalyze, &t.LastAutoanalyze, &t.VacuumCount, &t.AutovacuumCount, &t.VacuumThreshold); err != nil {
			return nil, err
		}
		if t.VacuumThreshold > 0 {
			t.VacuumProgress = t.DeadTuples / t.VacuumThreshold * 100
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
