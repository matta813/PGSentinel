package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) CreateNotificationDestination(ctx context.Context, v *models.NotificationDestination) error {
	now := time.Now().UTC()
	v.CreatedAt, v.UpdatedAt = now, now
	config, err := encryptNotificationConfig(s, v.Config)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO notification_configs(id,provider,name,config_cipher,enabled,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`, v.ID, v.Provider, v.Name, config, v.Enabled, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return err
}

func (s *Store) ListNotificationDestinations(ctx context.Context) ([]models.NotificationDestination, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,provider,name,enabled,created_at,updated_at FROM notification_configs ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.NotificationDestination{}
	for rows.Next() {
		var v models.NotificationDestination
		var created, updated string
		if err := rows.Scan(&v.ID, &v.Provider, &v.Name, &v.Enabled, &created, &updated); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetNotificationDestination(ctx context.Context, id string, includeConfig bool) (models.NotificationDestination, error) {
	var v models.NotificationDestination
	var config []byte
	var created, updated string
	err := s.DB.QueryRowContext(ctx, `SELECT id,provider,name,config_cipher,enabled,created_at,updated_at FROM notification_configs WHERE id=?`, id).Scan(&v.ID, &v.Provider, &v.Name, &config, &v.Enabled, &created, &updated)
	if err != nil {
		return v, err
	}
	if includeConfig {
		plain, err := s.cipher.Decrypt(config)
		if err != nil {
			return v, err
		}
		if err := json.Unmarshal([]byte(plain), &v.Config); err != nil {
			return v, err
		}
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return v, nil
}

func (s *Store) UpdateNotificationDestination(ctx context.Context, v *models.NotificationDestination) error {
	config, err := encryptNotificationConfig(s, v.Config)
	if err != nil {
		return err
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE notification_configs SET provider=?,name=?,config_cipher=?,enabled=?,updated_at=? WHERE id=?`, v.Provider, v.Name, config, v.Enabled, time.Now().UTC().Format(time.RFC3339Nano), v.ID)
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

func (s *Store) DeleteNotificationDestination(ctx context.Context, id string) error {
	result, err := s.DB.ExecContext(ctx, `DELETE FROM notification_configs WHERE id=?`, id)
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

func encryptNotificationConfig(s *Store, config map[string]string) ([]byte, error) {
	encoded, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	return s.cipher.Encrypt(string(encoded))
}
