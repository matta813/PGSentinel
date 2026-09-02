package storage

import (
	"context"
	"encoding/json"
	"github.com/matta813/pgsentinel/internal/models"
	"time"
)

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
