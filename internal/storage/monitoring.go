package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	"strings"
	"time"
)

func (s *Store) SaveSnapshot(ctx context.Context, serverID, kind string, value any, at time.Time) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO snapshots(server_id,kind,payload_json,collected_at) VALUES(?,?,?,?)`, serverID, kind, string(body), at.Format(time.RFC3339Nano))
	return err
}
func (s *Store) LatestSnapshot(ctx context.Context, serverID, kind string, dst any) error {
	var body string
	err := s.DB.QueryRowContext(ctx, `SELECT payload_json FROM snapshots WHERE server_id=? AND kind=? ORDER BY collected_at DESC LIMIT 1`, serverID, kind).Scan(&body)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(body), dst)
}
func (s *Store) RecentQuerySnapshots(ctx context.Context, serverID string, limit int) ([][]models.QueryStat, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT payload_json FROM snapshots WHERE server_id=? AND kind='queries' ORDER BY collected_at DESC LIMIT ?`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var newestFirst [][]models.QueryStat
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return nil, err
		}
		var sample []models.QueryStat
		if err := json.Unmarshal([]byte(body), &sample); err != nil {
			return nil, err
		}
		newestFirst = append(newestFirst, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for left, right := 0, len(newestFirst)-1; left < right; left, right = left+1, right-1 {
		newestFirst[left], newestFirst[right] = newestFirst[right], newestFirst[left]
	}
	return newestFirst, nil
}
func (s *Store) UpsertFindings(ctx context.Context, serverID string, findings []models.Finding) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seen := map[string]bool{}
	for _, f := range findings {
		var previousStatus, previousSeverity string
		lookupErr := tx.QueryRowContext(ctx, `SELECT status,severity FROM findings WHERE fingerprint=?`, f.Fingerprint).Scan(&previousStatus, &previousSeverity)
		if lookupErr != nil && lookupErr != sql.ErrNoRows {
			return lookupErr
		}
		seen[f.Fingerprint] = true
		evidence, _ := json.Marshal(f.Evidence)
		suggestions, _ := json.Marshal(f.Suggestions)
		_, err = tx.ExecContext(ctx, `INSERT INTO findings(id,rule_id,fingerprint,server_id,database_name,resource,severity,category,title,summary,cause,impact,evidence_json,suggestions_json,confidence,status,started_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(fingerprint) DO UPDATE SET severity=excluded.severity,title=excluded.title,summary=excluded.summary,cause=excluded.cause,impact=excluded.impact,evidence_json=excluded.evidence_json,suggestions_json=excluded.suggestions_json,confidence=excluded.confidence,status=CASE WHEN findings.status='acknowledged' THEN 'acknowledged' ELSE 'active' END,updated_at=excluded.updated_at,resolved_at=NULL`, f.ID, f.RuleID, f.Fingerprint, f.ServerID, f.Database, f.Resource, f.Severity, f.Category, f.Title, f.Summary, f.Cause, f.Impact, string(evidence), string(suggestions), f.Confidence, "active", f.StartedAt.Format(time.RFC3339Nano), f.UpdatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		eventType := ""
		if lookupErr == sql.ErrNoRows {
			eventType = "new"
		} else if lookupErr == nil && previousStatus == "resolved" {
			eventType = "reopened"
		} else if lookupErr == nil && severityRank(string(f.Severity)) > severityRank(previousSeverity) {
			eventType = "severity_increased"
		}
		if eventType != "" {
			if err := queueFindingNotification(ctx, tx, f, eventType); err != nil {
				return err
			}
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT fingerprint FROM findings WHERE server_id=? AND status IN ('active','acknowledged')`, serverID)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var fp string
		if err = rows.Scan(&fp); err != nil {
			rows.Close()
			return err
		}
		if !seen[fp] {
			stale = append(stale, fp)
		}
	}
	rows.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, fp := range stale {
		var finding models.Finding
		var evidence, suggestions, started, updated string
		if err = tx.QueryRowContext(ctx, `SELECT id,rule_id,fingerprint,server_id,database_name,resource,severity,category,title,summary,cause,impact,evidence_json,suggestions_json,confidence,status,started_at,updated_at FROM findings WHERE fingerprint=?`, fp).Scan(&finding.ID, &finding.RuleID, &finding.Fingerprint, &finding.ServerID, &finding.Database, &finding.Resource, &finding.Severity, &finding.Category, &finding.Title, &finding.Summary, &finding.Cause, &finding.Impact, &evidence, &suggestions, &finding.Confidence, &finding.Status, &started, &updated); err != nil {
			return err
		}
		_ = json.Unmarshal([]byte(evidence), &finding.Evidence)
		_ = json.Unmarshal([]byte(suggestions), &finding.Suggestions)
		finding.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
		finding.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		finding.Status = "resolved"
		finding.UpdatedAt, _ = time.Parse(time.RFC3339Nano, now)
		if _, err = tx.ExecContext(ctx, `UPDATE findings SET status='resolved',resolved_at=?,updated_at=? WHERE fingerprint=?`, now, now, fp); err != nil {
			return err
		}
		if err := queueFindingNotification(ctx, tx, finding, "resolved"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func severityRank(value string) int {
	return map[string]int{"INFO": 1, "LOW": 2, "MEDIUM": 3, "HIGH": 4, "CRITICAL": 5}[value]
}

func queueFindingNotification(ctx context.Context, tx *sql.Tx, finding models.Finding, eventType string) error {
	if severityRank(string(finding.Severity)) < severityRank(string(models.SeverityHigh)) {
		return nil
	}
	body, err := json.Marshal(finding)
	if err != nil {
		return err
	}
	eventID := uuid.NewString()
	if _, err = tx.ExecContext(ctx, `INSERT INTO finding_notification_events(id,finding_id,event_type,finding_json,created_at) VALUES(?,?,?,?,?)`, eventID, finding.ID, eventType, string(body), time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO finding_notification_deliveries(event_id,destination_id) SELECT ?,id FROM notification_configs WHERE enabled=1`, eventID)
	return err
}

type FindingFilter struct {
	Status, ServerID, Severity, Category, Search string
}

func (s *Store) ListFindings(ctx context.Context, status, serverID string) ([]models.Finding, error) {
	return s.FilterFindings(ctx, FindingFilter{Status: status, ServerID: serverID})
}

func (s *Store) FilterFindings(ctx context.Context, filter FindingFilter) ([]models.Finding, error) {
	q := `SELECT id,rule_id,fingerprint,server_id,database_name,resource,severity,category,title,summary,cause,impact,evidence_json,suggestions_json,confidence,status,started_at,updated_at,resolved_at FROM findings WHERE 1=1`
	args := []any{}
	if filter.Status == "open" {
		q += " AND status IN ('active','acknowledged')"
	} else if filter.Status != "" {
		q += " AND status=?"
		args = append(args, filter.Status)
	}
	if filter.ServerID != "" {
		q += " AND server_id=?"
		args = append(args, filter.ServerID)
	}
	if filter.Severity != "" {
		q += " AND severity=?"
		args = append(args, filter.Severity)
	}
	if filter.Category != "" {
		q += " AND LOWER(category)=LOWER(?)"
		args = append(args, filter.Category)
	}
	if filter.Search != "" {
		q += " AND LOWER(title || ' ' || summary || ' ' || cause || ' ' || impact || ' ' || database_name || ' ' || resource) LIKE ?"
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
	}
	q += " ORDER BY CASE severity WHEN 'CRITICAL' THEN 0 WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 WHEN 'LOW' THEN 3 ELSE 4 END, updated_at DESC"
	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Finding{}
	for rows.Next() {
		var f models.Finding
		var ev, su, started, updated string
		var resolved sql.NullString
		if err := rows.Scan(&f.ID, &f.RuleID, &f.Fingerprint, &f.ServerID, &f.Database, &f.Resource, &f.Severity, &f.Category, &f.Title, &f.Summary, &f.Cause, &f.Impact, &ev, &su, &f.Confidence, &f.Status, &started, &updated, &resolved); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(ev), &f.Evidence)
		json.Unmarshal([]byte(su), &f.Suggestions)
		f.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
		f.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		if resolved.Valid {
			v, _ := time.Parse(time.RFC3339Nano, resolved.String)
			f.ResolvedAt = &v
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *Store) SetFindingStatus(ctx context.Context, id, status string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE findings SET status=?,updated_at=? WHERE id=? AND status IN ('active','acknowledged')`, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (s *Store) Prune(ctx context.Context, before time.Time) error {
	cutoff := before.Format(time.RFC3339Nano)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM snapshots WHERE collected_at < ?`, cutoff); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM metrics WHERE collected_at < ?`, cutoff); err != nil {
		return err
	}
	return tx.Commit()
}
