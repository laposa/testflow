// Package db opens the PostgreSQL connection pool and applies the embedded
// schema migrations (spec D5, D6, §6.11).
package db

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config describes how to connect: either URL (a postgres:// URL or a
// key=value DSN) or Host, Port and Name. User and Password, when not empty,
// replace the ones in URL.
type Config struct {
	URL      string
	Host     string
	Port     int    // with Host; 0 means 5432
	Name     string // with Host; database name
	User     string
	Password string
}

// connString returns URL, or a postgres:// URL built from Host, Port and
// Name. Credentials are never put into it; Open sets them separately.
func (c Config) connString() string {
	if c.URL != "" || c.Host == "" {
		return c.URL
	}
	port := c.Port
	if port == 0 {
		port = 5432
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(c.Host, strconv.Itoa(port)), Path: "/" + c.Name}
	return u.String()
}

// Open creates a connection pool and checks that the database is reachable.
func Open(ctx context.Context, c Config) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(c.connString())
	if err != nil {
		// pgx redacts the password in parse errors.
		return nil, fmt.Errorf("parse database settings: %w", err)
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
