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
func (s *Store) RecentQueryObservations(ctx context.Context, serverID string, limit int) ([]models.QueryObservation, error) {
	if limit < 1 {
		return []models.QueryObservation{}, nil
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT payload_json,collected_at FROM snapshots WHERE server_id=? AND kind='query-regression' ORDER BY collected_at DESC LIMIT ?`, serverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var newestFirst []models.QueryObservation
	for rows.Next() {
		var body, collected string
		if err := rows.Scan(&body, &collected); err != nil {
			return nil, err
		}
		var observation models.QueryObservation
		if err := json.Unmarshal([]byte(body), &observation); err != nil {
			return nil, err
		}
		if observation.CollectedAt.IsZero() {
			observation.CollectedAt, _ = time.Parse(time.RFC3339Nano, collected)
		}
		newestFirst = append(newestFirst, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for left, right := 0, len(newestFirst)-1; left < right; left, right = left+1, right-1 {
		newestFirst[left], newestFirst[right] = newestFirst[right], newestFirst[left]
	}
	return newestFirst, nil
}

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

func severityRank(value string) int {
	return map[string]int{"INFO": 1, "LOW": 2, "MEDIUM": 3, "HIGH": 4, "CRITICAL": 5}[value]
}

func queueFindingNotification(ctx context.Context, tx *sql.Tx, finding models.Finding, eventType string) error {
	maintenance, suppressed, _, controlErr := findingControl(ctx, tx, finding, time.Now().UTC())
	if controlErr != nil {
		return controlErr
	}
	if maintenance || suppressed {
		return nil
	}
	var routeCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_routes`).Scan(&routeCount); err != nil {
		return err
	}
	if routeCount == 0 && severityRank(string(finding.Severity)) < severityRank(string(models.SeverityHigh)) {
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
	if routeCount == 0 {
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO finding_notification_deliveries(event_id,destination_id) SELECT ?,id FROM notification_configs WHERE enabled=1`, eventID)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM finding_notification_deliveries WHERE rowid NOT IN (SELECT d.rowid FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id ORDER BY e.created_at DESC,d.destination_id LIMIT 2000)`)
		if err == nil {
			_, err = tx.ExecContext(ctx, `DELETE FROM finding_notification_events WHERE NOT EXISTS (SELECT 1 FROM finding_notification_deliveries d WHERE d.event_id=finding_notification_events.id)`)
		}
		return err
	}
	var tagsJSON string
	if err := tx.QueryRowContext(ctx, `SELECT tags_json FROM servers WHERE id=?`, finding.ServerID).Scan(&tagsJSON); err != nil {
		return err
	}
	var serverTags []string
	_ = json.Unmarshal([]byte(tagsJSON), &serverTags)
	rows, err := tx.QueryContext(ctx, `SELECT id,severities_json,categories_json,server_ids_json,server_tags_json,transitions_json,cooldown_seconds FROM notification_routes WHERE enabled=1 ORDER BY priority,id`)
	if err != nil {
		return err
	}
	type match struct {
		id       string
		cooldown int
	}
	var matches []match
	for rows.Next() {
		var id, severitiesJSON, categoriesJSON, serversJSON, tagsJSON, transitionsJSON string
		var cooldown int
		if err := rows.Scan(&id, &severitiesJSON, &categoriesJSON, &serversJSON, &tagsJSON, &transitionsJSON, &cooldown); err != nil {
			rows.Close()
			return err
		}
		if routeMatches(severitiesJSON, string(finding.Severity)) && routeMatchesFold(categoriesJSON, finding.Category) && routeMatches(serversJSON, finding.ServerID) && routeTagsMatch(tagsJSON, serverTags) && routeMatches(transitionsJSON, eventType) {
			matches = append(matches, match{id, cooldown})
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	destinationCooldowns := map[string]int{}
	for _, matched := range matches {
		destinations, err := tx.QueryContext(ctx, `SELECT d.id FROM notification_route_destinations rd JOIN notification_configs d ON d.id=rd.destination_id WHERE rd.route_id=? AND d.enabled=1 ORDER BY d.id`, matched.id)
		if err != nil {
			return err
		}
		for destinations.Next() {
			var destinationID string
			if err := destinations.Scan(&destinationID); err != nil {
				destinations.Close()
				return err
			}
			current, exists := destinationCooldowns[destinationID]
			if !exists || matched.cooldown < current {
				destinationCooldowns[destinationID] = matched.cooldown
			}
		}
		if err := destinations.Close(); err != nil {
			return err
		}
	}
	for destinationID, cooldown := range destinationCooldowns {
		status, reason := "pending", ""
		if cooldown > 0 {
			var recent int
			cutoff := time.Now().UTC().Add(-time.Duration(cooldown) * time.Second).Format(time.RFC3339Nano)
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id WHERE d.destination_id=? AND e.finding_id=? AND d.status='delivered' AND d.delivered_at>=?`, destinationID, finding.ID, cutoff).Scan(&recent); err != nil {
				return err
			}
			if recent > 0 {
				status, reason = "cooldown", "A matching delivery succeeded within the route cooldown."
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO finding_notification_deliveries(event_id,destination_id,status,skipped_reason) VALUES(?,?,?,?)`, eventID, destinationID, status, reason); err != nil {
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM finding_notification_deliveries WHERE rowid NOT IN (SELECT d.rowid FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id ORDER BY e.created_at DESC,d.destination_id LIMIT 2000)`)
	if err == nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM finding_notification_events WHERE NOT EXISTS (SELECT 1 FROM finding_notification_deliveries d WHERE d.event_id=finding_notification_events.id)`)
	}
	return err
}

func routeMatches(raw, value string) bool {
	var values []string
	if json.Unmarshal([]byte(raw), &values) != nil || len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
func routeMatchesFold(raw, value string) bool {
	var values []string
	if json.Unmarshal([]byte(raw), &values) != nil || len(values) == 0 {
		return true
	}
	for _, candidate := range values {
		if strings.EqualFold(candidate, value) {
			return true
		}
	}
	return false
}
func routeTagsMatch(raw string, serverTags []string) bool {
	var wanted []string
	if json.Unmarshal([]byte(raw), &wanted) != nil || len(wanted) == 0 {
		return true
	}
	for _, want := range wanted {
		for _, actual := range serverTags {
			if strings.EqualFold(want, actual) {
				return true
			}
		}
	}
	return false
}

type FindingFilter struct {
	Status, ServerID, Severity, Category, Search, RuleID string
	Limit                                                int
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
