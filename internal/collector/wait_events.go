package collector

import (
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/matta813/pgsentinel/internal/models"
)

const maxWaitEventQueryBytes = 2000

const waitEventsSQL = `SELECT
  pid,
  COALESCE(datname, ''),
  COALESCE(usename, ''),
  COALESCE(application_name, ''),
  COALESCE(client_addr::text, ''),
  COALESCE(backend_type, ''),
  COALESCE(state, ''),
  COALESCE(wait_event_type, ''),
  COALESCE(wait_event, ''),
  LEFT(COALESCE(query, ''), 2000),
  query_start,
  xact_start,
  state_change,
  COALESCE(EXTRACT(EPOCH FROM clock_timestamp() - query_start), 0),
  COALESCE(EXTRACT(EPOCH FROM clock_timestamp() - xact_start), 0),
  COALESCE(EXTRACT(EPOCH FROM clock_timestamp() - state_change), 0)
FROM pg_stat_activity
WHERE wait_event IS NOT NULL
  AND pid <> pg_backend_pid()
ORDER BY query_start NULLS LAST, pid
LIMIT 500`

// CollectWaitEvents captures a bounded, cluster-wide view of backends currently
// reporting a PostgreSQL wait event. The ages are query, transaction, and state
// ages; PostgreSQL does not expose the time at which the current wait began.
func (c *Core) CollectWaitEvents(ctx context.Context) ([]models.WaitEventSample, error) {
	rows, err := c.db.Query(ctx, waitEventsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.WaitEventSample{}
	for rows.Next() {
		var sample models.WaitEventSample
		var queryStart, transactionStart, stateChange sql.NullTime
		if err := rows.Scan(
			&sample.PID, &sample.Database, &sample.User, &sample.Application,
			&sample.ClientAddress, &sample.BackendType, &sample.State,
			&sample.WaitEventType, &sample.WaitEvent, &sample.Query,
			&queryStart, &transactionStart, &stateChange,
			&sample.QueryAgeSeconds, &sample.TransactionAgeSeconds, &sample.StateAgeSeconds,
		); err != nil {
			return nil, err
		}
		sample.Query = boundWaitEventQuery(sample.Query)
		if queryStart.Valid {
			sample.QueryStartedAt = &queryStart.Time
		}
		if transactionStart.Valid {
			sample.TransactionStartedAt = &transactionStart.Time
		}
		if stateChange.Valid {
			sample.StateChangedAt = &stateChange.Time
		}
		out = append(out, sample)
	}
	return out, rows.Err()
}

func boundWaitEventQuery(query string) string {
	query = strings.TrimSpace(strings.ToValidUTF8(query, "�"))
	if len(query) <= maxWaitEventQueryBytes {
		return query
	}
	query = query[:maxWaitEventQueryBytes]
	for len(query) > 0 && !utf8.ValidString(query) {
		query = query[:len(query)-1]
	}
	return query
}
