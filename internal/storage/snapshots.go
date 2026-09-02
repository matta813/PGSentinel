package storage

import (
	"context"
	"encoding/json"
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
