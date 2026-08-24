package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) PendingFindingNotifications(ctx context.Context, limit int) ([]models.FindingNotificationDelivery, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT d.event_id,d.destination_id,e.event_type,e.finding_json,d.attempts FROM finding_notification_deliveries d JOIN finding_notification_events e ON e.id=d.event_id WHERE d.delivered_at IS NULL AND d.attempts < 3 ORDER BY e.created_at LIMIT ?`, limit)
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
	if sendErr == nil {
		_, err := s.DB.ExecContext(ctx, `UPDATE finding_notification_deliveries SET attempts=attempts+1,delivered_at=?,last_error='' WHERE event_id=? AND destination_id=?`, time.Now().UTC().Format(time.RFC3339Nano), eventID, destinationID)
		return err
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE finding_notification_deliveries SET attempts=attempts+1,last_error=? WHERE event_id=? AND destination_id=?`, sendErr.Error(), eventID, destinationID)
	return err
}
