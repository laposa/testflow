package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/telemetry"
)

// CookieName is the session cookie (spec §9).
const CookieName = "testflow_session"

// SessionStore is the part of store.Store the session manager needs.
type SessionStore interface {
	CreateWebSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error
	WebSessionUser(ctx context.Context, id string, now time.Time) (store.User, time.Time, error)
	ExtendWebSession(ctx context.Context, id string, expiresAt, now time.Time) error
	DeleteWebSession(ctx context.Context, id string) error
	DeleteExpiredWebSessions(ctx context.Context, now time.Time) (int64, error)
}

// Sessions manages server-side login sessions in web_sessions (spec §6.2, §9).
//
// A session lasts Lifetime since the last request (like Laravel's
// SESSION_LIFETIME). To avoid a write on every request, the expiry is moved
// forward only once less than half of Lifetime is left.
type Sessions struct {
	Store    SessionStore
	Lifetime time.Duration
	Secure   bool // Secure cookie flag; false only for ENVIRONMENT=development
	Logger   *slog.Logger
	Now      func() time.Time // nil means time.Now
}

func (s *Sessions) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Start creates a session for userID and sets its cookie. A session the
// request already had is deleted first, so the id is rotated on login.
func (s *Sessions) Start(w http.ResponseWriter, r *http.Request, userID int64) error {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		if err := s.Store.DeleteWebSession(r.Context(), c.Value); err != nil {
			return err
		}
	}
	id := NewToken()
	expires := s.now().Add(s.Lifetime)
	if err := s.Store.CreateWebSession(r.Context(), id, userID, expires); err != nil {
		return err
	}
	s.setCookie(w, id, expires)
	return nil
}

// End deletes the request's session, if any, and clears the cookie.
func (s *Sessions) End(w http.ResponseWriter, r *http.Request) error {
	s.clearCookie(w)
	c, err := r.Cookie(CookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	return s.Store.DeleteWebSession(r.Context(), c.Value)
}

// Lookup returns the signed-in user of the request. ok is false when there
// is no valid session; err is set only when the store failed.
func (s *Sessions) Lookup(w http.ResponseWriter, r *http.Request) (u store.User, ok bool, err error) {
	c, cerr := r.Cookie(CookieName)
	if cerr != nil || c.Value == "" {
		return store.User{}, false, nil
	}
	now := s.now()
	u, expires, err := s.Store.WebSessionUser(r.Context(), c.Value, now)
	if errors.Is(err, store.ErrNotFound) {
		s.clearCookie(w)
		return store.User{}, false, nil
	}
	if err != nil {
		return store.User{}, false, err
	}
	if expires.Sub(now) < s.Lifetime/2 {
		expires = now.Add(s.Lifetime)
		if err := s.Store.ExtendWebSession(r.Context(), c.Value, expires, now); err != nil {
			// The session is still valid; extending can wait for the next request.
			s.logger().WarnContext(r.Context(), "extend web session failed", "err", err)
		} else {
			s.setCookie(w, c.Value, expires)
		}
	}
	return u, true, nil
}

// Require lets only signed-in users through (spec §7). Others are sent to
// /login: with a redirect, or for htmx requests with 401 and HX-Redirect.
// The user is available to next through UserFrom.
func (s *Sessions) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok, err := s.Lookup(w, r)
		if err != nil {
			s.logger().ErrorContext(r.Context(), "session lookup failed", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !ok {
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		telemetry.SetUserID(r.Context(), u.ID)
		// Same *http.Request pointer, so the telemetry middleware still sees
		// the matched route pattern.
		*r = *r.WithContext(WithUser(r.Context(), u))
		next.ServeHTTP(w, r)
	})
}

// DeleteExpired removes expired sessions. The syncer loop calls it (§8.5).
func (s *Sessions) DeleteExpired(ctx context.Context) (int64, error) {
	return s.Store.DeleteExpiredWebSessions(ctx, s.now())
}

func (s *Sessions) setCookie(w http.ResponseWriter, id string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    id,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(expires.Sub(s.now()).Seconds()),
		HttpOnly: true,
		Secure:   s.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Sessions) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Sessions) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

type userKey struct{}

// WithUser returns ctx carrying the signed-in user.
func WithUser(ctx context.Context, u store.User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

// UserFrom returns the signed-in user set by Require.
func UserFrom(ctx context.Context) (store.User, bool) {
	u, ok := ctx.Value(userKey{}).(store.User)
	return u, ok
}
