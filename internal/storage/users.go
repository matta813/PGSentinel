package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
)

func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	_, err := s.DB.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,password_salt,must_change_password,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		user.ID, user.Username, user.PasswordHash, user.PasswordSalt, user.MustChangePassword, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	var user models.User
	var created, updated string
	err := s.DB.QueryRowContext(ctx, `SELECT id,username,password_hash,password_salt,must_change_password,created_at,updated_at FROM users WHERE username=?`, username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.PasswordSalt, &user.MustChangePassword, &created, &updated)
	if err != nil {
		return user, err
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return user, nil
}

func (s *Store) UpdateUserPassword(ctx context.Context, id string, hash, salt []byte) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE users SET password_hash=?,password_salt=?,must_change_password=0,updated_at=? WHERE id=?`, hash, salt, time.Now().UTC().Format(time.RFC3339Nano), id)
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
