package github

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testKey is generated once per test binary; 2048-bit generation is slow.
var testKey = sync.OnceValue(func() *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return k
})

func pkcs1PEM(k *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
}

func pkcs8PEM(t *testing.T, k any) []byte {
	t.Helper()
	b, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: b})
}

func TestParsePrivateKey(t *testing.T) {
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		pem     []byte
		wantErr string
	}{
		{"pkcs1", pkcs1PEM(testKey()), ""},
		{"pkcs8", pkcs8PEM(t, testKey()), ""},
		{"not pem", []byte("secret"), "no PEM block"},
		{"garbage pkcs1", []byte("-----BEGIN RSA PRIVATE KEY-----\nZmFrZQ==\n-----END RSA PRIVATE KEY-----\n"), "github app key"},
		{"ec key", pkcs8PEM(t, ec), "not an RSA key"},
		{"certificate", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("x")}), "unexpected PEM type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := ParsePrivateKey(tt.pem)
			if tt.wantErr == "" {
				if err != nil || !k.Equal(testKey()) {
					t.Fatalf("ParsePrivateKey = %v, %v", k != nil, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

// verifyJWT checks the RS256 signature and returns the claims.
func verifyJWT(t *testing.T, token string, pub *rsa.PublicKey) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt has %d parts", len(parts))
	}
	dec := func(s string) []byte {
		b, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			t.Fatalf("jwt segment: %v", err)
		}
		return b
	}
	var header map[string]string
	if err := json.Unmarshal(dec(parts[0]), &header); err != nil {
		t.Fatal(err)
	}
	if header["alg"] != "RS256" || header["typ"] != "JWT" {
		t.Errorf("header = %v", header)
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], dec(parts[2])); err != nil {
		t.Fatalf("signature: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(dec(parts[1]), &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

func TestAppJWT(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	token, err := AppJWT("Iv23liTestClient", testKey(), now)
	if err != nil {
		t.Fatal(err)
	}
	claims := verifyJWT(t, token, &testKey().PublicKey)
	if claims["iss"] != "Iv23liTestClient" {
		t.Errorf("iss = %v", claims["iss"])
	}
	if got := int64(claims["iat"].(float64)); got != now.Unix()-60 {
		t.Errorf("iat = %d, want now-60s", got)
	}
	if got := int64(claims["exp"].(float64)); got != now.Unix()+9*60 {
		t.Errorf("exp = %d, want now+9m", got)
	}
}

// tokenServer issues installation tokens "tok-1", "tok-2", ... that expire
// lifetime after the fake clock's current time.
type tokenServer struct {
	t        *testing.T
	clock    *fakeClock
	lifetime time.Duration
	issued   atomic.Int64
}

func (s *tokenServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/app/installations/42/access_tokens" {
		s.t.Errorf("token request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	jwt, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if claims := verifyJWT(s.t, jwt, &testKey().PublicKey); claims["iss"] != "cid" {
		s.t.Errorf("jwt iss = %v", claims["iss"])
	}
	n := s.issued.Add(1)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"token":"tok-%d","expires_at":%q,"repository_selection":"selected"}`,
		n, s.clock.Now().Add(s.lifetime).UTC().Format(time.RFC3339))
}

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func TestInstallationTokenCache(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)}
	ts := &tokenServer{t: t, clock: clock, lifetime: time.Hour}
	srv := httptest.NewServer(ts)
	defer srv.Close()
	c := newTestClient(t, srv, clock.Now)
	ctx := context.Background()

	get := func(want string) {
		t.Helper()
		tok, err := c.tokens.Token(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if tok != want {
			t.Fatalf("token = %q, want %q", tok, want)
		}
	}
	get("tok-1")
	get("tok-1") // cached
	clock.Advance(54 * time.Minute)
	get("tok-1") // 6 min left: still cached
	clock.Advance(2 * time.Minute)
	get("tok-2") // 4 min left: refreshed
	if n := ts.issued.Load(); n != 2 {
		t.Errorf("exchanges = %d, want 2", n)
	}

	c.tokens.invalidate("tok-1") // stale token: no effect
	get("tok-2")
	c.tokens.invalidate("tok-2")
	get("tok-3")
}

func TestInstallationTokenConcurrent(t *testing.T) {
	clock := &fakeClock{t: time.Now()}
	ts := &tokenServer{t: t, clock: clock, lifetime: time.Hour}
	srv := httptest.NewServer(ts)
	defer srv.Close()
	c := newTestClient(t, srv, clock.Now)

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if tok, err := c.tokens.Token(context.Background()); err != nil || tok != "tok-1" {
				t.Errorf("Token = %q, %v", tok, err)
			}
		})
	}
	wg.Wait()
	if n := ts.issued.Load(); n != 1 {
		t.Errorf("exchanges = %d, want 1", n)
	}
}

func TestInstallationTokenErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantErr string
	}{
		{"rejected", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"A JSON web token could not be decoded"}`))
		}, "status 401: A JSON web token could not be decoded"},
		{"no token", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{}`))
		}, "no token"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()
			c := newTestClient(t, srv, nil)
			_, err := c.tokens.Token(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func newTestClient(t *testing.T, srv *httptest.Server, now func() time.Time) *Client {
	t.Helper()
	c, err := New(Config{
		ClientID:       "cid",
		InstallationID: 42,
		PrivateKey:     pkcs1PEM(testKey()),
		APIURL:         srv.URL,
		HTTPClient:     srv.Client(),
		Now:            now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewValidates(t *testing.T) {
	key := pkcs1PEM(testKey())
	for name, cfg := range map[string]Config{
		"no client id":       {InstallationID: 1, PrivateKey: key},
		"no installation id": {ClientID: "cid", PrivateKey: key},
		"bad key":            {ClientID: "cid", InstallationID: 1, PrivateKey: []byte("nope")},
	} {
		if _, err := New(cfg); err == nil {
			t.Errorf("%s: New succeeded", name)
		}
	}
}
