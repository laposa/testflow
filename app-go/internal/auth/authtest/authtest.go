// Package authtest provides an in-memory store for login and session tests
// that don't need Postgres.
package authtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
)

// Store implements auth.SessionStore and the user upsert used by the login
// handlers. It is safe for concurrent use.
type Store struct {
	mu       sync.Mutex
	Users    map[int64]store.User // by id
	sessions map[string]session
	nextID   int64
}

type session struct {
	userID  int64
	expires time.Time
}

// NewStore returns an empty Store.
func NewStore() *Store {
	return &Store{Users: map[int64]store.User{}, sessions: map[string]session{}}
}

// Add stores a session for userID and returns its id.
func (s *Store) Add(userID int64, expires time.Time) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	id := hex.EncodeToString(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = session{userID, expires}
	return id
}

// Has reports whether session id is stored (expired or not).
func (s *Store) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[id]
	return ok
}

// Expiry returns when session id expires.
func (s *Store) Expiry(id string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, ok := s.sessions[id]
	return ws.expires, ok
}

// Count returns the number of stored sessions.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

// UpsertUser creates or updates the user by GitHub id. Like store.Store, an
// empty gh.Email keeps the stored email.
func (s *Store) UpsertUser(_ context.Context, gh store.GitHubUser) (store.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, u := range s.Users {
		if u.GitHubID == gh.ID {
			if gh.Email == "" {
				gh.Email = u.Email
			}
			u.Name, u.Email, u.Login = gh.DisplayName(), gh.Email, gh.Login
			s.Users[id] = u
			return u, nil
		}
	}
	s.nextID++
	for s.Users[s.nextID].ID != 0 {
		s.nextID++
	}
	u := store.User{ID: s.nextID, GitHubID: gh.ID, Name: gh.DisplayName(), Email: gh.Email, Login: gh.Login}
	s.Users[u.ID] = u
	return u, nil
}

func (s *Store) CreateWebSession(_ context.Context, id string, userID int64, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = session{userID, expiresAt}
	return nil
}

func (s *Store) WebSessionUser(_ context.Context, id string, now time.Time) (store.User, time.Time, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, ok := s.sessions[id]
	if !ok || !ws.expires.After(now) {
		return store.User{}, time.Time{}, store.ErrNotFound
	}
	u, ok := s.Users[ws.userID]
	if !ok {
		return store.User{}, time.Time{}, store.ErrNotFound
	}
	return u, ws.expires, nil
}

func (s *Store) ExtendWebSession(_ context.Context, id string, expiresAt, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ws, ok := s.sessions[id]; ok && ws.expires.After(now) {
		ws.expires = expiresAt
		s.sessions[id] = ws
	}
	return nil
}

func (s *Store) DeleteWebSession(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *Store) DeleteExpiredWebSessions(_ context.Context, now time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for id, ws := range s.sessions {
		if !ws.expires.After(now) {
			delete(s.sessions, id)
			n++
		}
	}
	return n, nil
}
