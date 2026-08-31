package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) RecordChangeEvent(ctx context.Context, event *models.ChangeEvent) error {
	if event == nil || !validChangeKind(event.Kind) || event.ServerID == "" || event.Summary == "" || len(event.Summary) > 300 || len(event.Details) > 50 {
		return fmt.Errorf("invalid change event")
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	event.CreatedAt = time.Now().UTC()
	details, err := json.Marshal(event.Details)
	if err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO change_events(id,server_id,kind,summary,details_json,occurred_at,created_at) VALUES(?,?,?,?,?,?,?)`, event.ID, event.ServerID, event.Kind, event.Summary, string(details), event.OccurredAt.UTC().Format(time.RFC3339Nano), event.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM change_events WHERE occurred_at<?`, event.CreatedAt.AddDate(-1, 0, 0).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM change_events WHERE id IN (SELECT id FROM change_events WHERE server_id=? ORDER BY occurred_at DESC,id DESC LIMIT -1 OFFSET 10000)`, event.ServerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListChangeEvents(ctx context.Context, serverID string, from, to time.Time, limit int) ([]models.ChangeEvent, error) {
	if serverID == "" || limit < 1 || limit > 200 {
		return nil, fmt.Errorf("invalid change event filter")
	}
	query := `SELECT id,server_id,kind,summary,details_json,occurred_at,created_at FROM change_events WHERE server_id=?`
	args := []any{serverID}
	if !from.IsZero() {
		query += ` AND occurred_at>=?`
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	if !to.IsZero() {
		query += ` AND occurred_at<=?`
		args = append(args, to.UTC().Format(time.RFC3339Nano))
	}
	query += ` ORDER BY occurred_at DESC,id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ChangeEvent{}
	for rows.Next() {
		var event models.ChangeEvent
		var details, occurred, created string
		if err := rows.Scan(&event.ID, &event.ServerID, &event.Kind, &event.Summary, &details, &occurred, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(details), &event.Details)
		event.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurred)
		event.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, event)
	}
	return out, rows.Err()
}

func (s *Store) DeleteChangeEvent(ctx context.Context, id string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM change_events WHERE id=? AND kind='deployment'`, id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func validChangeKind(kind string) bool { return kind == "deployment" || kind == "configuration" }
