package collector

import "context"

var monitoredSettings = []string{"shared_buffers", "work_mem", "maintenance_work_mem", "effective_cache_size", "max_connections", "checkpoint_timeout", "checkpoint_completion_target", "max_wal_size", "min_wal_size", "random_page_cost", "effective_io_concurrency", "autovacuum", "autovacuum_max_workers", "autovacuum_naptime", "autovacuum_vacuum_threshold", "autovacuum_vacuum_scale_factor", "track_io_timing", "shared_preload_libraries"}

func (c *Core) CollectConfiguration(ctx context.Context) (map[string]string, error) {
	rows, err := c.db.Query(ctx, `SELECT name,setting||COALESCE(unit,'') FROM pg_settings WHERE name=ANY($1)`, monitoredSettings)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
