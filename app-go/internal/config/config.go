// Package config reads and validates the environment configuration (spec §5).
package config

import (
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config is the validated runtime configuration. It is built once at startup
// by Load and passed down explicitly; nothing reads the environment later.
type Config struct {
	Addr        string
	BaseURL     *url.URL
	Environment string
	LogLevel    slog.Level
	// StaticDir serves /static/ from this directory instead of the embedded
	// files, so CSS and JS edits show on reload. Development only.
	StaticDir       string
	Database        Database
	SessionLifetime time.Duration

	GitHubClientID       string
	GitHubClientSecret   string
	GitHubInstallationID int64
	GitHubPrivateKey     []byte // PEM

	TestSessionEnvironments []string
	SyncInterval            time.Duration

	Mail MailConfig
}

// Database holds the PostgreSQL connection settings. Either URL or Host is
// set, never both. With Host, Port and Name are optional (5432, "testflow").
// User and Password are optional; when set they override any credentials in
// the URL, so the password can be kept out of the URL (no URL-encoding needed).
type Database struct {
	URL      string
	Host     string
	Port     int
	Name     string
	User     string
	Password string
}

// MailConfig selects and configures the mailer (spec §8.10).
type MailConfig struct {
	Mailer   string // "log" or "smtp"
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// AllowPlaintext lets the SMTP mailer send without STARTTLS, for a local
	// test server such as Mailpit. Development and a loopback host only.
	AllowPlaintext bool
}

// IsDevelopment reports whether ENVIRONMENT is "development".
func (c *Config) IsDevelopment() bool { return c.Environment == "development" }

// Load builds a Config from getenv (normally os.Getenv). All problems are
// collected and returned together so a misconfigured deployment can be fixed
// in one go.
func Load(getenv func(string) string) (*Config, error) {
	p := parser{getenv: getenv}
	c := &Config{
		Addr:            p.str("ADDR", ":8080"),
		BaseURL:         p.baseURL("BASE_URL"),
		Environment:     p.str("ENVIRONMENT", "development"),
		LogLevel:        p.level("LOG_LEVEL", slog.LevelInfo),
		Database:        p.database(),
		SessionLifetime: p.duration("SESSION_LIFETIME", 120*time.Minute),

		GitHubClientID:       p.required("GITHUB_CLIENT_ID"),
		GitHubClientSecret:   p.required("GITHUB_CLIENT_SECRET"),
		GitHubInstallationID: p.positiveInt("GITHUB_APP_INSTALLATION_ID"),
		GitHubPrivateKey:     p.pem("GITHUB_PRIVATE_KEY"),

		TestSessionEnvironments: p.list("TEST_SESSION_ENVIRONMENTS"),
		SyncInterval:            p.duration("SYNC_INTERVAL", 30*time.Second),
	}

	if c.StaticDir = p.str("STATIC_DIR", ""); c.StaticDir != "" && !c.IsDevelopment() {
		p.fail("STATIC_DIR", errors.New("is only allowed with ENVIRONMENT=development"))
	}

	// Outside development the mailer must be chosen explicitly, so a deploy
	// that forgets it does not silently log review emails instead of
	// sending them.
	if c.IsDevelopment() {
		c.Mail.Mailer = p.str("MAIL_MAILER", "log")
	} else {
		c.Mail.Mailer = p.required("MAIL_MAILER")
	}
	switch c.Mail.Mailer {
	case "": // reported by required
	case "log":
	case "smtp":
		c.Mail.Host = p.required("SMTP_HOST")
		c.Mail.Port = int(p.positiveInt("SMTP_PORT"))
		c.Mail.Username = p.required("SMTP_USERNAME")
		c.Mail.Password = p.required("SMTP_PASSWORD")
		if c.Mail.From = p.required("MAIL_FROM"); c.Mail.From != "" {
			if _, err := mail.ParseAddress(c.Mail.From); err != nil {
				p.fail("MAIL_FROM", fmt.Errorf("must be an email address such as TestFlow <testflow@example.com>: %w", err))
			}
		}
		c.Mail.AllowPlaintext = p.boolean("SMTP_ALLOW_PLAINTEXT")
		switch {
		case !c.Mail.AllowPlaintext:
		case !c.IsDevelopment():
			p.fail("SMTP_ALLOW_PLAINTEXT", errors.New("is only allowed with ENVIRONMENT=development"))
		case !isLoopback(c.Mail.Host):
			p.fail("SMTP_ALLOW_PLAINTEXT", fmt.Errorf("is only allowed with a local SMTP_HOST (localhost, 127.0.0.1 or ::1), got %q", c.Mail.Host))
		}
	default:
		p.fail("MAIL_MAILER", fmt.Errorf("must be \"log\" or \"smtp\", got %q", c.Mail.Mailer))
	}

	if err := errors.Join(p.errs...); err != nil {
		return nil, fmt.Errorf("invalid configuration:\n%w", err)
	}
	return c, nil
}

// MigrateConfig is the subset of settings `testflow migrate` needs, so it can
// run (e.g. as a deploy step) without the GitHub and mail secrets.
type MigrateConfig struct {
	Database    Database
	Environment string
	LogLevel    slog.Level
}

// LoadMigrate builds a MigrateConfig from getenv.
func LoadMigrate(getenv func(string) string) (*MigrateConfig, error) {
	p := parser{getenv: getenv}
	c := &MigrateConfig{
		Database:    p.database(),
		Environment: p.str("ENVIRONMENT", "development"),
		LogLevel:    p.level("LOG_LEVEL", slog.LevelInfo),
	}
	if err := errors.Join(p.errs...); err != nil {
		return nil, fmt.Errorf("invalid configuration:\n%w", err)
	}
	return c, nil
}

type parser struct {
	getenv func(string) string
	errs   []error
}

func (p *parser) fail(name string, err error) {
	p.errs = append(p.errs, fmt.Errorf("  %s: %w", name, err))
}

func (p *parser) str(name, def string) string {
	if v := strings.TrimSpace(p.getenv(name)); v != "" {
		return v
	}
	return def
}

// boolean parses an optional true/false value (default false).
func (p *parser) boolean(name string) bool {
	v := strings.TrimSpace(p.getenv(name))
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		p.fail(name, fmt.Errorf("must be true or false, got %q", v))
	}
	return b
}

// isLoopback reports whether host is this machine: the names net/smtp
// lets authenticate without TLS.
func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (p *parser) required(name string) string {
	v := strings.TrimSpace(p.getenv(name))
	if v == "" {
		p.fail(name, errors.New("is required"))
	}
	return v
}

func (p *parser) database() Database {
	d := Database{
		URL:  strings.TrimSpace(p.getenv("DATABASE_URL")),
		Host: strings.TrimSpace(p.getenv("DATABASE_HOST")),
		User: strings.TrimSpace(p.getenv("DATABASE_USER")),
		// Not trimmed: a password may legitimately start or end with spaces.
		Password: p.getenv("DATABASE_PASSWORD"),
	}
	switch {
	case d.URL == "" && d.Host == "":
		p.fail("DATABASE_URL", errors.New("is required (or set DATABASE_HOST)"))
	case d.URL != "" && d.Host != "":
		p.fail("DATABASE_HOST", errors.New("cannot be combined with DATABASE_URL; set one of them"))
	case d.Host != "":
		d.Port = 5432
		if v := strings.TrimSpace(p.getenv("DATABASE_PORT")); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 || n > 65535 {
				p.fail("DATABASE_PORT", fmt.Errorf("must be a port number, got %q", v))
			}
			d.Port = n
		}
		d.Name = p.str("DATABASE_NAME", "testflow")
	}
	return d
}

func (p *parser) baseURL(name string) *url.URL {
	v := p.required(name)
	if v == "" {
		return &url.URL{}
	}
	u, err := url.Parse(strings.TrimRight(v, "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		p.fail(name, fmt.Errorf("must be an absolute http(s) URL, got %q", v))
		return &url.URL{}
	}
	return u
}

func (p *parser) level(name string, def slog.Level) slog.Level {
	v := p.getenv(name)
	if strings.TrimSpace(v) == "" {
		return def
	}
	var l slog.Level
	if err := l.UnmarshalText([]byte(strings.TrimSpace(v))); err != nil {
		p.fail(name, fmt.Errorf("must be debug, info, warn or error, got %q", v))
		return def
	}
	return l
}

func (p *parser) duration(name string, def time.Duration) time.Duration {
	v := strings.TrimSpace(p.getenv(name))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		p.fail(name, fmt.Errorf("must be a positive duration such as 30s or 2h, got %q", v))
		return def
	}
	return d
}

func (p *parser) positiveInt(name string) int64 {
	v := p.required(name)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		p.fail(name, fmt.Errorf("must be a positive integer, got %q", v))
		return 0
	}
	return n
}

// pem accepts a PEM block with real newlines or with literal "\n" escapes, as
// is common when a key is passed through a single-line environment variable.
func (p *parser) pem(name string) []byte {
	v := p.required(name)
	if v == "" {
		return nil
	}
	key := []byte(strings.ReplaceAll(v, `\n`, "\n"))
	if block, _ := pem.Decode(key); block == nil {
		p.fail(name, errors.New("is not a PEM encoded key"))
		return nil
	}
	return key
}

func (p *parser) list(name string) []string {
	var out []string
	for item := range strings.SplitSeq(p.getenv(name), ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		p.fail(name, errors.New("is required (comma-separated list)"))
	}
	return out
}
