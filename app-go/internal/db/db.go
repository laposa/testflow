// Package db opens the PostgreSQL connection pool and applies the embedded
// schema migrations (spec D5, D6, §6.11).
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config describes how to connect. URL is a postgres:// URL or a key=value
// DSN. User and Password, when not empty, replace the ones in URL.
type Config struct {
	URL      string
	User     string
	Password string
}

// Open creates a connection pool and checks that the database is reachable.
func Open(ctx context.Context, c Config) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(c.URL)
	if err != nil {
		// pgx redacts the password in parse errors.
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	if c.User != "" {
		cfg.ConnConfig.User = c.User
	}
	if c.Password != "" {
		cfg.ConnConfig.Password = c.Password
	}
	cfg.ConnConfig.RuntimeParams["application_name"] = "testflow"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}
