package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/laposa/testflow/app-go/internal/auth"
)

func do(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func cookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// sameOrigin marks r as a same-origin browser request for the CSRF check.
func sameOrigin(r *http.Request) *http.Request {
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	return r
}

func TestPrivateRoutesNeedLogin(t *testing.T) {
	h, _ := newTestHandler(t, Options{})

	rec := get(h, "/")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Errorf("GET / = %d %q, want redirect to /login", rec.Code, rec.Header().Get("Location"))
	}

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("HX-Request", "true")
	rec = do(h, r)
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("HX-Redirect") != "/login" {
		t.Errorf("htmx GET / = %d HX-Redirect=%q", rec.Code, rec.Header().Get("HX-Redirect"))
	}

	// Public routes stay public.
	for _, target := range []string{"/login", "/healthz", "/static/app.css"} {
		if rec := get(h, target); rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, rec.Code)
		}
	}
}

func TestLoginPage(t *testing.T) {
	e := newEnv(t, Options{})

	rec := get(e.h, "/login")
	body := rec.Body.String()
	for _, want := range []string{`<form method="post" action="/auth/redirect">`, "Sign in with GitHub", "<title>Login · TestFlow</title>"} {
		if !strings.Contains(body, want) {
			t.Errorf("login page missing %q", want)
		}
	}
	if strings.Contains(body, "/auth/logout") {
		t.Error("login page shows logout")
	}

	if body := get(e.h, "/login?error=state").Body.String(); !strings.Contains(body, loginErrors["state"]) {
		t.Error("known error code not shown")
	}
	if body := get(e.h, "/login?error=<script>").Body.String(); strings.Contains(body, "script&gt;") || strings.Contains(body, `role="alert"`) {
		t.Error("unknown error code is echoed")
	}

	rec = get(e.h, "/login", e.signIn("Ada"))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Errorf("signed-in GET /login = %d %q, want redirect to /", rec.Code, rec.Header().Get("Location"))
	}
}

// startLogin posts to /auth/redirect and returns the state GitHub would
// receive together with the state cookie.
func startLogin(t *testing.T, h http.Handler) (string, *http.Cookie) {
	t.Helper()
	rec := do(h, sameOrigin(httptest.NewRequest(http.MethodPost, "/auth/redirect", nil)))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("POST /auth/redirect = %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || loc.Host != "github.test" {
		t.Fatalf("redirect to %q", rec.Header().Get("Location"))
	}
	c := cookie(rec, stateCookie)
	if c == nil || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/auth/callback" || c.MaxAge <= 0 {
		t.Fatalf("state cookie = %+v", c)
	}
	if loc.Query().Get("state") != c.Value {
		t.Fatalf("state in URL %q != cookie %q", loc.Query().Get("state"), c.Value)
	}
	return c.Value, c
}

func callback(h http.Handler, query string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/auth/callback?"+query, nil)
	// The browser arrives from github.com: a cross-site top-level navigation.
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	return do(h, r)
}

func TestLoginFlow(t *testing.T) {
	e := newEnv(t, Options{})
	state, stateC := startLogin(t, e.h)

	// An existing (e.g. pre-login) session id must be rotated.
	old := e.signIn("Someone")

	rec := callback(e.h, "code=good&state="+url.QueryEscape(state), stateC, old)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
		t.Fatalf("callback = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if c := cookie(rec, stateCookie); c == nil || c.MaxAge >= 0 {
		t.Error("state cookie not cleared")
	}
	sess := cookie(rec, auth.CookieName)
	if sess == nil || sess.Value == "" || sess.Value == old.Value {
		t.Fatalf("session cookie = %+v", sess)
	}
	if e.store.Has(old.Value) {
		t.Error("old session not deleted on login")
	}

	// The user was created from the GitHub profile, and the session works.
	rec = get(e.h, "/", sess)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Ada Lovelace") {
		t.Fatalf("home after login = %d", rec.Code)
	}
	if !strings.Contains(e.logs.String(), `"user.id":`) {
		t.Error("request log has no user id")
	}

	// Logout deletes the session and clears the cookie.
	r := sameOrigin(httptest.NewRequest(http.MethodPost, "/auth/logout", nil))
	r.AddCookie(sess)
	rec = do(e.h, r)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Errorf("logout = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if c := cookie(rec, auth.CookieName); c == nil || c.MaxAge >= 0 {
		t.Error("session cookie not cleared")
	}
	if e.store.Has(sess.Value) {
		t.Error("session not deleted on logout")
	}
	if rec := get(e.h, "/", sess); rec.Code != http.StatusSeeOther {
		t.Errorf("old cookie after logout = %d, want redirect", rec.Code)
	}
}

func TestLoginTwiceKeepsOneUser(t *testing.T) {
	e := newEnv(t, Options{})
	for range 2 {
		state, c := startLogin(t, e.h)
		if rec := callback(e.h, "code=good&state="+url.QueryEscape(state), c); rec.Code != http.StatusSeeOther {
			t.Fatal(rec.Code)
		}
	}
	if n := len(e.store.Users); n != 1 {
		t.Errorf("%d users after two logins, want 1", n)
	}
}

func TestCallbackFailures(t *testing.T) {
	e := newEnv(t, Options{})
	state, c := startLogin(t, e.h)
	q := func(code, st string) string { return "code=" + code + "&state=" + url.QueryEscape(st) }

	tests := []struct {
		name    string
		query   string
		cookies []*http.Cookie
		want    string
	}{
		{"user cancelled on GitHub", "error=access_denied&state=" + url.QueryEscape(state), []*http.Cookie{c}, "denied"},
		{"no state cookie", q("good", state), nil, "state"},
		{"state mismatch", q("good", state+"x"), []*http.Cookie{c}, "state"},
		{"empty state on both sides", q("good", ""), []*http.Cookie{{Name: stateCookie, Value: ""}}, "state"},
		{"missing code", "state=" + url.QueryEscape(state), []*http.Cookie{c}, "state"},
		{"GitHub rejects code", q("bad", state), []*http.Cookie{c}, "github"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := callback(e.h, tt.query, tt.cookies...)
			if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login?error="+tt.want {
				t.Errorf("got %d %q, want /login?error=%s", rec.Code, rec.Header().Get("Location"), tt.want)
			}
			if cookie(rec, auth.CookieName) != nil {
				t.Error("session cookie set on failure")
			}
		})
	}
	if e.store.Count() != 0 || len(e.store.Users) != 0 {
		t.Error("failed callbacks created users or sessions")
	}
}

func TestCrossOriginPostsRejected(t *testing.T) {
	e := newEnv(t, Options{BaseURL: &url.URL{Scheme: "https", Host: "testflow.example.com"}})
	sess := e.signIn("Ada")

	for _, target := range []string{"/auth/logout", "/auth/redirect"} {
		r := httptest.NewRequest(http.MethodPost, target, nil)
		r.Header.Set("Sec-Fetch-Site", "cross-site")
		r.AddCookie(sess)
		if rec := do(e.h, r); rec.Code != http.StatusForbidden {
			t.Errorf("cross-site POST %s = %d, want 403", target, rec.Code)
		}
	}
	if !e.store.Has(sess.Value) {
		t.Error("cross-site logout ended the session")
	}

	// Older browsers without Sec-Fetch-Site: Origin is checked against Host
	// and the trusted BASE_URL origin.
	for origin, want := range map[string]int{
		"https://evil.example":         http.StatusForbidden,
		"https://testflow.example.com": http.StatusSeeOther,
	} {
		r := httptest.NewRequest(http.MethodPost, "http://internal:8080/auth/redirect", nil)
		r.Header.Set("Origin", origin)
		if rec := do(e.h, r); rec.Code != want {
			t.Errorf("POST with Origin %s = %d, want %d", origin, rec.Code, want)
		}
	}
}

func TestLogoutIsPostOnly(t *testing.T) {
	h, _ := newTestHandler(t, Options{})
	if rec := get(h, "/auth/logout"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET /auth/logout = %d, want 405", rec.Code)
	}
}
