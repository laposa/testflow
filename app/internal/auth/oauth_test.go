package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeGitHub serves the OAuth token endpoint and the /user APIs.
func fakeGitHub(t *testing.T, userJSON, emailsJSON string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("token Accept = %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "application/json")
		if r.PostForm.Get("code") != "good" || r.PostForm.Get("client_secret") != "secret" ||
			r.PostForm.Get("redirect_uri") != "http://app/auth/callback" {
			// GitHub reports a bad code with 200 and an error field.
			w.Write([]byte(`{"error":"bad_verification_code","error_description":"The code passed is incorrect or expired."}`))
			return
		}
		w.Write([]byte(`{"access_token":"tok","token_type":"bearer","scope":"read:user,user:email"}`))
	})
	api := func(body string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer tok" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("X-GitHub-Api-Version") == "" || r.Header.Get("Accept") != "application/vnd.github+json" {
				t.Errorf("API headers = %v", r.Header)
			}
			if body == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Write([]byte(body))
		}
	}
	mux.HandleFunc("GET /user", api(userJSON))
	mux.HandleFunc("GET /user/emails", api(emailsJSON))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newOAuth(srv *httptest.Server) *GitHubOAuth {
	return &GitHubOAuth{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURL:  "http://app/auth/callback",
		TokenURL:     srv.URL + "/login/oauth/access_token",
		APIURL:       srv.URL,
		HTTPClient:   srv.Client(),
	}
}

func TestAuthCodeURL(t *testing.T) {
	o := &GitHubOAuth{ClientID: "cid", RedirectURL: "http://app/auth/callback"}
	u, err := url.Parse(o.AuthCodeURL("st"))
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Scheme + "://" + u.Host + u.Path; got != DefaultAuthURL {
		t.Errorf("auth URL = %s", got)
	}
	q := u.Query()
	for k, want := range map[string]string{
		"client_id": "cid", "redirect_uri": "http://app/auth/callback", "scope": "read:user user:email", "state": "st",
	} {
		if q.Get(k) != want {
			t.Errorf("%s = %q, want %q", k, q.Get(k), want)
		}
	}
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name, user, emails string
		code               string
		wantErr            string
		wantName, wantMail string
	}{
		{
			name: "public email", code: "good",
			user:     `{"id":7,"login":"ada","name":"Ada Lovelace","email":"ada@example.com"}`,
			wantName: "Ada Lovelace", wantMail: "ada@example.com",
		},
		{
			name: "private email from /user/emails", code: "good",
			user:     `{"id":7,"login":"ada","name":null,"email":null}`,
			emails:   `[{"email":"old@example.com","primary":false,"verified":true},{"email":"ada@example.com","primary":true,"verified":true}]`,
			wantName: "", wantMail: "ada@example.com",
		},
		{
			name: "unverified primary email ignored", code: "good",
			user:   `{"id":7,"login":"ada"}`,
			emails: `[{"email":"ada@example.com","primary":true,"verified":false}]`,
		},
		{
			name: "emails endpoint forbidden is not fatal", code: "good",
			user: `{"id":7,"login":"ada"}`,
		},
		{name: "bad code", code: "bad", user: `{"id":7,"login":"ada"}`, wantErr: "bad_verification_code"},
		{name: "user API fails", code: "good", wantErr: "GET /user: status 403"},
		{name: "user without id", code: "good", user: `{"login":"ada"}`, wantErr: "no id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := newOAuth(fakeGitHub(t, tt.user, tt.emails))
			gh, err := o.Authenticate(t.Context(), tt.code)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gh.ID != 7 || gh.Login != "ada" || gh.Name != tt.wantName || gh.Email != tt.wantMail {
				t.Errorf("user = %+v", gh)
			}
		})
	}
}

func TestNewToken(t *testing.T) {
	a, b := NewToken(), NewToken()
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil || len(raw) != 32 {
		t.Errorf("token %q: %d bytes, %v", a, len(raw), err)
	}
	if a == b {
		t.Error("tokens repeat")
	}
	if !EqualTokens(a, a) || EqualTokens(a, b) || EqualTokens("", "") {
		t.Error("EqualTokens wrong")
	}
}
