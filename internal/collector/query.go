package collector

import (
	"context"
	"github.com/matta813/pgsentinel/internal/models"
	"strings"
)

// ImpactScore balances cumulative load, latency, disk reads, temporary IO and WAL.
// Each component is log-scaled so one extreme counter cannot hide every other signal.
func ImpactScore(q models.QueryStat) float64 {
	log := func(v float64) float64 {
		if v <= 0 {
			return 0
		}
		n := 0.0
		for v >= 10 {
			v /= 10
			n++
		}
		return n + v/10
	}
	return log(q.TotalExecMS)*35 + log(q.MeanExecMS)*20 + log(q.SharedRead*8192)*20 + log((q.TempRead+q.TempWritten)*8192)*15 + log(q.WALBytes)*10
}
func (c *Core) CollectQueries(ctx context.Context) ([]models.QueryStat, bool, error) {
	var available bool
	if err := c.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_stat_statements')`).Scan(&available); err != nil || !available {
		return nil, available, err
	}
	rows, err := c.db.Query(ctx, `SELECT queryid::text,d.datname,query,calls,total_exec_time,mean_exec_time,min_exec_time,max_exec_time,rows,shared_blks_hit,shared_blks_read,temp_blks_read,temp_blks_written,COALESCE(wal_bytes,0) FROM pg_stat_statements p JOIN pg_database d ON d.oid=p.dbid ORDER BY total_exec_time DESC LIMIT 200`)
	if err != nil {
		return nil, true, err
	}
	defer rows.Close()
	out := []models.QueryStat{}
	for rows.Next() {
		var q models.QueryStat
		if err := rows.Scan(&q.QueryID, &q.Database, &q.Query, &q.Calls, &q.TotalExecMS, &q.MeanExecMS, &q.MinExecMS, &q.MaxExecMS, &q.Rows, &q.SharedHit, &q.SharedRead, &q.TempRead, &q.TempWritten, &q.WALBytes); err != nil {
			return nil, true, err
		}
		q.Query = strings.TrimSpace(q.Query)
		q.ImpactScore = ImpactScore(q)
		out = append(out, q)
	}
	return out, true, rows.Err()
}
