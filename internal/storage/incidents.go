package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/incidents"
	"github.com/matta813/pgsentinel/internal/models"
)

const incidentFindingLimit = 500

type IncidentFilter struct {
	Status, ServerID string
	Limit, Offset    int
}

func (s *Store) RebuildIncidents(ctx context.Context, serverID string, now time.Time) error {
	findings, err := s.recentIncidentFindings(ctx, serverID, now.Add(-24*time.Hour))
	if err != nil {
		return err
	}
	candidates := incidents.Correlate(findings)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seen := make(map[string]bool, len(candidates))
	for _, incident := range candidates {
		seen[incident.ID] = true
		rationale, marshalErr := json.Marshal(incident.Rationale)
		if marshalErr != nil {
			return marshalErr
		}
		var resolved any
		if incident.ResolvedAt != nil {
			resolved = incident.ResolvedAt.UTC().Format(time.RFC3339Nano)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO incidents(id,server_id,title,summary,rationale_json,severity,status,started_at,updated_at,resolved_at)
			VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,summary=excluded.summary,rationale_json=excluded.rationale_json,
			severity=excluded.severity,status=excluded.status,updated_at=excluded.updated_at,resolved_at=excluded.resolved_at`,
			incident.ID, incident.ServerID, incident.Title, incident.Summary, string(rationale), incident.Severity, incident.Status,
			incident.StartedAt.UTC().Format(time.RFC3339Nano), incident.UpdatedAt.UTC().Format(time.RFC3339Nano), resolved); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM incident_findings WHERE incident_id=?`, incident.ID); err != nil {
			return err
		}
		relationship := strings.Join(incident.Rationale, "; ")
		for _, finding := range incident.Findings {
			if _, err := tx.ExecContext(ctx, `INSERT INTO incident_findings(incident_id,finding_id,relationship) VALUES(?,?,?)`, incident.ID, finding.ID, relationship); err != nil {
				return err
			}
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT id FROM incidents WHERE server_id=? AND status='active'`, serverID)
	if err != nil {
		return err
	}
	var stale []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		if !seen[id] {
			stale = append(stale, id)
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	resolvedAt := now.UTC().Format(time.RFC3339Nano)
	for _, id := range stale {
		if _, err := tx.ExecContext(ctx, `UPDATE incidents SET status='resolved',updated_at=?,resolved_at=? WHERE id=?`, resolvedAt, resolvedAt, id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM incidents WHERE status='resolved' AND updated_at < ?`, now.UTC().Add(-90*24*time.Hour).Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) recentIncidentFindings(ctx context.Context, serverID string, resolvedSince time.Time) ([]models.Finding, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,rule_id,fingerprint,server_id,database_name,resource,severity,category,title,summary,cause,impact,evidence_json,suggestions_json,confidence,status,started_at,updated_at,resolved_at
		FROM findings WHERE server_id=? AND (status IN ('active','acknowledged') OR resolved_at>=?) ORDER BY started_at DESC LIMIT ?`,
		serverID, resolvedSince.UTC().Format(time.RFC3339Nano), incidentFindingLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Finding, 0)
	for rows.Next() {
		finding, scanErr := scanIncidentFinding(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, finding)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(...any) error }

func scanIncidentFinding(row rowScanner) (models.Finding, error) {
	var finding models.Finding
	var evidence, suggestions, started, updated string
	var resolved sql.NullString
	err := row.Scan(&finding.ID, &finding.RuleID, &finding.Fingerprint, &finding.ServerID, &finding.Database, &finding.Resource,
		&finding.Severity, &finding.Category, &finding.Title, &finding.Summary, &finding.Cause, &finding.Impact, &evidence, &suggestions,
		&finding.Confidence, &finding.Status, &started, &updated, &resolved)
	if err != nil {
		return models.Finding{}, err
	}
	_ = json.Unmarshal([]byte(evidence), &finding.Evidence)
	_ = json.Unmarshal([]byte(suggestions), &finding.Suggestions)
	finding.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	finding.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if resolved.Valid {
		at, parseErr := time.Parse(time.RFC3339Nano, resolved.String)
		if parseErr == nil {
			finding.ResolvedAt = &at
		}
	}
	return finding, nil
}

func (s *Store) ListIncidents(ctx context.Context, filter IncidentFilter) ([]models.Incident, error) {
	if filter.Limit < 1 || filter.Limit > 100 || filter.Offset < 0 || filter.Offset > 10000 {
		return nil, fmt.Errorf("invalid incident pagination")
	}
	query := `SELECT id,server_id,title,summary,rationale_json,severity,status,started_at,updated_at,resolved_at FROM incidents WHERE 1=1`
	args := make([]any, 0, 5)
	if filter.Status != "" {
		query += ` AND status=?`
		args = append(args, filter.Status)
	}
	if filter.ServerID != "" {
		query += ` AND server_id=?`
		args = append(args, filter.ServerID)
	}
	query += ` ORDER BY CASE status WHEN 'active' THEN 0 ELSE 1 END,updated_at DESC,id LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]models.Incident, 0)
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, incident)
	}
	return out, rows.Err()
}

func (s *Store) GetIncident(ctx context.Context, id string) (models.Incident, error) {
	incident, err := scanIncident(s.DB.QueryRowContext(ctx, `SELECT id,server_id,title,summary,rationale_json,severity,status,started_at,updated_at,resolved_at FROM incidents WHERE id=?`, id))
	if err != nil {
		return models.Incident{}, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT f.id,f.rule_id,f.fingerprint,f.server_id,f.database_name,f.resource,f.severity,f.category,f.title,f.summary,f.cause,f.impact,f.evidence_json,f.suggestions_json,f.confidence,f.status,f.started_at,f.updated_at,f.resolved_at
		FROM incident_findings i JOIN findings f ON f.id=i.finding_id WHERE i.incident_id=? ORDER BY f.started_at,f.id`, id)
	if err != nil {
		return models.Incident{}, err
	}
	defer rows.Close()
	for rows.Next() {
		finding, scanErr := scanIncidentFinding(rows)
		if scanErr != nil {
			return models.Incident{}, scanErr
		}
		incident.Findings = append(incident.Findings, finding)
		incident.Timeline = append(incident.Timeline, models.IncidentEvent{At: finding.StartedAt, Type: "finding_started", FindingID: finding.ID, Title: finding.Title, Detail: "PGSentinel first observed this finding.", Severity: finding.Severity})
		if finding.ResolvedAt != nil {
			incident.Timeline = append(incident.Timeline, models.IncidentEvent{At: *finding.ResolvedAt, Type: "finding_resolved", FindingID: finding.ID, Title: finding.Title, Detail: "The finding evidence no longer crossed its rule threshold.", Severity: finding.Severity})
		}
	}
	if err := rows.Err(); err != nil {
		return models.Incident{}, err
	}
	sort.Slice(incident.Timeline, func(i, j int) bool {
		if incident.Timeline[i].At.Equal(incident.Timeline[j].At) {
			return incident.Timeline[i].FindingID < incident.Timeline[j].FindingID
		}
		return incident.Timeline[i].At.Before(incident.Timeline[j].At)
	})
	return incident, nil
}

func scanIncident(row rowScanner) (models.Incident, error) {
	var incident models.Incident
	var rationale, started, updated string
	var resolved sql.NullString
	err := row.Scan(&incident.ID, &incident.ServerID, &incident.Title, &incident.Summary, &rationale, &incident.Severity, &incident.Status, &started, &updated, &resolved)
	if err != nil {
		return models.Incident{}, err
	}
	_ = json.Unmarshal([]byte(rationale), &incident.Rationale)
	incident.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	incident.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if resolved.Valid {
		at, parseErr := time.Parse(time.RFC3339Nano, resolved.String)
		if parseErr == nil {
			incident.ResolvedAt = &at
		}
	}
	return incident, nil
}
