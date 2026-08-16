package storage

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) SaveMetrics(ctx context.Context, metrics []models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, metric := range metrics {
		labels, err := json.Marshal(metric.Labels)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO metrics(server_id,database_name,name,value,labels_json,collected_at) VALUES(?,?,?,?,?,?)`, metric.ServerID, metric.Database, metric.Name, metric.Value, string(labels), metric.CollectedAt.UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListMetrics(ctx context.Context, serverID, name string, from time.Time, limit int) ([]models.Metric, error) {
	query := `SELECT server_id,database_name,name,value,labels_json,collected_at FROM metrics WHERE server_id=? AND name=?`
	args := []any{serverID, name}
	if !from.IsZero() {
		query += ` AND collected_at>=?`
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	query += ` ORDER BY collected_at ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Metric{}
	for rows.Next() {
		var metric models.Metric
		var labels, collected string
		if err := rows.Scan(&metric.ServerID, &metric.Database, &metric.Name, &metric.Value, &labels, &collected); err != nil {
			return nil, err
		}
		if strings.TrimSpace(labels) != "" {
			_ = json.Unmarshal([]byte(labels), &metric.Labels)
		}
		metric.CollectedAt, _ = time.Parse(time.RFC3339Nano, collected)
		out = append(out, metric)
	}
	return out, rows.Err()
}
