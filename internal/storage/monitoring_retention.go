package storage

import (
	"context"
	"time"
)

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
