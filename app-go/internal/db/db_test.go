package db_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/laposa/testflow/app-go/internal/db"
	"github.com/laposa/testflow/app-go/internal/db/dbtest"
)

// TestOpenWithCredentials connects as a password-protected role whose
// credentials are passed separately from a URL that has none.
func TestOpenWithCredentials(t *testing.T) {
	adminURL := dbtest.NewSchemaURL(t)
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)

	b := make([]byte, 6)
	_, _ = rand.Read(b)
	role := "tf_login_" + hex.EncodeToString(b)
	const password = "p@ss:w/rd #1" // characters that would need URL-encoding
	if _, err := admin.Exec(ctx, "CREATE ROLE "+role+" LOGIN PASSWORD '"+password+"'"); err != nil {
		t.Skipf("cannot create a test role (needs CREATEROLE): %v", err)
	}
	// A defer (not t.Cleanup) so it runs while admin is still open.
	defer func() {
		if _, err := admin.Exec(ctx, "DROP ROLE "+role); err != nil {
			t.Errorf("drop role %s: %v", role, err)
		}
	}()

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Skip("TEST_DATABASE_URL is not a URL")
	}
	u.User = nil // no credentials in the URL
	bare := u.String()

	pool, err := db.Open(ctx, db.Config{URL: bare, User: role, Password: password})
	if err != nil {
		t.Fatalf("open with credentials: %v", err)
	}
	var who string
	if err := pool.QueryRow(ctx, "SELECT current_user").Scan(&who); err != nil || who != role {
		t.Errorf("current_user = %q, %v; want %q", who, err, role)
	}
	pool.Close() // the role can only be dropped once nothing uses it

	_, err = db.Open(ctx, db.Config{URL: bare, User: role, Password: "wrong"})
	if err == nil {
		t.Fatal("wrong password accepted")
	}
	if strings.Contains(err.Error(), password) {
		t.Error("error message contains the password")
	}
}
