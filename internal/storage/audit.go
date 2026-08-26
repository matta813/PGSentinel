package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
)

const (
	auditRetention = 365 * 24 * time.Hour
	auditMaxEvents = 100000
)

type AuditFilter struct {
	Actor, Action, ResourceType, Search string
	From, To                            time.Time
	Limit, Offset                       int
}

func (s *Store) RecordAuditEvent(ctx context.Context, event *models.AuditEvent) error {
	if event == nil || strings.TrimSpace(event.Actor) == "" || strings.TrimSpace(event.Action) == "" || strings.TrimSpace(event.ResourceType) == "" || strings.TrimSpace(event.Summary) == "" || len(event.Actor) > 100 || len(event.Action) > 100 || len(event.ResourceType) > 100 || len(event.ResourceID) > 200 || len(event.Summary) > 500 {
		return fmt.Errorf("invalid audit event")
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_events(id,occurred_at,actor,action,resource_type,resource_id,summary) VALUES(?,?,?,?,?,?,?)`,
		event.ID, event.OccurredAt.UTC().Format(time.RFC3339Nano), event.Actor, event.Action, event.ResourceType, event.ResourceID, event.Summary); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM audit_events WHERE occurred_at < ?`, event.OccurredAt.UTC().Add(-auditRetention).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM audit_events WHERE id IN (SELECT id FROM audit_events ORDER BY occurred_at DESC,id DESC LIMIT -1 OFFSET ?)`, auditMaxEvents); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListAuditEvents(ctx context.Context, filter AuditFilter) ([]models.AuditEvent, error) {
	if filter.Limit < 1 || filter.Limit > 100 || filter.Offset < 0 || filter.Offset > 10000 {
		return nil, fmt.Errorf("invalid audit pagination")
	}
	query := `SELECT id,occurred_at,actor,action,resource_type,resource_id,summary FROM audit_events WHERE 1=1`
	args := make([]any, 0, 10)
	for _, item := range []struct{ column, value string }{{"actor", filter.Actor}, {"action", filter.Action}, {"resource_type", filter.ResourceType}} {
		if item.value != "" {
			query += " AND " + item.column + "=?"
			args = append(args, item.value)
		}
	}
	if !filter.From.IsZero() {
		query += ` AND occurred_at>=?`
		args = append(args, filter.From.UTC().Format(time.RFC3339Nano))
	}
	if !filter.To.IsZero() {
		query += ` AND occurred_at<=?`
		args = append(args, filter.To.UTC().Format(time.RFC3339Nano))
	}
	if filter.Search != "" {
		query += ` AND (actor LIKE ? ESCAPE '\' OR action LIKE ? ESCAPE '\' OR resource_type LIKE ? ESCAPE '\' OR resource_id LIKE ? ESCAPE '\' OR summary LIKE ? ESCAPE '\')`
		pattern := "%" + escapeLike(filter.Search) + "%"
		args = append(args, pattern, pattern, pattern, pattern, pattern)
	}
	query += ` ORDER BY occurred_at DESC,id DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.AuditEvent, 0)
	for rows.Next() {
		var event models.AuditEvent
		var occurred string
		if err := rows.Scan(&event.ID, &occurred, &event.Actor, &event.Action, &event.ResourceType, &event.ResourceID, &event.Summary); err != nil {
			return nil, err
		}
		event.OccurredAt, _ = time.Parse(time.RFC3339Nano, occurred)
		out = append(out, event)
	}
	return out, rows.Err()
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
