package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Session ids are stored only as their SHA-256 hash (spec §6.2), so someone
// who can read web_sessions (a backup, a read-only role) cannot use the rows
// to sign in. The id is 32 random bytes, so a fast unsalted hash is enough.
//
// The callers pass now instead of the queries using Postgres now(), so the
// app's clock alone decides expiry, the same clock that sets the cookie.

func hashSessionID(id string) string {
	sum := sha256.Sum256([]byte(id))
	return hex.EncodeToString(sum[:])
}

// CreateWebSession stores a login session (spec §6.2).
func (s *Store) CreateWebSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO web_sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashSessionID(id), userID, expiresAt)
	if err != nil {
		return fmt.Errorf("create web session: %w", err)
	}
	return nil
}

// WebSessionUser returns the user of session id if it has not expired at now,
// and when it expires, or ErrNotFound.
func (s *Store) WebSessionUser(ctx context.Context, id string, now time.Time) (User, time.Time, error) {
	var (
		u         User
		expiresAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.github_id, u.name, COALESCE(u.email, ''), u.login, u.created_at, u.updated_at,
		       ws.expires_at
		FROM web_sessions ws JOIN users u ON u.id = ws.user_id
		WHERE ws.id = $1 AND ws.expires_at > $2`, hashSessionID(id), now).
		Scan(&u.ID, &u.GitHubID, &u.Name, &u.Email, &u.Login, &u.CreatedAt, &u.UpdatedAt, &expiresAt)
	if err != nil {
		return User{}, time.Time{}, notFound(err)
	}
	return u, expiresAt, nil
}

// ExtendWebSession moves the expiry of a session that has not expired at now.
func (s *Store) ExtendWebSession(ctx context.Context, id string, expiresAt, now time.Time) error {
	_, err := s.db.Exec(ctx,
		`UPDATE web_sessions SET expires_at = $2 WHERE id = $1 AND expires_at > $3`,
		hashSessionID(id), expiresAt, now)
	if err != nil {
		return fmt.Errorf("extend web session: %w", err)
	}
	return nil
}

// DeleteWebSession removes a session. Deleting a missing one is not an error.
func (s *Store) DeleteWebSession(ctx context.Context, id string) error {
	if _, err := s.db.Exec(ctx, `DELETE FROM web_sessions WHERE id = $1`, hashSessionID(id)); err != nil {
		return fmt.Errorf("delete web session: %w", err)
	}
	return nil
}

// DeleteExpiredWebSessions removes sessions expired at now and returns how many.
func (s *Store) DeleteExpiredWebSessions(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM web_sessions WHERE expires_at <= $1`, now)
	if err != nil {
		return 0, fmt.Errorf("delete expired web sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
