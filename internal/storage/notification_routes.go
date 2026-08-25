package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) CreateNotificationRoute(ctx context.Context, route *models.NotificationRoute) error {
	now := time.Now().UTC()
	route.CreatedAt, route.UpdatedAt = now, now
	return s.saveNotificationRoute(ctx, route, true)
}

func (s *Store) UpdateNotificationRoute(ctx context.Context, route *models.NotificationRoute) error {
	route.UpdatedAt = time.Now().UTC()
	return s.saveNotificationRoute(ctx, route, false)
}

func (s *Store) saveNotificationRoute(ctx context.Context, route *models.NotificationRoute, create bool) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	severities, _ := json.Marshal(route.Severities)
	categories, _ := json.Marshal(route.Categories)
	servers, _ := json.Marshal(route.ServerIDs)
	tags, _ := json.Marshal(route.ServerTags)
	transitions, _ := json.Marshal(route.Transitions)
	if create {
		_, err = tx.ExecContext(ctx, `INSERT INTO notification_routes(id,name,enabled,priority,severities_json,categories_json,server_ids_json,server_tags_json,transitions_json,cooldown_seconds,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, route.ID, route.Name, route.Enabled, route.Priority, severities, categories, servers, tags, transitions, route.CooldownSeconds, route.CreatedAt.Format(time.RFC3339Nano), route.UpdatedAt.Format(time.RFC3339Nano))
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE notification_routes SET name=?,enabled=?,priority=?,severities_json=?,categories_json=?,server_ids_json=?,server_tags_json=?,transitions_json=?,cooldown_seconds=?,updated_at=? WHERE id=?`, route.Name, route.Enabled, route.Priority, severities, categories, servers, tags, transitions, route.CooldownSeconds, route.UpdatedAt.Format(time.RFC3339Nano), route.ID)
		if err == nil {
			if n, e := result.RowsAffected(); e != nil {
				err = e
			} else if n == 0 {
				err = sql.ErrNoRows
			}
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `DELETE FROM notification_route_destinations WHERE route_id=?`, route.ID)
		}
	}
	if err != nil {
		return err
	}
	for _, destinationID := range route.DestinationIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO notification_route_destinations(route_id,destination_id) VALUES(?,?)`, route.ID, destinationID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListNotificationRoutes(ctx context.Context) ([]models.NotificationRoute, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,enabled,priority,severities_json,categories_json,server_ids_json,server_tags_json,transitions_json,cooldown_seconds,created_at,updated_at FROM notification_routes ORDER BY priority,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	routes := []models.NotificationRoute{}
	for rows.Next() {
		var route models.NotificationRoute
		var severities, categories, servers, tags, transitions, created, updated string
		if err := rows.Scan(&route.ID, &route.Name, &route.Enabled, &route.Priority, &severities, &categories, &servers, &tags, &transitions, &route.CooldownSeconds, &created, &updated); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(severities), &route.Severities)
		_ = json.Unmarshal([]byte(categories), &route.Categories)
		_ = json.Unmarshal([]byte(servers), &route.ServerIDs)
		_ = json.Unmarshal([]byte(tags), &route.ServerTags)
		_ = json.Unmarshal([]byte(transitions), &route.Transitions)
		route.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		route.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		destinationRows, queryErr := s.DB.QueryContext(ctx, `SELECT destination_id FROM notification_route_destinations WHERE route_id=? ORDER BY destination_id`, route.ID)
		if queryErr != nil {
			return nil, queryErr
		}
		for destinationRows.Next() {
			var id string
			if err := destinationRows.Scan(&id); err != nil {
				destinationRows.Close()
				return nil, err
			}
			route.DestinationIDs = append(route.DestinationIDs, id)
		}
		if err := destinationRows.Close(); err != nil {
			return nil, err
		}
		routes = append(routes, route)
	}
	return routes, rows.Err()
}

func (s *Store) DeleteNotificationRoute(ctx context.Context, id string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM notification_routes WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
