package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/matta813/pgsentinel/internal/models"
	"strings"
	"time"
)

type FindingFilter struct {
	Status, ServerID, Database, Severity, Category, Search, RuleID string
	Limit                                                          int
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
	if filter.Database != "" {
		q += " AND database_name=?"
		args = append(args, filter.Database)
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
	if filter.RuleID != "" {
		q += " AND rule_id=?"
		args = append(args, filter.RuleID)
	}
	q += " ORDER BY CASE severity WHEN 'CRITICAL' THEN 0 WHEN 'HIGH' THEN 1 WHEN 'MEDIUM' THEN 2 WHEN 'LOW' THEN 3 ELSE 4 END, updated_at DESC"
	if filter.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, filter.Limit)
	}
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
