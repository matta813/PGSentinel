package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/matta813/pgsentinel/internal/models"
	"time"
)

func (s *Store) OpenFindingsByRule(ctx context.Context, serverID, ruleID string) ([]models.Finding, error) {
	return s.FilterFindings(ctx, FindingFilter{Status: "open", ServerID: serverID, RuleID: ruleID, Limit: 200})
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
