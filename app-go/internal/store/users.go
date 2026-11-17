package store

import (
	"context"
	"fmt"
	"time"
)

// User is a row of users (spec §6.1).
type User struct {
	ID        int64
	GitHubID  int64
	Name      string
	Email     string // empty when private on GitHub
	Login     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GitHubUser is what a GitHub login knows about a user.
type GitHubUser struct {
	ID    int64
	Login string
	Name  string
	Email string
}

// DisplayName is the name stored for u: the GitHub name, falling back to the
// email, then the login (legacy: AuthController::callback).
func (u GitHubUser) DisplayName() string {
	switch {
	case u.Name != "":
		return u.Name
	case u.Email != "":
		return u.Email
	default:
		return u.Login
	}
}

const userColumns = `id, github_id, name, COALESCE(email, ''), login, created_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.GitHubID, &u.Name, &u.Email, &u.Login, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}

// UpsertUser creates the user with gh.ID or updates its name, email and
// login, and returns the stored row.
//
// An empty gh.Email keeps the stored email: GitHub always has a verified
// primary email, so empty only means it could not be read this time (e.g.
// /user/emails failed). The name follows DisplayName with that email.
func (s *Store) UpsertUser(ctx context.Context, gh GitHubUser) (User, error) {
	if gh.ID <= 0 || gh.Login == "" {
		return User{}, fmt.Errorf("upsert user: GitHub id and login are required")
	}
	u, err := scanUser(s.db.QueryRow(ctx, `
		INSERT INTO users (github_id, name, email, login)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		ON CONFLICT (github_id) DO UPDATE
		SET email = COALESCE(EXCLUDED.email, users.email),
		    name  = COALESCE(NULLIF($5, ''), EXCLUDED.email, users.email, EXCLUDED.login),
		    login = EXCLUDED.login
		RETURNING `+userColumns,
		gh.ID, gh.DisplayName(), gh.Email, gh.Login, gh.Name))
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

// User returns the user with id, or ErrNotFound.
func (s *Store) User(ctx context.Context, id int64) (User, error) {
	u, err := scanUser(s.db.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		return User{}, notFound(err)
	}
	return u, nil
}
