package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/matta813/pgsentinel/internal/models"
	"github.com/matta813/pgsentinel/migrations"
	_ "modernc.org/sqlite"
)

type Store struct {
	DB     *sql.DB
	cipher *Cipher
}

func Open(path, key string) (*Store, error) {
	c, err := NewCipher(key)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{DB: db, cipher: c}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		version, err := strconv.Atoi(strings.SplitN(e.Name(), "_", 2)[0])
		if err != nil {
			return err
		}
		var exists int
		if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}
		body, err := migrations.FS.ReadFile(e.Name())
		if err != nil {
			return err
		}
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, string(body)); err == nil {
			_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(?,?)`, version, time.Now().UTC().Format(time.RFC3339Nano))
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", e.Name(), err)
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) CreateServer(ctx context.Context, v *models.Server) error {
	now := time.Now().UTC()
	v.CreatedAt = now
	v.UpdatedAt = now
	if v.Status == "" {
		v.Status = "unknown"
	}
	pass, err := s.cipher.Encrypt(v.Password)
	if err != nil {
		return err
	}
	tags, _ := json.Marshal(v.Tags)
	_, err = s.DB.ExecContext(ctx, `INSERT INTO servers(id,name,host,port,username,password_cipher,ssl_mode,status,tags_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, v.ID, v.Name, v.Host, v.Port, v.User, pass, v.SSLMode, v.Status, string(tags), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return err
}
func (s *Store) UpdateServer(ctx context.Context, v *models.Server) error {
	tags, _ := json.Marshal(v.Tags)
	now := time.Now().UTC()
	var (
		result sql.Result
		err    error
	)
	if v.Password == "" {
		result, err = s.DB.ExecContext(ctx, `UPDATE servers SET name=?,host=?,port=?,username=?,ssl_mode=?,tags_json=?,updated_at=? WHERE id=?`, v.Name, v.Host, v.Port, v.User, v.SSLMode, string(tags), now.Format(time.RFC3339Nano), v.ID)
	} else {
		pass, encryptErr := s.cipher.Encrypt(v.Password)
		if encryptErr != nil {
			return encryptErr
		}
		result, err = s.DB.ExecContext(ctx, `UPDATE servers SET name=?,host=?,port=?,username=?,password_cipher=?,ssl_mode=?,tags_json=?,updated_at=? WHERE id=?`, v.Name, v.Host, v.Port, v.User, pass, v.SSLMode, string(tags), now.Format(time.RFC3339Nano), v.ID)
	}
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
func (s *Store) ListServers(ctx context.Context) ([]models.Server, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,host,port,username,ssl_mode,version,status,last_connected_at,last_error,tags_json,created_at,updated_at FROM servers ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Server{}
	for rows.Next() {
		var v models.Server
		var last sql.NullString
		var tags, created, updated string
		if err := rows.Scan(&v.ID, &v.Name, &v.Host, &v.Port, &v.User, &v.SSLMode, &v.Version, &v.Status, &last, &v.LastError, &tags, &created, &updated); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(tags), &v.Tags)
		v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		if last.Valid {
			t, _ := time.Parse(time.RFC3339Nano, last.String)
			v.LastConnectedAt = &t
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) GetServer(ctx context.Context, id string, includePassword bool) (models.Server, error) {
	var v models.Server
	var pass []byte
	var last sql.NullString
	var tags, created, updated string
	err := s.DB.QueryRowContext(ctx, `SELECT id,name,host,port,username,password_cipher,ssl_mode,version,status,last_connected_at,last_error,tags_json,created_at,updated_at FROM servers WHERE id=?`, id).Scan(&v.ID, &v.Name, &v.Host, &v.Port, &v.User, &pass, &v.SSLMode, &v.Version, &v.Status, &last, &v.LastError, &tags, &created, &updated)
	if err != nil {
		return v, err
	}
	if includePassword {
		v.Password, err = s.cipher.Decrypt(pass)
		if err != nil {
			return v, err
		}
	}
	json.Unmarshal([]byte(tags), &v.Tags)
	v.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	v.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	if last.Valid {
		t, _ := time.Parse(time.RFC3339Nano, last.String)
		v.LastConnectedAt = &t
	}
	return v, nil
}
func (s *Store) DeleteServer(ctx context.Context, id string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM servers WHERE id=?`, id)
	return err
}
func (s *Store) UpdateServerStatus(ctx context.Context, id, status, version, lastError string, connected bool) error {
	var last any = nil
	if connected {
		last = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE servers SET status=?,version=?,last_error=?,last_connected_at=COALESCE(?,last_connected_at),updated_at=? WHERE id=?`, status, version, lastError, last, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}
