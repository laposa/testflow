package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/laposa/testflow/app-go/internal/config"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	env := map[string]string{
		"ADDR":                       "127.0.0.1:0",
		"BASE_URL":                   "http://localhost:8080",
		"DATABASE_URL":               "postgres://localhost/testflow",
		"GITHUB_CLIENT_ID":           "id",
		"GITHUB_CLIENT_SECRET":       "secret",
		"GITHUB_APP_INSTALLATION_ID": "1",
		"GITHUB_PRIVATE_KEY":         "-----BEGIN RSA PRIVATE KEY-----\nZmFrZQ==\n-----END RSA PRIVATE KEY-----",
		"TEST_SESSION_ENVIRONMENTS":  "preprod",
	}
	cfg, err := config.Load(func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
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
		{[]string{"migrate"}, 1, "", "not available yet"},
		{[]string{"serve"}, 1, "", "BASE_URL: is required"}, // empty environment
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

// TestServeGracefulShutdown starts the real server, makes a request and
// checks that cancelling the context stops it cleanly.
func TestServeGracefulShutdown(t *testing.T) {
	cfg := testConfig(t)
	var logs bytes.Buffer
	logger := newLogger(&logs, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, logger, ready) }()

	var addr string
	select {
	case addr = <-ready:
	case err := <-done:
		t.Fatalf("serve exited early: %v", err)
	case <-time.After(5 * time.Second):
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
	for _, want := range []string{`"body":"server starting"`, `"body":"http request"`, `"body":"server stopped"`, `"service.version":"dev"`} {
		if !strings.Contains(out, want) {
			t.Errorf("logs missing %s", want)
		}
	}
	if strings.Contains(out, "secret") {
		t.Error("startup log leaks a secret")
	}
}
