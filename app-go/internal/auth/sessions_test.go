package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth/authtest"
	"github.com/laposa/testflow/app-go/internal/store"
)

var t0 = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

func newSessions(st *authtest.Store, now *time.Time) *Sessions {
	return &Sessions{Store: st, Lifetime: 2 * time.Hour, Secure: true, Now: func() time.Time { return *now }}
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == CookieName {
			return c
		}
	}
	return nil
}

func TestStartSetsCookieAndRotates(t *testing.T) {
	now := t0
	st := authtest.NewStore(func() time.Time { return now })
	s := newSessions(st, &now)

	rec := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/auth/callback", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: st.Add(1, now.Add(time.Hour))})
	old := r.Cookies()[0].Value
	if err := s.Start(rec, r, 1); err != nil {
		t.Fatal(err)
	}

	c := sessionCookie(t, rec)
	if c == nil {
		t.Fatal("no session cookie set")
	}
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.MaxAge != 7200 {
		t.Errorf("cookie attributes = %+v", c)
	}
	if c.Value == old {
		t.Error("session id not rotated")
	}
	if st.Has(old) {
		t.Error("old session still stored")
	}
	if exp, ok := st.Expiry(c.Value); !ok || !exp.Equal(now.Add(2*time.Hour)) {
		t.Errorf("new session expiry = %v, %v", exp, ok)
	}
}

func TestRequire(t *testing.T) {
	now := t0
	st := authtest.NewStore(func() time.Time { return now })
	st.Users[1] = store.User{ID: 1, Name: "Ada"}
	s := newSessions(st, &now)

	var seen store.User
	h := s.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen, _ = UserFrom(r.Context())
	}))

	call := func(cookie string, htmx bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/sessions", nil)
		if cookie != "" {
			r.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
		}
		if htmx {
			r.Header.Set("HX-Request", "true")
		}
		rec := httptest.NewRecorder()
		seen = store.User{}
		h.ServeHTTP(rec, r)
		return rec
	}

	t.Run("no cookie redirects", func(t *testing.T) {
		rec := call("", false)
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
			t.Errorf("got %d %s", rec.Code, rec.Header().Get("Location"))
		}
	})
	t.Run("htmx gets 401 with HX-Redirect", func(t *testing.T) {
		rec := call("", true)
		if rec.Code != http.StatusUnauthorized || rec.Header().Get("HX-Redirect") != "/login" {
			t.Errorf("got %d HX-Redirect=%q", rec.Code, rec.Header().Get("HX-Redirect"))
		}
	})
	t.Run("unknown session clears cookie", func(t *testing.T) {
		rec := call("bogus", false)
		c := sessionCookie(t, rec)
		if rec.Code != http.StatusSeeOther || c == nil || c.MaxAge >= 0 {
			t.Errorf("got %d, cookie %+v", rec.Code, c)
		}
	})
	t.Run("expired session", func(t *testing.T) {
		id := st.Add(1, now.Add(-time.Second))
		if rec := call(id, false); rec.Code != http.StatusSeeOther {
			t.Errorf("got %d", rec.Code)
		}
	})
	t.Run("valid session passes the user on", func(t *testing.T) {
		id := st.Add(1, now.Add(2*time.Hour))
		rec := call(id, false)
		if rec.Code != http.StatusOK || seen.Name != "Ada" {
			t.Errorf("got %d, user %+v", rec.Code, seen)
		}
		if sessionCookie(t, rec) != nil {
			t.Error("fresh session should not be re-sent")
		}
	})
	t.Run("session past half its lifetime is extended", func(t *testing.T) {
		id := st.Add(1, now.Add(30*time.Minute))
		rec := call(id, false)
		c := sessionCookie(t, rec)
		if rec.Code != http.StatusOK || c == nil || c.Value != id {
			t.Fatalf("got %d, cookie %+v", rec.Code, c)
		}
		if exp, _ := st.Expiry(id); !exp.Equal(now.Add(2 * time.Hour)) {
			t.Errorf("expiry = %v", exp)
		}
	})
}

func TestEnd(t *testing.T) {
	now := t0
	st := authtest.NewStore(func() time.Time { return now })
	s := newSessions(st, &now)
	id := st.Add(1, now.Add(time.Hour))

	r := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	r.AddCookie(&http.Cookie{Name: CookieName, Value: id})
	rec := httptest.NewRecorder()
	if err := s.End(rec, r); err != nil {
		t.Fatal(err)
	}
	if st.Has(id) {
		t.Error("session not deleted")
	}
	if c := sessionCookie(t, rec); c == nil || c.MaxAge >= 0 {
		t.Errorf("cookie not cleared: %+v", c)
	}
	// Without a cookie, End only clears it.
	if err := s.End(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil)); err != nil {
		t.Error(err)
	}
}

func TestDeleteExpired(t *testing.T) {
	now := t0
	st := authtest.NewStore(func() time.Time { return now })
	s := newSessions(st, &now)
	st.Add(1, now.Add(-time.Minute))
	keep := st.Add(1, now.Add(time.Minute))
	n, err := s.DeleteExpired(t.Context())
	if err != nil || n != 1 || !st.Has(keep) {
		t.Errorf("DeleteExpired = %d, %v", n, err)
	}
}
