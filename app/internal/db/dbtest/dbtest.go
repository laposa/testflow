// Package dbtest gives store and integration tests an isolated PostgreSQL
// schema (spec §13).
//
// Tests are skipped unless TEST_DATABASE_URL is set (`make test-integration`
// sets it to the compose Postgres). Each call creates a fresh schema named
// test_<random>, points the connection's search_path at it, and drops it when
// the test ends, so tests can run in parallel against one database.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/laposa/testflow/app-go/internal/db"
)

// EnvVar names the variable holding the test database URL.
const EnvVar = "TEST_DATABASE_URL"

// New returns a pool on a new, fully migrated schema.
func New(t testing.TB) *pgxpool.Pool {
	t.Helper()
	pool := Open(t, NewSchemaURL(t))
	migrations, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Migrate(t.Context(), pool, migrations); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}
	return pool
}

// NewSchemaURL creates a new, empty schema and returns a database URL whose
// connections use it. Use it to test code that opens its own pool.
func NewSchemaURL(t testing.TB) string {
	t.Helper()
	base := os.Getenv(EnvVar)
	if base == "" {
		t.Skipf("%s not set; skipping database test (run make test-integration)", EnvVar)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect to %s: %v", EnvVar, err)
	}
	defer conn.Close(ctx)

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	schema := "test_" + hex.EncodeToString(b)
	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		conn, err := pgx.Connect(ctx, base)
		if err != nil {
			t.Errorf("drop schema %s: %v", schema, err)
			return
		}
		defer conn.Close(ctx)
		if _, err := conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop schema %s: %v", schema, err)
		}
	})
	return withSearchPath(t, base, schema)
}

// Open returns a pool for url (credentials in the URL) that is closed when
// the test ends.
func Open(t testing.TB, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := db.Open(t.Context(), db.Config{URL: url})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// withSearchPath adds search_path to a URL or key=value DSN. pgx sends
// unknown parameters to the server as run-time settings.
func withSearchPath(t testing.TB, dsn, schema string) string {
	t.Helper()
	if !strings.Contains(dsn, "://") {
		return dsn + " search_path=" + schema
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse %s: %v", EnvVar, err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}
