package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/matta813/pgsentinel/internal/models"
	"regexp"
	"strings"
	"time"
)

func (s *Store) PendingFindingNotifications(ctx context.Context, limit int) ([]models.FindingNotificationDelivery, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT d.event_id,d.destination_id,e.event_type,e.finding_json,d.attempts FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id WHERE d.status IN ('pending','retry') AND d.attempts < 3 AND (d.next_attempt_at IS NULL OR d.next_attempt_at<=?) ORDER BY e.created_at,d.destination_id LIMIT ?`, time.Now().UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.FindingNotificationDelivery
	for rows.Next() {
		var item models.FindingNotificationDelivery
		var body string
		if err := rows.Scan(&item.EventID, &item.DestinationID, &item.EventType, &body, &item.Attempts); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(body), &item.Finding); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) RecordFindingNotification(ctx context.Context, eventID, destinationID string, sendErr error) error {
	now := time.Now().UTC()
	if sendErr == nil {
		_, err := s.DB.ExecContext(ctx, `UPDATE finding_notification_deliveries SET attempts=attempts+1,status='delivered',delivered_at=?,last_attempt_at=?,next_attempt_at=NULL,last_error='' WHERE event_id=? AND destination_id=?`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), eventID, destinationID)
		if err == nil {
			err = s.pruneNotificationHistory(ctx, 2000)
		}
		return err
	}
	var attempts int
	if err := s.DB.QueryRowContext(ctx, `SELECT attempts FROM finding_notification_deliveries WHERE event_id=? AND destination_id=?`, eventID, destinationID).Scan(&attempts); err != nil {
		return err
	}
	attempts++
	status := "retry"
	var next any = now.Add(time.Duration(attempts*attempts) * time.Minute).Format(time.RFC3339Nano)
	if attempts >= 3 {
		status, next = "failed", nil
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE finding_notification_deliveries SET attempts=?,status=?,last_attempt_at=?,next_attempt_at=?,last_error=? WHERE event_id=? AND destination_id=?`, attempts, status, now.Format(time.RFC3339Nano), next, SafeDeliveryError(sendErr), eventID, destinationID)
	if err == nil {
		err = s.pruneNotificationHistory(ctx, 2000)
	}
	return err
}

var deliveryURL = regexp.MustCompile(`https?://[^\s]+`)

func SafeDeliveryError(err error) string {
	if err == nil {
		return ""
	}
	value := deliveryURL.ReplaceAllString(err.Error(), "[redacted target]")
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 500 {
		value = value[:500]
	}
	return value
}
func (s *Store) pruneNotificationHistory(ctx context.Context, keep int) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM finding_notification_deliveries WHERE rowid NOT IN (SELECT d.rowid FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id ORDER BY e.created_at DESC,d.destination_id LIMIT ?)`, keep); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM finding_notification_events WHERE NOT EXISTS (SELECT 1 FROM finding_notification_deliveries d WHERE d.event_id=finding_notification_events.id)`); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListNotificationDeliveryHistory(ctx context.Context, limit, offset int) ([]models.NotificationDeliveryHistory, error) {
	if limit < 1 || limit > 200 || offset < 0 {
		return nil, fmt.Errorf("invalid delivery history pagination")
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT d.event_id,d.destination_id,n.name,e.event_type,e.finding_id,e.finding_json,d.status,d.attempts,d.last_error,e.created_at,d.last_attempt_at,d.delivered_at,d.next_attempt_at,d.skipped_reason,COALESCE(s.name,'') FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id LEFT JOIN notification_configs n ON n.id=d.destination_id LEFT JOIN findings f ON f.id=e.finding_id LEFT JOIN servers s ON s.id=f.server_id ORDER BY e.created_at DESC,d.destination_id LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.NotificationDeliveryHistory{}
	for rows.Next() {
		var item models.NotificationDeliveryHistory
		var body, created, skipped string
		var name, last, delivered, next sql.NullString
		if err := rows.Scan(&item.EventID, &item.DestinationID, &name, &item.EventType, &item.FindingID, &body, &item.Status, &item.Attempts, &item.LastError, &created, &last, &delivered, &next, &skipped, &item.ServerName); err != nil {
			return nil, err
		}
		var finding models.Finding
		if err := json.Unmarshal([]byte(body), &finding); err != nil {
			return nil, err
		}
		item.DestinationName = name.String
		item.FindingTitle = finding.Title
		item.ServerID = finding.ServerID
		item.Severity = string(finding.Severity)
		item.Category = finding.Category
		item.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if item.Status == "cooldown" {
			item.LastError = skipped
		}
		if last.Valid {
			v, _ := time.Parse(time.RFC3339Nano, last.String)
			item.LastAttemptAt = &v
		}
		if delivered.Valid {
			v, _ := time.Parse(time.RFC3339Nano, delivered.String)
			item.DeliveredAt = &v
		}
		if next.Valid {
			v, _ := time.Parse(time.RFC3339Nano, next.String)
			item.NextAttemptAt = &v
		}
		out = append(out, item)
	}
	return out, rows.Err()
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
