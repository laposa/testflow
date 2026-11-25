package github

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// JWT timing (spec §9): iat is backdated to allow for clock drift, and GitHub
// rejects an exp more than 10 minutes ahead.
const (
	jwtBackdate = 60 * time.Second
	jwtLifetime = 9 * time.Minute
)

// tokenRefreshBefore is how long before expires_at a cached installation
// token is replaced (spec §9).
const tokenRefreshBefore = 5 * time.Minute

// ParsePrivateKey reads the GitHub App's RSA private key. GitHub issues PKCS#1
// ("RSA PRIVATE KEY") keys; PKCS#8 ("PRIVATE KEY") is accepted too.
func ParsePrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("github app key: no PEM block")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		k, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("github app key: %w", err)
		}
		return k, nil
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("github app key: %w", err)
		}
		rk, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("github app key: %T is not an RSA key", k)
		}
		return rk, nil
	default:
		return nil, fmt.Errorf("github app key: unexpected PEM type %q", block.Type)
	}
}

// AppJWT returns the RS256 JWT that authenticates as the GitHub App itself
// (spec §9): iat = now−60s, exp = now+9m, iss = the app's client id.
func AppJWT(clientID string, key *rsa.PrivateKey, now time.Time) (string, error) {
	header, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT"})
	claims, _ := json.Marshal(map[string]any{
		"iat": now.Add(-jwtBackdate).Unix(),
		"exp": now.Add(jwtLifetime).Unix(),
		"iss": clientID,
	})
	enc := base64.RawURLEncoding
	signed := enc.EncodeToString(header) + "." + enc.EncodeToString(claims)
	sum := sha256.Sum256([]byte(signed))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("sign app jwt: %w", err)
	}
	return signed + "." + enc.EncodeToString(sig), nil
}

// installationTokens exchanges the app JWT for an installation access token
// and caches it in memory (spec §3, §9). Callers that arrive while a refresh
// is running wait for it, so at most one exchange is in flight; a waiter
// gives up when its own context ends.
type installationTokens struct {
	clientID       string
	installationID int64
	key            *rsa.PrivateKey
	client         *Client // sends the exchange request (without a token)
	now            func() time.Time

	// lock is a one-slot semaphore guarding token and expires. Unlike a
	// sync.Mutex, acquiring it can be abandoned when ctx is done.
	lock    chan struct{}
	token   string
	expires time.Time
}

func newInstallationTokens(clientID string, installationID int64, key *rsa.PrivateKey, client *Client, now func() time.Time) *installationTokens {
	return &installationTokens{
		clientID:       clientID,
		installationID: installationID,
		key:            key,
		client:         client,
		now:            now,
		lock:           make(chan struct{}, 1),
	}
}

// Token returns a cached installation token, fetching a new one when none is
// cached or the cached one expires within tokenRefreshBefore.
//
// Errors are not wrapped with %w: a 404 here means the installation is wrong
// or gone, and must not look like ErrNotFound to the caller of, say, Dir.
func (s *installationTokens) Token(ctx context.Context) (string, error) {
	select {
	case s.lock <- struct{}{}:
	case <-ctx.Done():
		return "", fmt.Errorf("installation token: %w", context.Cause(ctx))
	}
	defer func() { <-s.lock }()
	now := s.now()
	if s.token != "" && now.Before(s.expires.Add(-tokenRefreshBefore)) {
		return s.token, nil
	}
	jwt, err := AppJWT(s.clientID, s.key, now)
	if err != nil {
		return "", err
	}
	var body struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}
	path := "/app/installations/" + strconv.FormatInt(s.installationID, 10) + "/access_tokens"
	err = s.client.do(ctx, request{method: http.MethodPost, path: path, bearer: jwt}, decodeJSON(&body))
	if err != nil {
		return "", fmt.Errorf("installation token: %v", err)
	}
	if body.Token == "" || body.ExpiresAt.IsZero() {
		return "", errors.New("installation token: response has no token or expires_at")
	}
	s.token, s.expires = body.Token, body.ExpiresAt
	return s.token, nil
}

// invalidate drops token from the cache if it is still the cached one, e.g.
// after GitHub answered 401 for it (revoked, or the clocks disagree).
func (s *installationTokens) invalidate(token string) {
	s.lock <- struct{}{}
	defer func() { <-s.lock }()
	if s.token == token {
		s.token, s.expires = "", time.Time{}
	}
}
