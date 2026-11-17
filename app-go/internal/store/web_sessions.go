package store

import (
	"context"
	"fmt"
	"time"
)

// CreateWebSession stores a login session (spec §6.2).
func (s *Store) CreateWebSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO web_sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		id, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("create web session: %w", err)
	}
	return nil
}

// WebSessionUser returns the user of the unexpired session id and when the
// session expires, or ErrNotFound.
func (s *Store) WebSessionUser(ctx context.Context, id string) (User, time.Time, error) {
	var (
		u         User
		expiresAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.github_id, u.name, COALESCE(u.email, ''), u.login, u.created_at, u.updated_at,
		       ws.expires_at
		FROM web_sessions ws JOIN users u ON u.id = ws.user_id
		WHERE ws.id = $1 AND ws.expires_at > now()`, id).
		Scan(&u.ID, &u.GitHubID, &u.Name, &u.Email, &u.Login, &u.CreatedAt, &u.UpdatedAt, &expiresAt)
	if err != nil {
		return User{}, time.Time{}, notFound(err)
	}
	return u, expiresAt, nil
}

// ExtendWebSession moves the expiry of an unexpired session.
func (s *Store) ExtendWebSession(ctx context.Context, id string, expiresAt time.Time) error {
	_, err := s.db.Exec(ctx,
		`UPDATE web_sessions SET expires_at = $2 WHERE id = $1 AND expires_at > now()`,
		id, expiresAt)
	if err != nil {
		return fmt.Errorf("extend web session: %w", err)
	}
	return nil
}

// DeleteWebSession removes a session. Deleting a missing one is not an error.
func (s *Store) DeleteWebSession(ctx context.Context, id string) error {
	if _, err := s.db.Exec(ctx, `DELETE FROM web_sessions WHERE id = $1`, id); err != nil {
		return fmt.Errorf("delete web session: %w", err)
	}
	return nil
}

// DeleteExpiredWebSessions removes expired sessions and returns how many.
func (s *Store) DeleteExpiredWebSessions(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `DELETE FROM web_sessions WHERE expires_at <= now()`)
	if err != nil {
		return 0, fmt.Errorf("delete expired web sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
