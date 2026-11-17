package store_test

import (
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

	// Same GitHub id: the row is updated, not duplicated. A private email
	// is stored as NULL and read back as "".
	u2, err := s.UpsertUser(ctx, store.GitHubUser{ID: 42, Login: "ada2", Name: "Ada Lovelace"})
	if err != nil {
		t.Fatal(err)
	}
	if u2.ID != u.ID || u2.Name != "Ada Lovelace" || u2.Email != "" || u2.Login != "ada2" {
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

func TestWebSessions(t *testing.T) {
	s := store.New(dbtest.New(t))
	ctx := t.Context()
	u, err := s.UpsertUser(ctx, store.GitHubUser{ID: 1, Login: "ada", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}

	future := time.Now().Add(time.Hour).Truncate(time.Microsecond)
	if err := s.CreateWebSession(ctx, "live", u.ID, future); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateWebSession(ctx, "old", u.ID, time.Now().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, exp, err := s.WebSessionUser(ctx, "live")
	if err != nil || got.ID != u.ID || got.Name != "Ada" || !exp.Equal(future) {
		t.Errorf("WebSessionUser(live) = %+v, %v, %v", got, exp, err)
	}
	if _, _, err := s.WebSessionUser(ctx, "old"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expired session err = %v, want ErrNotFound", err)
	}
	if _, _, err := s.WebSessionUser(ctx, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing session err = %v, want ErrNotFound", err)
	}

	later := future.Add(time.Hour)
	if err := s.ExtendWebSession(ctx, "live", later); err != nil {
		t.Fatal(err)
	}
	if _, exp, _ := s.WebSessionUser(ctx, "live"); !exp.Equal(later) {
		t.Errorf("extended expiry = %v, want %v", exp, later)
	}

	n, err := s.DeleteExpiredWebSessions(ctx)
	if err != nil || n != 1 {
		t.Errorf("DeleteExpiredWebSessions = %d, %v, want 1", n, err)
	}

	if err := s.DeleteWebSession(ctx, "live"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.WebSessionUser(ctx, "live"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("deleted session err = %v", err)
	}
	if err := s.DeleteWebSession(ctx, "live"); err != nil {
		t.Errorf("deleting twice: %v", err)
	}
}
