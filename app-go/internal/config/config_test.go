package config

import (
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"
)

// testPEM is structurally valid PEM. Its content is not a real key; parsing the
// key itself is the job of internal/github.
const testPEM = "-----BEGIN RSA PRIVATE KEY-----\\nZmFrZQ==\\n-----END RSA PRIVATE KEY-----"

func validEnv() map[string]string {
	return map[string]string{
		"BASE_URL":                   "https://testflow.example.com/",
		"DATABASE_URL":               "postgres://u:p@localhost/testflow",
		"GITHUB_CLIENT_ID":           "Iv1.abc",
		"GITHUB_CLIENT_SECRET":       "secret",
		"GITHUB_APP_INSTALLATION_ID": "12345",
		"GITHUB_PRIVATE_KEY":         testPEM,
		"TEST_SESSION_ENVIRONMENTS":  " preprod, beta ,,prod ",
	}
}

func load(env map[string]string) (*Config, error) {
	return Load(func(k string) string { return env[k] })
}

func TestLoadDefaults(t *testing.T) {
	c, err := load(validEnv())
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8080" {
		t.Errorf("Addr = %q", c.Addr)
	}
	if got := c.BaseURL.String(); got != "https://testflow.example.com" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", got)
	}
	if !c.IsDevelopment() {
		t.Errorf("Environment = %q, want development", c.Environment)
	}
	if c.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v", c.LogLevel)
	}
	if c.SessionLifetime != 120*time.Minute {
		t.Errorf("SessionLifetime = %v", c.SessionLifetime)
	}
	if c.SyncInterval != 30*time.Second {
		t.Errorf("SyncInterval = %v", c.SyncInterval)
	}
	if c.GitHubInstallationID != 12345 {
		t.Errorf("GitHubInstallationID = %d", c.GitHubInstallationID)
	}
	if !strings.Contains(string(c.GitHubPrivateKey), "\nZmFrZQ==\n") {
		t.Errorf("GitHubPrivateKey escapes not expanded: %q", c.GitHubPrivateKey)
	}
	if want := []string{"preprod", "beta", "prod"}; !slices.Equal(c.TestSessionEnvironments, want) {
		t.Errorf("TestSessionEnvironments = %q, want %q", c.TestSessionEnvironments, want)
	}
	if c.Mail.Mailer != "log" {
		t.Errorf("Mail.Mailer = %q", c.Mail.Mailer)
	}
}

func TestLoadOverrides(t *testing.T) {
	env := validEnv()
	env["ADDR"] = ":9000"
	env["ENVIRONMENT"] = "production"
	env["LOG_LEVEL"] = "DEBUG"
	env["SESSION_LIFETIME"] = "8h"
	env["SYNC_INTERVAL"] = "1m"
	env["MAIL_MAILER"] = "smtp"
	env["SMTP_HOST"] = "smtp.example.com"
	env["SMTP_PORT"] = "587"
	env["SMTP_USERNAME"] = "user"
	env["SMTP_PASSWORD"] = "pass"
	env["MAIL_FROM"] = "testflow@example.com"

	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":9000" || c.IsDevelopment() || c.LogLevel != slog.LevelDebug ||
		c.SessionLifetime != 8*time.Hour || c.SyncInterval != time.Minute {
		t.Errorf("overrides not applied: %+v", c)
	}
	if c.Mail.Host != "smtp.example.com" || c.Mail.Port != 587 || c.Mail.From != "testflow@example.com" {
		t.Errorf("Mail = %+v", c.Mail)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name string
		edit func(map[string]string)
		want []string
	}{
		{"all required missing", func(e map[string]string) { clear(e) }, []string{
			"BASE_URL: is required", "DATABASE_URL: is required", "GITHUB_CLIENT_ID: is required",
			"GITHUB_CLIENT_SECRET: is required", "GITHUB_APP_INSTALLATION_ID: is required",
			"GITHUB_PRIVATE_KEY: is required", "TEST_SESSION_ENVIRONMENTS: is required",
		}},
		{"relative base url", func(e map[string]string) { e["BASE_URL"] = "/testflow" }, []string{"BASE_URL: must be an absolute"}},
		{"bad scheme", func(e map[string]string) { e["BASE_URL"] = "ftp://x" }, []string{"BASE_URL: must be an absolute"}},
		{"bad level", func(e map[string]string) { e["LOG_LEVEL"] = "loud" }, []string{"LOG_LEVEL"}},
		{"bad duration", func(e map[string]string) { e["SESSION_LIFETIME"] = "120" }, []string{"SESSION_LIFETIME"}},
		{"negative duration", func(e map[string]string) { e["SYNC_INTERVAL"] = "-1s" }, []string{"SYNC_INTERVAL"}},
		{"bad installation id", func(e map[string]string) { e["GITHUB_APP_INSTALLATION_ID"] = "abc" }, []string{"GITHUB_APP_INSTALLATION_ID: must be a positive integer"}},
		{"not pem", func(e map[string]string) { e["GITHUB_PRIVATE_KEY"] = "secret" }, []string{"GITHUB_PRIVATE_KEY: is not a PEM"}},
		{"only commas", func(e map[string]string) { e["TEST_SESSION_ENVIRONMENTS"] = " , ," }, []string{"TEST_SESSION_ENVIRONMENTS: is required"}},
		{"bad mailer", func(e map[string]string) { e["MAIL_MAILER"] = "sendmail" }, []string{"MAIL_MAILER"}},
		{"static dir outside development", func(e map[string]string) {
			e["ENVIRONMENT"], e["STATIC_DIR"] = "production", "internal/web/static"
		}, []string{"STATIC_DIR: is only allowed with ENVIRONMENT=development"}},
		{"bad mail from", func(e map[string]string) {
			e["MAIL_MAILER"], e["SMTP_HOST"], e["SMTP_PORT"], e["SMTP_USERNAME"], e["SMTP_PASSWORD"] = "smtp", "h", "587", "u", "p"
			e["MAIL_FROM"] = "TestFlow <not an address"
		}, []string{"MAIL_FROM: must be an email address"}},
		{"smtp incomplete", func(e map[string]string) { e["MAIL_MAILER"] = "smtp" }, []string{
			"SMTP_HOST: is required", "SMTP_PORT: is required", "SMTP_USERNAME: is required",
			"SMTP_PASSWORD: is required", "MAIL_FROM: is required",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validEnv()
			tt.edit(env)
			_, err := load(env)
			if err == nil {
				t.Fatal("expected an error")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q does not contain %q", err, w)
				}
			}
		})
	}
}

func TestLoadMigrate(t *testing.T) {
	c, err := loadMigrateEnv(map[string]string{"DATABASE_URL": "postgres://x", "LOG_LEVEL": "warn"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Database.URL != "postgres://x" || c.LogLevel != slog.LevelWarn || c.Environment != "development" {
		t.Errorf("got %+v", c)
	}
	if _, err := loadMigrateEnv(map[string]string{}); err == nil || !strings.Contains(err.Error(), "DATABASE_URL: is required") {
		t.Errorf("err = %v", err)
	}
}

func TestLoadDatabaseCredentials(t *testing.T) {
	env := validEnv()
	env["DATABASE_URL"] = "postgres://db.example.com:5432/testflow"
	env["DATABASE_USER"] = " testflow "
	env["DATABASE_PASSWORD"] = " p@ss:w/rd "
	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	want := Database{URL: "postgres://db.example.com:5432/testflow", User: "testflow", Password: " p@ss:w/rd "}
	if c.Database != want {
		t.Errorf("Database = %+v, want %+v (user trimmed, password kept as is)", c.Database, want)
	}
	if strings.Contains(c.LogValue().String(), "p@ss") {
		t.Error("LogValue leaks the database password")
	}

	m, err := loadMigrateEnv(env)
	if err != nil || m.Database != want {
		t.Errorf("LoadMigrate Database = %+v, %v", m.Database, err)
	}
}

func TestLoadDatabaseHost(t *testing.T) {
	env := validEnv()
	delete(env, "DATABASE_URL")
	env["DATABASE_HOST"] = " localhost "
	env["DATABASE_USER"] = "postgres"
	env["DATABASE_PASSWORD"] = "secret"
	c, err := load(env)
	if err != nil {
		t.Fatal(err)
	}
	want := Database{Host: "localhost", Port: 5432, Name: "testflow", User: "postgres", Password: "secret"}
	if c.Database != want {
		t.Errorf("Database = %+v, want %+v (defaults for port and name)", c.Database, want)
	}

	env["DATABASE_PORT"] = "5433"
	env["DATABASE_NAME"] = "other"
	if c, err = load(env); err != nil || c.Database.Port != 5433 || c.Database.Name != "other" {
		t.Errorf("Database = %+v, %v", c.Database, err)
	}

	for name, edit := range map[string]func(map[string]string){
		"DATABASE_PORT: must be a port number":   func(e map[string]string) { e["DATABASE_PORT"] = "abc" },
		"DATABASE_HOST: cannot be combined with": func(e map[string]string) { e["DATABASE_URL"] = "postgres://x" },
	} {
		e := validEnv()
		delete(e, "DATABASE_URL")
		e["DATABASE_HOST"] = "localhost"
		edit(e)
		if _, err := load(e); err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("err = %v, want %q", err, name)
		}
	}
}

func loadMigrateEnv(env map[string]string) (*MigrateConfig, error) {
	return LoadMigrate(func(k string) string { return env[k] })
}

func TestLogValueHasNoSecrets(t *testing.T) {
	c, err := load(validEnv())
	if err != nil {
		t.Fatal(err)
	}
	got := c.LogValue().String()
	for _, secret := range []string{"secret", "ZmFrZQ", "u:p@"} {
		if strings.Contains(got, secret) {
			t.Errorf("LogValue leaks %q: %s", secret, got)
		}
	}
}

func TestLoadStaticDir(t *testing.T) {
	env := validEnv()
	c, err := load(env)
	if err != nil || c.StaticDir != "" {
		t.Fatalf("default StaticDir = %q, %v; want empty", c.StaticDir, err)
	}
	env["STATIC_DIR"] = " internal/web/static "
	if c, err = load(env); err != nil || c.StaticDir != "internal/web/static" {
		t.Errorf("StaticDir = %q, %v", c.StaticDir, err)
	}
}
