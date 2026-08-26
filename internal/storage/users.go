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
	_, err := s.DB.ExecContext(ctx, `INSERT INTO users(id,username,role,password_hash,password_salt,must_change_password,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`,
		user.ID, user.Username, user.Role, user.PasswordHash, user.PasswordSalt, user.MustChangePassword, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (models.User, error) {
	var user models.User
	var created, updated string
	err := s.DB.QueryRowContext(ctx, `SELECT id,username,role,password_hash,password_salt,must_change_password,created_at,updated_at FROM users WHERE username=?`, username).
		Scan(&user.ID, &user.Username, &user.Role, &user.PasswordHash, &user.PasswordSalt, &user.MustChangePassword, &created, &updated)
	if err != nil {
		return user, err
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return user, nil
}

func (s *Store) ListUsers(ctx context.Context) ([]models.User, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,username,role,must_change_password,created_at,updated_at FROM users ORDER BY username COLLATE NOCASE LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		var user models.User
		var created, updated string
		if err := rows.Scan(&user.ID, &user.Username, &user.Role, &user.MustChangePassword, &created, &updated); err != nil {
			return nil, err
		}
		user.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) UpdateUserRole(ctx context.Context, id, role string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE users SET role=?,updated_at=? WHERE id=?`, role, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
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
