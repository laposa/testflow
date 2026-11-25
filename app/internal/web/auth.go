package web

import (
	"context"
	"net/http"
	"time"

	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/web/views"
)

// OAuthProvider is the GitHub login (auth.GitHubOAuth); tests use a fake.
type OAuthProvider interface {
	AuthCodeURL(state string) string
	Authenticate(ctx context.Context, code string) (store.GitHubUser, error)
}

// UserStore creates and updates users on login.
type UserStore interface {
	UpsertUser(ctx context.Context, gh store.GitHubUser) (store.User, error)
}

// stateCookie holds the OAuth state between POST /auth/redirect and
// GET /auth/callback (spec §9). SameSite=Lax still sends it on the top-level
// redirect back from GitHub.
const (
	stateCookie   = "testflow_oauth_state"
	stateCookieTo = "/auth/callback"
	stateLifetime = 10 * time.Minute
)

// loginErrors maps the ?error= codes /login accepts to fixed messages, so
// the page never echoes text from the URL.
var loginErrors = map[string]string{
	"denied": "GitHub sign-in was cancelled.",
	"state":  "Your sign-in attempt expired or was invalid. Please try again.",
	"github": "Could not sign in with GitHub. Please try again.",
}

// login shows the "Sign in with GitHub" page, or sends signed-in users home
// (legacy: AuthController::login).
func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if _, ok, _ := s.Sessions.Lookup(w, r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, r, http.StatusOK, views.LoginPage(s.page(r, "Login"), loginErrors[r.URL.Query().Get("error")]))
}

// authRedirect starts the OAuth flow (legacy: AuthController::redirect).
func (s *server) authRedirect(w http.ResponseWriter, r *http.Request) {
	state := auth.NewToken()
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookie,
		Value:    state,
		Path:     stateCookieTo,
		MaxAge:   int(stateLifetime.Seconds()),
		HttpOnly: true,
		Secure:   s.Sessions.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, s.OAuth.AuthCodeURL(state), http.StatusSeeOther)
}

// authCallback finishes the OAuth flow: it checks state, reads the GitHub
// profile, upserts the user and starts a new session
// (legacy: AuthController::callback).
func (s *server) authCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	var state string
	if c, err := r.Cookie(stateCookie); err == nil {
		state = c.Value
	}
	// The state is single-use.
	http.SetCookie(w, &http.Cookie{
		Name: stateCookie, Path: stateCookieTo, MaxAge: -1,
		HttpOnly: true, Secure: s.Sessions.Secure, SameSite: http.SameSiteLaxMode,
	})

	fail := func(code string) { http.Redirect(w, r, "/login?error="+code, http.StatusSeeOther) }

	if e := q.Get("error"); e != "" {
		s.Logger.InfoContext(ctx, "github sign-in not completed", "error", e)
		fail("denied")
		return
	}
	if !auth.EqualTokens(state, q.Get("state")) || q.Get("code") == "" {
		s.Logger.WarnContext(ctx, "oauth callback with missing or wrong state")
		fail("state")
		return
	}
	gh, err := s.OAuth.Authenticate(ctx, q.Get("code"))
	if err != nil {
		s.Logger.ErrorContext(ctx, "github sign-in failed", "err", err)
		fail("github")
		return
	}
	u, err := s.Users.UpsertUser(ctx, gh)
	if err != nil {
		s.serverError(w, r, "save user", err)
		return
	}
	if err := s.Sessions.Start(w, r, u.ID); err != nil {
		s.serverError(w, r, "start session", err)
		return
	}
	s.Logger.InfoContext(ctx, "user signed in", "user.id", u.ID, "user.login", u.Login)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// logout ends the session (legacy: AuthController::logout, which was a GET).
func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.Sessions.End(w, r); err != nil {
		s.Logger.ErrorContext(r.Context(), "end session failed", "err", err)
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *server) serverError(w http.ResponseWriter, r *http.Request, what string, err error) {
	s.Logger.ErrorContext(r.Context(), what+" failed", "err", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}
