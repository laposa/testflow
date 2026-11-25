package db_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/laposa/testflow/app-go/internal/db"
	"github.com/laposa/testflow/app-go/internal/db/dbtest"
)

func TestEmbeddedMigrations(t *testing.T) {
	migrations, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 || migrations[0].Name != "0001_init.sql" {
		t.Fatalf("migrations = %v", migrations)
	}
	for i, m := range migrations {
		if m.Version != int64(i+1) {
			t.Errorf("%s: version %d, want %d (no gaps)", m.Name, m.Version, i+1)
		}
	}
}

func TestLoadMigrations(t *testing.T) {
	ok := fstest.MapFS{
		"0002_b.sql": {Data: []byte("SELECT 2")},
		"0001_a.sql": {Data: []byte("SELECT 1")},
		"0010_c.sql": {Data: []byte("SELECT 10")},
	}
	got, err := db.LoadMigrations(ok)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, m := range got {
		names = append(names, m.Name)
	}
	if want := []string{"0001_a.sql", "0002_b.sql", "0010_c.sql"}; !slices.Equal(names, want) {
		t.Errorf("order = %v, want %v", names, want)
	}
	if got[2].Version != 10 || got[2].SQL != "SELECT 10" {
		t.Errorf("parsed %+v", got[2])
	}

	bad := map[string]fstest.MapFS{
		"bad name":     {"init.sql": {}},
		"upper case":   {"0001_Init.sql": {}},
		"zero version": {"0000_x.sql": {}},
		"duplicate":    {"0001_a.sql": {}, "0001_b.sql": {}},
		"not sql":      {"0001_a.txt": {}},
	}
	for name, fsys := range bad {
		if _, err := db.LoadMigrations(fsys); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestMigrateFreshDatabase(t *testing.T) {
	ctx := t.Context()
	pool := dbtest.Open(t, dbtest.NewSchemaURL(t))
	migrations, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}

	st, err := db.CheckStatus(ctx, pool, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Pending) != len(migrations) {
		t.Fatalf("pending = %d, want all %d", len(st.Pending), len(migrations))
	}
	if len(tables(t, pool)) != 0 {
		t.Fatal("CheckStatus must not create tables")
	}

	applied, err := db.Migrate(ctx, pool, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != len(migrations) {
		t.Errorf("applied %d, want %d", len(applied), len(migrations))
	}

	want := []string{
		"activity_log", "review_requests", "run_test_results", "schema_migrations",
		"service_runs", "service_suites", "session_services", "suite_tests",
		"test_sessions", "users", "web_sessions",
	}
	if got := tables(t, pool); !slices.Equal(got, want) {
		t.Errorf("tables = %v\nwant     %v", got, want)
	}

	// A second run is a no-op.
	again, err := db.Migrate(ctx, pool, migrations)
	if err != nil || len(again) != 0 {
		t.Errorf("second run applied %d, err %v", len(again), err)
	}
	if st, err := db.CheckStatus(ctx, pool, migrations); err != nil || len(st.Pending) != 0 || len(st.Unknown) != 0 {
		t.Errorf("status after migrate = %+v, %v", st, err)
	}
}

func TestMigrateFailureRollsBack(t *testing.T) {
	ctx := t.Context()
	pool := dbtest.Open(t, dbtest.NewSchemaURL(t))
	migrations, err := db.LoadMigrations(fstest.MapFS{
		"0001_ok.sql":     {Data: []byte("CREATE TABLE a (id int);")},
		"0002_broken.sql": {Data: []byte("CREATE TABLE b (id int); SELECT no_such_column FROM a;")},
	})
	if err != nil {
		t.Fatal(err)
	}

	applied, err := db.Migrate(ctx, pool, migrations)
	if err == nil || !strings.Contains(err.Error(), "0002_broken.sql") {
		t.Fatalf("err = %v, want failure naming the file", err)
	}
	if len(applied) != 1 || applied[0].Version != 1 {
		t.Errorf("applied = %v, want only 0001", applied)
	}
	if got := tables(t, pool); !slices.Equal(got, []string{"a", "schema_migrations"}) {
		t.Errorf("tables = %v: table b must be rolled back", got)
	}
	st, err := db.CheckStatus(ctx, pool, migrations)
	if err != nil || len(st.Pending) != 1 || st.Pending[0].Version != 2 {
		t.Errorf("status = %+v, %v; want 0002 still pending", st, err)
	}
}

func TestMigrateReportsUnknownVersions(t *testing.T) {
	ctx := t.Context()
	pool := dbtest.Open(t, dbtest.NewSchemaURL(t))
	newer, _ := db.LoadMigrations(fstest.MapFS{
		"0001_a.sql": {Data: []byte("SELECT 1")},
		"0002_b.sql": {Data: []byte("SELECT 1")},
	})
	if _, err := db.Migrate(ctx, pool, newer); err != nil {
		t.Fatal(err)
	}
	st, err := db.CheckStatus(ctx, pool, newer[:1]) // an older binary
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(st.Unknown, []int64{2}) || len(st.Pending) != 0 {
		t.Errorf("status = %+v, want unknown [2]", st)
	}
}

func TestMigrateConcurrently(t *testing.T) {
	url := dbtest.NewSchemaURL(t)
	migrations, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		total int
	)
	for range 3 {
		pool := dbtest.Open(t, url)
		wg.Go(func() {
			applied, err := db.Migrate(t.Context(), pool, migrations)
			if err != nil {
				t.Error(err)
			}
			mu.Lock()
			total += len(applied)
			mu.Unlock()
		})
	}
	wg.Wait()
	if total != len(migrations) {
		t.Errorf("applied %d in total, want each migration exactly once (%d)", total, len(migrations))
	}
}

// TestSchemaConstraints checks the parts of §6 that the database enforces:
// cascades, checks, defaults and the updated_at trigger.
func TestSchemaConstraints(t *testing.T) {
	ctx := t.Context()
	pool := dbtest.New(t)

	var userID, sessionID, serviceID, suiteID, testID, runID int64
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO users (github_id, name, login) VALUES (1, 'Ada', 'ada') RETURNING id`), &userID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO test_sessions (issuer_id, name, environment) VALUES ($1, 'Release', 'preprod') RETURNING id`, userID), &sessionID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO session_services (session_id, type, name, path, repository_id, repository_name, commit_sha)
		VALUES ($1, 'automated', 'web', 'tests/web', 7, 'laposa/shop', 'abc') RETURNING id`, sessionID), &serviceID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO service_suites (service_id, name, path) VALUES ($1, 'login', 'tests/web/tests/login') RETURNING id`, serviceID), &suiteID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO suite_tests (suite_id, name, path) VALUES ($1, 'ok.cy.js', 'tests/web/tests/login/ok.cy.js') RETURNING id`, suiteID), &testID)
	mustScan(t, pool.QueryRow(ctx, `INSERT INTO service_runs (service_id, type, status) VALUES ($1, 'automated', 'unknown') RETURNING id`, serviceID), &runID)
	mustExec(t, pool, `INSERT INTO run_test_results (run_id, test_id, status) VALUES ($1, $2, 'pass')`, runID, testID)
	mustExec(t, pool, `INSERT INTO activity_log (session_id, type, body) VALUES ($1, 'session_created', 'Ada created the session.')`, sessionID)
	mustExec(t, pool, `INSERT INTO review_requests (session_id, requester_id, reviewer_id) VALUES ($1, $2, $2)`, sessionID, userID)

	t.Run("defaults", func(t *testing.T) {
		var branch, status, metadata string
		mustScan(t, pool.QueryRow(ctx, `SELECT branch FROM session_services WHERE id = $1`, serviceID), &branch)
		mustScan(t, pool.QueryRow(ctx, `SELECT status FROM review_requests WHERE session_id = $1`, sessionID), &status)
		mustScan(t, pool.QueryRow(ctx, `SELECT metadata::text FROM activity_log WHERE session_id = $1`, sessionID), &metadata)
		if branch != "master" || status != "pending" || metadata != "{}" {
			t.Errorf("branch %q, review status %q, metadata %q", branch, status, metadata)
		}
	})

	t.Run("check constraints", func(t *testing.T) {
		for name, sql := range map[string]string{
			"service type":     `UPDATE session_services SET type = 'other'`,
			"run status":       `UPDATE service_runs SET status = 'done'`,
			"result status":    `UPDATE run_test_results SET status = 'ok'`,
			"activity type":    `UPDATE activity_log SET type = 'deleted'`,
			"review status":    `UPDATE review_requests SET status = 'maybe'`,
			"second pending":   `INSERT INTO review_requests (session_id, requester_id, reviewer_id) SELECT session_id, requester_id, reviewer_id FROM review_requests`,
			"duplicate github": `INSERT INTO users (github_id, name, login) VALUES (1, 'Copy', 'copy')`,
		} {
			_, err := pool.Exec(ctx, sql)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || (pgErr.Code != "23514" && pgErr.Code != "23505") {
				t.Errorf("%s: err = %v, want check or unique violation", name, err)
			}
		}
	})

	t.Run("updated_at trigger", func(t *testing.T) {
		var changed bool
		mustScan(t, pool.QueryRow(ctx, `WITH before AS (SELECT updated_at FROM users WHERE id = $1)
			UPDATE users SET name = 'Ada L.', updated_at = '2000-01-01' WHERE id = $1
			RETURNING updated_at > (SELECT updated_at FROM before)`, userID), &changed)
		if !changed {
			t.Error("updated_at not refreshed by the trigger")
		}
	})

	t.Run("cascade from session", func(t *testing.T) {
		mustExec(t, pool, `DELETE FROM test_sessions WHERE id = $1`, sessionID)
		for _, table := range []string{"session_services", "service_suites", "suite_tests", "service_runs", "run_test_results", "activity_log", "review_requests"} {
			var n int
			mustScan(t, pool.QueryRow(ctx, `SELECT count(*) FROM `+table), &n)
			if n != 0 {
				t.Errorf("%s has %d rows after deleting the session", table, n)
			}
		}
	})
}

func tables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(t.Context(), `SELECT table_name::text FROM information_schema.tables
		WHERE table_schema = current_schema() ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return names
}

func mustScan(t *testing.T, row pgx.Row, dest ...any) {
	t.Helper()
	if err := row.Scan(dest...); err != nil {
		t.Fatal(err)
	}
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}
