package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) RecordCollectionResource(ctx context.Context, serverID, resource, state string, expected time.Duration, at time.Time, errorSummary string) error {
	seconds := int64(expected / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	stamp := at.UTC().Format(time.RFC3339Nano)
	var success any
	if state == "fresh" {
		success = stamp
	}
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO collection_resource_status(server_id,resource,state,last_attempt_at,last_success_at,expected_interval_seconds,consecutive_failures,error_summary)
		VALUES(?,?,?,?,?,?,CASE WHEN ?='fresh' THEN 0 ELSE 1 END,?)
		ON CONFLICT(server_id,resource) DO UPDATE SET
			state=excluded.state,last_attempt_at=excluded.last_attempt_at,
			last_success_at=COALESCE(excluded.last_success_at,collection_resource_status.last_success_at),
			expected_interval_seconds=excluded.expected_interval_seconds,
			consecutive_failures=CASE WHEN excluded.state='fresh' THEN 0 ELSE collection_resource_status.consecutive_failures+1 END,
			error_summary=excluded.error_summary`, serverID, resource, state, stamp, success, seconds, state, errorSummary)
	return err
}

func (s *Store) ListCollectionResources(ctx context.Context, serverID string, now time.Time) ([]models.CollectionResourceStatus, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT server_id,resource,state,last_attempt_at,last_success_at,expected_interval_seconds,consecutive_failures,error_summary FROM collection_resource_status WHERE server_id=? ORDER BY resource`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.CollectionResourceStatus{}
	for rows.Next() {
		var item models.CollectionResourceStatus
		var attempt string
		var success sql.NullString
		if err := rows.Scan(&item.ServerID, &item.Resource, &item.State, &attempt, &success, &item.ExpectedIntervalSeconds, &item.ConsecutiveFailures, &item.ErrorSummary); err != nil {
			return nil, err
		}
		item.LastAttemptAt, _ = time.Parse(time.RFC3339Nano, attempt)
		if success.Valid {
			parsed, parseErr := time.Parse(time.RFC3339Nano, success.String)
			if parseErr == nil {
				item.LastSuccessfulAt = &parsed
				item.CollectedAt = &parsed
				age := int64(now.UTC().Sub(parsed).Seconds())
				if age < 0 {
					age = 0
				}
				item.AgeSeconds = &age
				if item.State != "unavailable" && age > item.ExpectedIntervalSeconds*2 {
					item.State = "stale"
				}
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
