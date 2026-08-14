package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/matta813/pgsentinel/internal/models"
)

type Core struct{ db *pgxpool.Pool }

const connectionsSQL = `
SELECT
  count(*) FILTER (WHERE state = 'active'),
  count(*) FILTER (WHERE state = 'idle'),
  count(*) FILTER (WHERE state = 'idle in transaction'),
  count(*) FILTER (WHERE wait_event IS NOT NULL),
  count(*),
  (SELECT setting::int FROM pg_settings WHERE name = 'max_connections'),
  COALESCE(EXTRACT(EPOCH FROM now() - (min(xact_start) FILTER (WHERE xact_start IS NOT NULL))), 0),
  COALESCE(EXTRACT(EPOCH FROM now() - (min(state_change) FILTER (WHERE state = 'idle in transaction'))), 0)
FROM pg_stat_activity`

func NewCore(db *pgxpool.Pool) *Core { return &Core{db: db} }
func (c *Core) Collect(ctx context.Context, serverID string) (models.Snapshot, error) {
	s := models.Snapshot{ServerID: serverID, CollectedAt: time.Now().UTC(), Settings: map[string]string{}, Capabilities: map[string]bool{}}
	if err := c.db.QueryRow(ctx, `SELECT current_setting('server_version'), EXTRACT(EPOCH FROM now()-pg_postmaster_start_time())`).Scan(&s.Version, &s.UptimeSeconds); err != nil {
		return s, err
	}
	if err := c.connections(ctx, &s); err != nil {
		return s, fmt.Errorf("connections: %w", err)
	}
	if err := c.databases(ctx, &s); err != nil {
		return s, fmt.Errorf("databases: %w", err)
	}
	return s, nil
}
func (c *Core) connections(ctx context.Context, s *models.Snapshot) error {
	return c.db.QueryRow(ctx, connectionsSQL).Scan(&s.Connections.Active, &s.Connections.Idle, &s.Connections.IdleInTransaction, &s.Connections.Waiting, &s.Connections.Total, &s.Connections.Max, &s.Connections.LongestTransactionSeconds, &s.Connections.LongestIdleTransactionSeconds)
}
func (c *Core) databases(ctx context.Context, s *models.Snapshot) error {
	rows, err := c.db.Query(ctx, `SELECT datname,pg_database_size(datname),xact_commit,xact_rollback,deadlocks,temp_files,temp_bytes,blks_read,blks_hit FROM pg_stat_database WHERE datname IS NOT NULL ORDER BY datname`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var d models.DatabaseStat
		if err := rows.Scan(&d.Name, &d.SizeBytes, &d.Commits, &d.Rollbacks, &d.Deadlocks, &d.TempFiles, &d.TempBytes, &d.BlocksRead, &d.BlocksHit); err != nil {
			return err
		}
		s.Databases = append(s.Databases, d)
	}
	if s.Connections.Max > 0 {
		s.Connections.Utilization = float64(s.Connections.Total) / float64(s.Connections.Max) * 100
	}
	return rows.Err()
}
