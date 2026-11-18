// Package store holds the SQL queries, one file per aggregate (spec §4, §6).
// It uses pgx directly (D5). updated_at is maintained by triggers (§6.11), so
// queries never set it.
package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound is returned when a looked-up row does not exist.
var ErrNotFound = errors.New("not found")

// DB is the part of pgxpool.Pool (and pgx.Tx) the store uses, so the same
// queries run on a pool or inside a transaction.
type DB interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store runs queries against a DB.
type Store struct {
	db DB
}

// New returns a Store on db.
func New(db DB) *Store { return &Store{db: db} }

// inTx runs fn in a transaction (a savepoint if s already runs in one). It
// commits if fn returns nil and rolls back otherwise.
func (s *Store) inTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.db, fn)
}

// notFound maps pgx.ErrNoRows to ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
