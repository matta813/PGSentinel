package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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
	query := `SELECT server_id,database_name,name,value,labels_json,collected_at FROM (SELECT id,server_id,database_name,name,value,labels_json,collected_at FROM metrics WHERE server_id=? AND name=?`
	args := []any{serverID, name}
	if !from.IsZero() {
		query += ` AND collected_at>=?`
		args = append(args, from.UTC().Format(time.RFC3339Nano))
	}
	query += ` ORDER BY collected_at DESC,id DESC LIMIT ?) ORDER BY collected_at ASC,id ASC`
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	aggregates, err := s.listMetricAggregates(ctx, serverID, name, from, limit)
	if err != nil {
		return nil, err
	}
	out = append(out, aggregates...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].CollectedAt.Before(out[j].CollectedAt) })
	if len(out) > limit {
		if len(aggregates) == 0 {
			out = out[len(out)-limit:]
		} else {
			out = evenlySpacedMetrics(out, limit)
		}
	}
	return out, nil
}

func (s *Store) listMetricAggregates(ctx context.Context, serverID, name string, from time.Time, limit int) ([]models.Metric, error) {
	out := make([]models.Metric, 0)
	for _, tier := range []string{"medium", "long"} {
		query := `SELECT server_id,database_name,name,labels_json,bucket_start,minimum,maximum,value_sum,sample_count
			FROM metric_aggregates a WHERE server_id=? AND name=? AND tier=?`
		args := []any{serverID, name, tier}
		if tier == "long" {
			query += ` AND bucket_start < COALESCE((SELECT MIN(bucket_start) FROM metric_aggregates m WHERE m.server_id=a.server_id AND m.name=a.name AND m.tier='medium'),'9999-12-31T23:59:59Z')`
		}
		if !from.IsZero() {
			query += ` AND bucket_start>=?`
			args = append(args, from.UTC().Format(time.RFC3339Nano))
		}
		query += ` ORDER BY bucket_start DESC LIMIT ?`
		args = append(args, limit)
		rows, err := s.DB.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var metric models.Metric
			var labels, collected string
			var minimum, maximum, sum float64
			if err := rows.Scan(&metric.ServerID, &metric.Database, &metric.Name, &labels, &collected, &minimum, &maximum, &sum, &metric.Samples); err != nil {
				rows.Close()
				return nil, err
			}
			_ = json.Unmarshal([]byte(labels), &metric.Labels)
			metric.CollectedAt, _ = time.Parse(time.RFC3339Nano, collected)
			metric.Value = sum / float64(metric.Samples)
			metric.Minimum, metric.Maximum = &minimum, &maximum
			if tier == "medium" {
				metric.Resolution = "15m"
			} else {
				metric.Resolution = "6h"
			}
			out = append(out, metric)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func evenlySpacedMetrics(points []models.Metric, limit int) []models.Metric {
	if len(points) <= limit {
		return points
	}
	if limit == 1 {
		return points[len(points)-1:]
	}
	selected := make([]models.Metric, 0, limit)
	for i := 0; i < limit; i++ {
		index := i * (len(points) - 1) / (limit - 1)
		selected = append(selected, points[index])
	}
	return selected
}

type MetricRetentionPolicy struct {
	Raw, Medium, Long time.Duration
}

func (p MetricRetentionPolicy) validate() error {
	if p.Raw < time.Hour || p.Medium < p.Raw || p.Long < p.Medium {
		return fmt.Errorf("invalid metric retention policy")
	}
	return nil
}

// PruneMonitoringHistory rolls expiring raw metrics into fixed aggregate tiers
// before deleting them. The transaction makes cleanup safe to retry: either all
// aggregate upserts and deletes commit, or the raw samples remain untouched.
func (s *Store) PruneMonitoringHistory(ctx context.Context, now time.Time, snapshotRetention time.Duration, maxSnapshotsPerResource int, policy MetricRetentionPolicy) error {
	if snapshotRetention <= 0 {
		return fmt.Errorf("snapshot retention must be positive")
	}
	if maxSnapshotsPerResource < 10 {
		return fmt.Errorf("snapshot sample limit must be at least 10")
	}
	if err := policy.validate(); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rawCutoff := now.UTC().Add(-policy.Raw).Format(time.RFC3339Nano)
	longCutoff := now.UTC().Add(-policy.Long).Format(time.RFC3339Nano)
	for _, tier := range []struct {
		name    string
		seconds int64
	}{{"medium", 15 * 60}, {"long", 6 * 60 * 60}} {
		statement := `INSERT INTO metric_aggregates(tier,server_id,database_name,name,labels_json,bucket_start,minimum,maximum,value_sum,sample_count)
			SELECT ?,server_id,database_name,name,labels_json,strftime('%Y-%m-%dT%H:%M:%SZ',(unixepoch(collected_at)/?)*?,'unixepoch'),MIN(value),MAX(value),SUM(value),COUNT(*)
			FROM metrics WHERE collected_at < ? AND collected_at >= ?
			GROUP BY server_id,database_name,name,labels_json,(unixepoch(collected_at)/?)
			ON CONFLICT(tier,server_id,database_name,name,labels_json,bucket_start) DO UPDATE SET
			 minimum=MIN(minimum,excluded.minimum),maximum=MAX(maximum,excluded.maximum),value_sum=value_sum+excluded.value_sum,sample_count=sample_count+excluded.sample_count`
		if _, err := tx.ExecContext(ctx, statement, tier.name, tier.seconds, tier.seconds, rawCutoff, longCutoff, tier.seconds); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM metrics WHERE collected_at < ?`, rawCutoff); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM metric_aggregates WHERE tier='medium' AND bucket_start < ?`, now.UTC().Add(-policy.Medium).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM metric_aggregates WHERE tier='long' AND bucket_start < ?`, longCutoff); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM snapshots WHERE collected_at < ?`, now.UTC().Add(-snapshotRetention).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM snapshots WHERE id IN (
		SELECT id FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY server_id,kind ORDER BY collected_at DESC,id DESC) AS sample_number
			FROM snapshots
		) WHERE sample_number > ?
	)`, maxSnapshotsPerResource); err != nil {
		return err
	}
	return tx.Commit()
}
