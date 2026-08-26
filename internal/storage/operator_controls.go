package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

type controlQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type maintenanceControl struct{ reason, serverID, tag, category, rule string }
type suppressionControl struct{ reason, findingID, rule, serverID, tag string }

func (s *Store) ApplyOperatorControls(ctx context.Context, findings []models.Finding, now time.Time) error {
	windows, suppressions, err := loadActiveControls(ctx, s.DB, now)
	if err != nil {
		return err
	}
	tagsByServer := map[string][]string{}
	for index := range findings {
		tags, ok := tagsByServer[findings[index].ServerID]
		if !ok {
			var raw string
			if err := s.DB.QueryRowContext(ctx, `SELECT tags_json FROM servers WHERE id=?`, findings[index].ServerID).Scan(&raw); err != nil {
				return err
			}
			_ = json.Unmarshal([]byte(raw), &tags)
			tagsByServer[findings[index].ServerID] = tags
		}
		maintenance, suppressed, reason := matchControls(findings[index], tags, windows, suppressions)
		findings[index].Maintenance = maintenance
		findings[index].Suppressed = maintenance || suppressed
		findings[index].SuppressionReason = reason
	}
	return nil
}

func findingControl(ctx context.Context, q controlQueryer, finding models.Finding, now time.Time) (bool, bool, string, error) {
	var rawTags string
	if err := q.QueryRowContext(ctx, `SELECT tags_json FROM servers WHERE id=?`, finding.ServerID).Scan(&rawTags); err != nil {
		return false, false, "", err
	}
	var tags []string
	_ = json.Unmarshal([]byte(rawTags), &tags)
	windows, suppressions, err := loadActiveControls(ctx, q, now)
	if err != nil {
		return false, false, "", err
	}
	maintenance, suppressed, reason := matchControls(finding, tags, windows, suppressions)
	return maintenance, suppressed, reason, nil
}

func loadActiveControls(ctx context.Context, q controlQueryer, now time.Time) ([]maintenanceControl, []suppressionControl, error) {
	windows := []maintenanceControl{}
	rows, err := q.QueryContext(ctx, `SELECT description,COALESCE(server_id,''),server_tag,category,rule_id FROM maintenance_windows WHERE starts_at<=? AND ends_at>? ORDER BY starts_at,id LIMIT 200`, stamp(now), stamp(now))
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var item maintenanceControl
		if err := rows.Scan(&item.reason, &item.serverID, &item.tag, &item.category, &item.rule); err != nil {
			rows.Close()
			return nil, nil, err
		}
		windows = append(windows, item)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	suppressions := []suppressionControl{}
	rows, err = q.QueryContext(ctx, `SELECT reason,COALESCE(finding_id,''),rule_id,COALESCE(server_id,''),server_tag FROM finding_suppressions WHERE expires_at>? ORDER BY expires_at,id LIMIT 200`, stamp(now))
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var item suppressionControl
		if err := rows.Scan(&item.reason, &item.findingID, &item.rule, &item.serverID, &item.tag); err != nil {
			rows.Close()
			return nil, nil, err
		}
		suppressions = append(suppressions, item)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	return windows, suppressions, nil
}

func matchControls(finding models.Finding, tags []string, windows []maintenanceControl, suppressions []suppressionControl) (bool, bool, string) {
	matchesTag := func(want string) bool {
		if want == "" {
			return true
		}
		for _, tag := range tags {
			if strings.EqualFold(tag, want) {
				return true
			}
		}
		return false
	}
	for _, item := range windows {
		if (item.serverID == "" || item.serverID == finding.ServerID) && matchesTag(item.tag) && (item.category == "" || strings.EqualFold(item.category, finding.Category)) && (item.rule == "" || item.rule == finding.RuleID) {
			return true, false, "Maintenance: " + item.reason
		}
	}
	for _, item := range suppressions {
		if (item.findingID == "" || item.findingID == finding.ID) && (item.rule == "" || item.rule == finding.RuleID) && (item.serverID == "" || item.serverID == finding.ServerID) && matchesTag(item.tag) {
			return false, true, item.reason
		}
	}
	return false, false, ""
}

func (s *Store) ListMaintenanceWindows(ctx context.Context, limit int, now time.Time) ([]models.MaintenanceWindow, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,description,COALESCE(server_id,''),server_tag,category,rule_id,starts_at,ends_at,created_at,updated_at FROM maintenance_windows ORDER BY ends_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.MaintenanceWindow{}
	for rows.Next() {
		var item models.MaintenanceWindow
		var start, end, created, updated string
		if err := rows.Scan(&item.ID, &item.Description, &item.ServerID, &item.ServerTag, &item.Category, &item.RuleID, &start, &end, &created, &updated); err != nil {
			return nil, err
		}
		item.StartsAt, _ = time.Parse(time.RFC3339Nano, start)
		item.EndsAt, _ = time.Parse(time.RFC3339Nano, end)
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		item.State = temporalState(item.StartsAt, item.EndsAt, now)
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Store) CreateMaintenanceWindow(ctx context.Context, item *models.MaintenanceWindow) error {
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO maintenance_windows(id,starts_at,ends_at,description,server_id,server_tag,category,rule_id,created_at,updated_at) VALUES(?,?,?,?,NULLIF(?,''),?,?,?,?,?)`, item.ID, stamp(item.StartsAt), stamp(item.EndsAt), item.Description, item.ServerID, item.ServerTag, item.Category, item.RuleID, stamp(now), stamp(now))
	return err
}
func (s *Store) DeleteMaintenanceWindow(ctx context.Context, id string) error {
	return deleteControl(ctx, s.DB, "maintenance_windows", id)
}

func (s *Store) ListSuppressions(ctx context.Context, limit int, now time.Time) ([]models.FindingSuppression, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,COALESCE(finding_id,''),rule_id,COALESCE(server_id,''),server_tag,reason,expires_at,created_at,updated_at FROM finding_suppressions ORDER BY expires_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.FindingSuppression{}
	for rows.Next() {
		var item models.FindingSuppression
		var expires, created, updated string
		if err := rows.Scan(&item.ID, &item.FindingID, &item.RuleID, &item.ServerID, &item.ServerTag, &item.Reason, &expires, &created, &updated); err != nil {
			return nil, err
		}
		item.ExpiresAt, _ = time.Parse(time.RFC3339Nano, expires)
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		if now.Before(item.ExpiresAt) {
			item.State = "active"
		} else {
			item.State = "expired"
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Store) CreateSuppression(ctx context.Context, item *models.FindingSuppression) error {
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO finding_suppressions(id,finding_id,rule_id,server_id,server_tag,expires_at,reason,created_at,updated_at) VALUES(?,NULLIF(?,''),?,NULLIF(?,''),?,?,?,?,?)`, item.ID, item.FindingID, item.RuleID, item.ServerID, item.ServerTag, stamp(item.ExpiresAt), item.Reason, stamp(now), stamp(now))
	return err
}
func (s *Store) DeleteSuppression(ctx context.Context, id string) error {
	return deleteControl(ctx, s.DB, "finding_suppressions", id)
}

func (s *Store) ListThresholdOverrides(ctx context.Context, limit int) ([]models.ThresholdOverride, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,rule_id,scope_type,scope_value,value,reason,created_at,updated_at FROM threshold_overrides ORDER BY rule_id,CASE scope_type WHEN 'server' THEN 0 WHEN 'tag' THEN 1 ELSE 2 END,scope_value LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []models.ThresholdOverride{}
	for rows.Next() {
		var item models.ThresholdOverride
		var created, updated string
		if err := rows.Scan(&item.ID, &item.RuleID, &item.ScopeType, &item.ScopeValue, &item.Value, &item.Reason, &created, &updated); err != nil {
			return nil, err
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		item.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		items = append(items, item)
	}
	return items, rows.Err()
}
func (s *Store) CreateThresholdOverride(ctx context.Context, item *models.ThresholdOverride) error {
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO threshold_overrides(id,rule_id,scope_type,scope_value,value,reason,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, item.ID, item.RuleID, item.ScopeType, item.ScopeValue, item.Value, item.Reason, stamp(now), stamp(now))
	return err
}
func (s *Store) DeleteThresholdOverride(ctx context.Context, id string) error {
	return deleteControl(ctx, s.DB, "threshold_overrides", id)
}

func (s *Store) EffectiveThresholdOverrides(ctx context.Context, server models.Server) ([]models.ThresholdOverride, error) {
	items, err := s.ListThresholdOverrides(ctx, 200)
	if err != nil {
		return nil, err
	}
	selected := map[string]models.ThresholdOverride{}
	rank := map[string]int{}
	for _, item := range items {
		match, r := item.ScopeType == "global", 1
		if item.ScopeType == "tag" {
			for _, tag := range server.Tags {
				if strings.EqualFold(tag, item.ScopeValue) {
					match = true
					r = 2
					break
				}
			}
		}
		if item.ScopeType == "server" && item.ScopeValue == server.ID {
			match = true
			r = 3
		}
		if match && (r > rank[item.RuleID] || (r == rank[item.RuleID] && item.ScopeValue < selected[item.RuleID].ScopeValue)) {
			selected[item.RuleID] = item
			rank[item.RuleID] = r
		}
	}
	out := make([]models.ThresholdOverride, 0, len(selected))
	for _, item := range selected {
		out = append(out, item)
	}
	return out, nil
}

func temporalState(start, end, now time.Time) string {
	if now.Before(start) {
		return "upcoming"
	}
	if now.Before(end) {
		return "active"
	}
	return "expired"
}
func stamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func deleteControl(ctx context.Context, db *sql.DB, table, id string) error {
	allowed := map[string]bool{"maintenance_windows": true, "finding_suppressions": true, "threshold_overrides": true}
	if !allowed[table] {
		return sql.ErrNoRows
	}
	result, err := db.ExecContext(ctx, "DELETE FROM "+table+" WHERE id=?", id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return sql.ErrNoRows
	}
	return err
}
