package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.scruzzi.com/root/postgresqlui/internal/models"
)

type Client struct{ pool *pgxpool.Pool }

func Connect(ctx context.Context, s models.Server) (*Client, error) {
	u := &url.URL{Scheme: "postgres", Host: fmt.Sprintf("%s:%d", s.Host, s.Port), User: url.UserPassword(s.User, s.Password), Path: "/postgres"}
	q := u.Query()
	q.Set("sslmode", s.SSLMode)
	q.Set("application_name", "pgsentinel")
	u.RawQuery = q.Encode()
	cfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, fmt.Errorf("invalid connection configuration: %w", err)
	}
	cfg.MaxConns = 2
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, friendly(err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	if err = pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, friendly(err)
	}
	return &Client{pool: pool}, nil
}
func friendly(err error) error {
	return fmt.Errorf("unable to connect to PostgreSQL; verify reachability, pg_hba.conf, credentials and SSL mode: %w", err)
}
func (c *Client) Close()              { c.pool.Close() }
func (c *Client) Pool() *pgxpool.Pool { return c.pool }
func (c *Client) Version(ctx context.Context) (string, error) {
	var v string
	err := c.pool.QueryRow(ctx, "SHOW server_version").Scan(&v)
	return v, err
}
