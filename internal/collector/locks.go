package collector

import (
	"context"
	"gitlab.scruzzi.com/root/postgresqlui/internal/models"
)

func (c *Core) CollectLocks(ctx context.Context) ([]models.LockInfo, error) {
	rows, err := c.db.Query(ctx, `SELECT blocked.pid,blocking.pid,EXTRACT(EPOCH FROM now()-blocked.query_start),blocked.datname,blocked.usename,blocked.application_name,blocked.query,blocking.query FROM pg_stat_activity blocked CROSS JOIN LATERAL unnest(pg_blocking_pids(blocked.pid)) blocker(pid) JOIN pg_stat_activity blocking ON blocking.pid=blocker.pid ORDER BY blocked.query_start`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.LockInfo{}
	for rows.Next() {
		var l models.LockInfo
		if err := rows.Scan(&l.BlockedPID, &l.BlockingPID, &l.DurationSeconds, &l.Database, &l.User, &l.Application, &l.Query, &l.BlockingQuery); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
