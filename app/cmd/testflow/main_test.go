package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/config"
	"github.com/laposa/testflow/app-go/internal/db/dbtest"
)

// testKeyPEM is a real RSA key (serve parses it), generated once since
// generation is slow.
var testKeyPEM = sync.OnceValue(func() string {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}))
})

func testEnv(databaseURL string) map[string]string {
	return map[string]string{
		"ADDR":                       "127.0.0.1:0",
		"BASE_URL":                   "http://localhost:8080",
		"DATABASE_URL":               databaseURL,
		"GITHUB_CLIENT_ID":           "id",
		"GITHUB_CLIENT_SECRET":       "secret",
		"GITHUB_APP_INSTALLATION_ID": "1",
		"GITHUB_PRIVATE_KEY":         testKeyPEM(),
		"TEST_SESSION_ENVIRONMENTS":  "preprod",
	}
}

func TestRunCommands(t *testing.T) {
	tests := []struct {
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{nil, 2, "", "Usage: testflow"},
		{[]string{"bogus"}, 2, "", `unknown command "bogus"`},
		{[]string{"version"}, 0, "dev", ""},
		{[]string{"help"}, 0, "Commands:", ""},
		{[]string{"migrate"}, 1, "", "DATABASE_URL: is required"}, // empty environment
		{[]string{"serve"}, 1, "", "BASE_URL: is required"},
		{[]string{"serve", "-bogus"}, 2, "", "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), tt.args, func(string) string { return "" }, &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout = %q, want it to contain %q", stdout.String(), tt.wantStdout)
			}
			if !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}

func TestMigrateCommand(t *testing.T) {
	env := testEnv(dbtest.NewSchemaURL(t))
	getenv := func(k string) string { return env[k] }

	var stdout, stderr bytes.Buffer
	if code := run(t.Context(), []string{"migrate"}, getenv, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code %d: %s %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"migration":"0001_init.sql"`) {
		t.Errorf("applied migration not logged:\n%s", stdout.String())
	}

	stdout.Reset()
	if code := run(t.Context(), []string{"migrate"}, getenv, &stdout, &stderr); code != 0 {
		t.Fatalf("second run exit code %d", code)
	}
	if !strings.Contains(stdout.String(), `"applied":0`) {
		t.Errorf("second run should apply nothing:\n%s", stdout.String())
	}
}

func TestServeRefusesPendingMigrations(t *testing.T) {
	cfg := loadConfig(t, dbtest.NewSchemaURL(t)) // empty schema
	var logs bytes.Buffer
	err := serve(t.Context(), cfg, newLogger(&logs, cfg.LogLevel, cfg.Environment), false, nil)
	if err == nil || !strings.Contains(err.Error(), "pending migration") {
		t.Fatalf("serve error = %v, want pending migrations", err)
	}
}

// TestServeGracefulShutdown starts the real server with -migrate, makes a
// request and checks that cancelling the context stops it cleanly.
func TestServeGracefulShutdown(t *testing.T) {
	cfg := loadConfig(t, dbtest.NewSchemaURL(t))
	var logs bytes.Buffer
	logger := newLogger(&logs, cfg.LogLevel, cfg.Environment)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, logger, true, ready) }()

	var addr string
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("serve exited early: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("server did not start")
	}

	resp, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), `"ok"`) {
		t.Errorf("healthz = %d %s", resp.StatusCode, body)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}

	out := logs.String()
	for _, want := range []string{`"migration":"0001_init.sql"`, `"body":"server starting"`, `"body":"http request"`, `"body":"server stopped"`, `"service.version":"dev"`} {
		if !strings.Contains(out, want) {
			t.Errorf("logs missing %s", want)
		}
	}
	if strings.Contains(out, "secret") {
		t.Error("startup log leaks a secret")
	}
}

// The startup log line carries only allowlisted settings, never a secret.
func TestStartupLogHasNoSecrets(t *testing.T) {
	env := testEnv("postgres://dbuser:db-url-pass@db.example.com/testflow")
	secrets := map[string]string{
		"GITHUB_CLIENT_SECRET": "gh-client-secret",
		"DATABASE_PASSWORD":    "db-password",
		"SMTP_PASSWORD":        "smtp-password",
	}
	for k, v := range secrets {
		env[k] = v
	}
	env["ENVIRONMENT"] = "production"
	env["MAIL_MAILER"], env["SMTP_HOST"], env["SMTP_PORT"] = "smtp", "smtp.example.com", "587"
	env["SMTP_USERNAME"], env["MAIL_FROM"] = "smtp-user", "testflow@example.com"
	cfg, err := config.Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}

	var logs bytes.Buffer
	newLogger(&logs, cfg.LogLevel, cfg.Environment).Info("server starting", startupAttrs(cfg, "127.0.0.1:8080")...)
	out := logs.String()
	for _, want := range []string{`"addr":"127.0.0.1:8080"`, `"config":{`, `"environment":"production"`, `"mail_mailer":"smtp"`} {
		if !strings.Contains(out, want) {
			t.Errorf("startup log missing %s: %s", want, out)
		}
	}
	leaks := []string{"db-url-pass", "dbuser", "PRIVATE KEY", "smtp-user"}
	for _, v := range secrets {
		leaks = append(leaks, v)
	}
	for _, leak := range leaks {
		if strings.Contains(out, leak) {
			t.Errorf("startup log leaks %q: %s", leak, out)
		}
	}
}

func loadConfig(t *testing.T, databaseURL string) *config.Config {
	t.Helper()
	env := testEnv(databaseURL)
	cfg, err := config.Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// A PEM block that is not a usable RSA key passes config validation but stops
// serve before it touches the database.
func TestServeRejectsBadGitHubKey(t *testing.T) {
	cfg := loadConfig(t, "postgres://unused@127.0.0.1:1/none")
	cfg.GitHubPrivateKey = []byte("-----BEGIN RSA PRIVATE KEY-----\nZmFrZQ==\n-----END RSA PRIVATE KEY-----\n")
	err := serve(t.Context(), cfg, newLogger(io.Discard, cfg.LogLevel, cfg.Environment), false, nil)
	if err == nil || !strings.Contains(err.Error(), "github app key") {
		t.Fatalf("serve err = %v, want a github app key error", err)
	}
}
