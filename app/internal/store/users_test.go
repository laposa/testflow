package store_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/db/dbtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

func TestDisplayName(t *testing.T) {
	tests := []struct {
		gh   store.GitHubUser
		want string
	}{
		{store.GitHubUser{Name: "Ada", Email: "ada@example.com", Login: "ada"}, "Ada"},
		{store.GitHubUser{Email: "ada@example.com", Login: "ada"}, "ada@example.com"},
		{store.GitHubUser{Login: "ada"}, "ada"},
	}
	for _, tt := range tests {
		if got := tt.gh.DisplayName(); got != tt.want {
			t.Errorf("DisplayName(%+v) = %q, want %q", tt.gh, got, tt.want)
		}
	}
}

func TestUpsertUser(t *testing.T) {
	s := store.New(dbtest.New(t))
	ctx := t.Context()

	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 42, Login: "ada", Email: "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 || u.GitHubID != 42 || u.Name != "ada@example.com" || u.Email != "ada@example.com" || u.Login != "ada" {
		t.Errorf("created user = %+v", u)
	}

	// Same GitHub id: the row is updated, not duplicated.
	u2, err := s.UpsertUser(ctx, store.GitHubUser{ID: 42, Login: "ada2", Name: "Ada Lovelace", Email: "ada@lovelace.dev"})
	if err != nil {
		t.Fatal(err)
	}
	if u2.ID != u.ID || u2.Name != "Ada Lovelace" || u2.Email != "ada@lovelace.dev" || u2.Login != "ada2" {
		t.Errorf("updated user = %+v, want id %d", u2, u.ID)
	}

	got, err := s.User(ctx, u.ID)
	if err != nil || got.Name != "Ada Lovelace" {
		t.Errorf("User(%d) = %+v, %v", u.ID, got, err)
	}
	if _, err := s.User(ctx, u.ID+1000); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing user err = %v, want ErrNotFound", err)
	}

	if _, err := s.UpsertUser(ctx, store.GitHubUser{Login: "nobody"}); err == nil {
		t.Error("upsert without GitHub id should fail")
	}
}

// An email GitHub could not return this time (empty) keeps the stored one,
// and the name falls back to it as on the first login.
func TestUpsertUserKeepsEmailWhenUnknown(t *testing.T) {
	s := store.New(dbtest.New(t))
	ctx := t.Context()

	if _, err := s.UpsertUser(ctx, store.GitHubUser{ID: 7, Login: "grace", Email: "grace@example.com"}); err != nil {
		t.Fatal(err)
	}
	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 7, Login: "grace"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "grace@example.com" || u.Name != "grace@example.com" {
		t.Errorf("after login without email: %+v, want email and name kept", u)
	}

	// A new user without an email is stored with NULL and named by login.
	n, err := s.UpsertUser(ctx, store.GitHubUser{ID: 8, Login: "linus"})
	if err != nil {
		t.Fatal(err)
	}
	if n.Email != "" || n.Name != "linus" {
		t.Errorf("new user without email = %+v", n)
	}
}

func TestWebSessions(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()
	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}

	// A clock far from the database's: expiry must follow it, not now().
	now := time.Date(2040, 1, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	if err := s.CreateWebSession(ctx, "live", u.ID, future); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateWebSession(ctx, "old", u.ID, now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, exp, err := s.WebSessionUser(ctx, "live", now)
	if err != nil || got.ID != u.ID || got.Name != "Ada" || !exp.Equal(future) {
		t.Errorf("WebSessionUser(live) = %+v, %v, %v", got, exp, err)
	}
	if _, _, err := s.WebSessionUser(ctx, "live", future); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("session at its expiry err = %v, want ErrNotFound", err)
	}
	if _, _, err := s.WebSessionUser(ctx, "old", now); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired session err = %v, want ErrNotFound", err)
	}
	if _, _, err := s.WebSessionUser(ctx, "missing", now); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing session err = %v, want ErrNotFound", err)
	}

	later := future.Add(time.Hour)
	if err := s.ExtendWebSession(ctx, "live", later, now); err != nil {
		t.Fatal(err)
	}
	if _, exp, _ := s.WebSessionUser(ctx, "live", now); !exp.Equal(later) {
		t.Errorf("extended expiry = %v, want %v", exp, later)
	}
	// An expired session is not brought back.
	if err := s.ExtendWebSession(ctx, "old", later, now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.WebSessionUser(ctx, "old", now); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("extending an expired session revived it: %v", err)
	}

	n, err := s.DeleteExpiredWebSessions(ctx, now)
	if err != nil || n != 1 {
		t.Errorf("DeleteExpiredWebSessions = %d, %v, want 1", n, err)
	}

	if err := s.DeleteWebSession(ctx, "live"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.WebSessionUser(ctx, "live", now); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deleted session err = %v", err)
	}
	if err := s.DeleteWebSession(ctx, "live"); err != nil {
		t.Errorf("deleting twice: %v", err)
	}
}

// The table holds only the SHA-256 of the session id, never the id itself.
func TestWebSessionIDIsHashed(t *testing.T) {
	pool := dbtest.New(t)
	s := store.New(pool)
	ctx := t.Context()
	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada"})
	if err != nil {
		t.Fatal(err)
	}
	const id = "secret-session-id"
	if err := s.CreateWebSession(ctx, id, u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := pool.QueryRow(ctx, `SELECT id FROM web_sessions`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(id))
	if want := hex.EncodeToString(sum[:]); stored != want {
		t.Errorf("stored id = %q, want sha256 %q", stored, want)
	}

	// The stored value does not work as a cookie.
	if _, _, err := s.WebSessionUser(ctx, stored, time.Now()); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("looking up by the stored hash err = %v, want ErrNotFound", err)
	}
}
