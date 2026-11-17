// Package auth implements the GitHub OAuth login flow, server-side web
// sessions and the authentication middleware (spec §7, §9).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/laposa/testflow/app-go/internal/store"
)

// Scope is the OAuth scope requested from GitHub (spec §9).
const Scope = "read:user user:email"

// GitHub's OAuth web-flow and REST endpoints.
const (
	DefaultAuthURL  = "https://github.com/login/oauth/authorize"
	DefaultTokenURL = "https://github.com/login/oauth/access_token"
	DefaultAPIURL   = "https://api.github.com"
)

// apiVersion is sent as X-GitHub-Api-Version.
const apiVersion = "2022-11-28"

// requestTimeout bounds each call to GitHub.
const requestTimeout = 10 * time.Second

// GitHubOAuth runs GitHub's OAuth web flow by hand with net/http (spec §9).
// The user's access token is used only to read the profile and is then
// dropped (D14).
type GitHubOAuth struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string // BASE_URL + /auth/callback

	// Endpoints; empty means GitHub's. Tests point them at httptest servers.
	AuthURL  string
	TokenURL string
	APIURL   string

	HTTPClient *http.Client // nil means a client with no extra settings
}

// AuthCodeURL is the GitHub page the browser is sent to for the given state.
func (o *GitHubOAuth) AuthCodeURL(state string) string {
	q := url.Values{
		"client_id":    {o.ClientID},
		"redirect_uri": {o.RedirectURL},
		"scope":        {Scope},
		"state":        {state},
		"allow_signup": {"false"},
	}
	return or(o.AuthURL, DefaultAuthURL) + "?" + q.Encode()
}

// Authenticate exchanges an authorization code for a token and reads the
// user's profile with it.
func (o *GitHubOAuth) Authenticate(ctx context.Context, code string) (store.GitHubUser, error) {
	token, err := o.exchange(ctx, code)
	if err != nil {
		return store.GitHubUser{}, err
	}
	return o.user(ctx, token)
}

func (o *GitHubOAuth) exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"client_id":     {o.ClientID},
		"client_secret": {o.ClientSecret},
		"code":          {code},
		"redirect_uri":  {o.RedirectURL},
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, or(o.TokenURL, DefaultTokenURL), strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// GitHub answers 200 with an "error" field for a bad or expired code.
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := o.do(req, &body); err != nil {
		return "", fmt.Errorf("oauth token exchange: %w", err)
	}
	if body.Error != "" {
		return "", fmt.Errorf("oauth token exchange: %s: %s", body.Error, body.Description)
	}
	if body.AccessToken == "" {
		return "", errors.New("oauth token exchange: no access token in response")
	}
	return body.AccessToken, nil
}

func (o *GitHubOAuth) user(ctx context.Context, token string) (store.GitHubUser, error) {
	var u struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := o.api(ctx, token, "/user", &u); err != nil {
		return store.GitHubUser{}, err
	}
	if u.ID <= 0 || u.Login == "" {
		return store.GitHubUser{}, errors.New("GET /user: response has no id or login")
	}
	gh := store.GitHubUser{ID: u.ID, Login: u.Login, Name: u.Name, Email: u.Email}

	// /user shows only the public email. The primary one comes from
	// /user/emails. A failure there is not fatal: email is optional.
	if gh.Email == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := o.api(ctx, token, "/user/emails", &emails); err == nil {
			for _, e := range emails {
				if e.Primary && e.Verified {
					gh.Email = e.Email
					break
				}
			}
		}
	}
	return gh, nil
}

func (o *GitHubOAuth) api(ctx context.Context, token, path string, v any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, or(o.APIURL, DefaultAPIURL)+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if err := o.do(req, v); err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	return nil
}

// do sends req and decodes a 2xx JSON response into v.
func (o *GitHubOAuth) do(req *http.Request, v any) error {
	client := o.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body := io.LimitReader(resp.Body, 1<<20)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(body).Decode(v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// NewToken returns 32 random bytes, base64url-encoded without padding. It is
// used for OAuth state values and web session ids (spec §6.2, §9).
func NewToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read never fails (Go 1.24+)
	return base64.RawURLEncoding.EncodeToString(b)
}

// EqualTokens compares two tokens in constant time.
func EqualTokens(a, b string) bool {
	return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func or(v, def string) string {
	if v != "" {
		return v
	}
	return def
}
