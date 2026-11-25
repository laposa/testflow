package db

import (
	"cmp"
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"slices"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var embedded embed.FS

// Migration is one forward-only schema change, read from
// migrations/NNNN_description.sql.
type Migration struct {
	Version int64
	Name    string // file name
	SQL     string
}

// lockKey is the pg_advisory_lock key that serialises concurrent migration
// runs (e.g. two instances starting with -migrate).
const lockKey = 0x7465737466 // "testf"

var fileName = regexp.MustCompile(`^(\d{4})_[a-z0-9_]+\.sql$`)

// Migrations returns the migrations embedded in the binary, in order.
func Migrations() ([]Migration, error) {
	sub, err := fs.Sub(embedded, "migrations")
	if err != nil {
		return nil, err
	}
	return LoadMigrations(sub)
}

// LoadMigrations reads *.sql files from the root of fsys, sorted by version.
// Every file must be named NNNN_description.sql and versions must be unique.
func LoadMigrations(fsys fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var out []Migration
	seen := map[int64]string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileName.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("migration %q: name must match NNNN_description.sql", e.Name())
		}
		version, _ := strconv.ParseInt(m[1], 10, 64)
		if version == 0 {
			return nil, fmt.Errorf("migration %q: versions start at 0001", e.Name())
		}
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("migrations %q and %q have the same version", prev, e.Name())
		}
		seen[version] = e.Name()
		sql, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: version, Name: e.Name(), SQL: string(sql)})
	}
	slices.SortFunc(out, func(a, b Migration) int { return cmp.Compare(a.Version, b.Version) })
	return out, nil
}

const createVersionsTable = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version     bigint PRIMARY KEY,
	applied_at  timestamptz NOT NULL DEFAULT now()
)`

// Status compares migrations with the database. Pending are not applied yet;
// Unknown are versions recorded in the database that this binary doesn't
// have, which means the database was migrated by a newer version.
type Status struct {
	Pending []Migration
	Unknown []int64
}

// CheckStatus reports which migrations are pending, without changing the
// database. A database that was never migrated has all of them pending.
func CheckStatus(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) (Status, error) {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return Status{}, fmt.Errorf("check schema_migrations: %w", err)
	}
	if !exists {
		return Status{Pending: migrations}, nil
	}
	return status(ctx, pool, migrations)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func status(ctx context.Context, q querier, migrations []Migration) (Status, error) {
	rows, err := q.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return Status{}, fmt.Errorf("read schema_migrations: %w", err)
	}
	applied, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return Status{}, fmt.Errorf("read schema_migrations: %w", err)
	}

	var st Status
	for _, m := range migrations {
		if !slices.Contains(applied, m.Version) {
			st.Pending = append(st.Pending, m)
		}
	}
	for _, v := range applied {
		if !slices.ContainsFunc(migrations, func(m Migration) bool { return m.Version == v }) {
			st.Unknown = append(st.Unknown, v)
		}
	}
	slices.Sort(st.Unknown)
	return st, nil
}

// Migrate applies all pending migrations in order, each in its own
// transaction together with its schema_migrations row, and returns the ones
// it applied. A session-level advisory lock makes concurrent runs wait for
// each other; the second run then finds nothing to do.
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations []Migration) (applied []Migration, err error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		return nil, fmt.Errorf("lock: %w", err)
	}
	defer func() {
		// Use a fresh context so the lock is released even if ctx was cancelled.
		if _, uerr := conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, lockKey); uerr != nil {
			err = errors.Join(err, fmt.Errorf("unlock: %w", uerr))
		}
	}()

	if _, err := conn.Exec(ctx, createVersionsTable); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}
	st, err := status(ctx, conn, migrations)
	if err != nil {
		return nil, err
	}

	for _, m := range st.Pending {
		if err := apply(ctx, conn.Conn(), m); err != nil {
			return applied, err
		}
		applied = append(applied, m)
	}
	return applied, nil
}

func apply(ctx context.Context, conn *pgx.Conn, m Migration) error {
	return pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		// Without arguments, pgx uses the simple protocol, which allows a
		// file to contain several statements.
		if _, err := tx.Exec(ctx, m.SQL); err != nil {
			return fmt.Errorf("migration %s: %w", m.Name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, m.Version); err != nil {
			return fmt.Errorf("migration %s: record version: %w", m.Name, err)
		}
		return nil
	})
}
