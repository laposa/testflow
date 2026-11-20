// Command testflow is the TestFlow server and its maintenance commands
// (spec D10): `testflow serve` and `testflow migrate`.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	netmail "net/mail"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/laposa/testflow/app-go/internal/auth"
	"github.com/laposa/testflow/app-go/internal/config"
	"github.com/laposa/testflow/app-go/internal/db"
	"github.com/laposa/testflow/app-go/internal/discovery"
	"github.com/laposa/testflow/app-go/internal/github"
	"github.com/laposa/testflow/app-go/internal/mail"
	"github.com/laposa/testflow/app-go/internal/runs"
	"github.com/laposa/testflow/app-go/internal/store"
	"github.com/laposa/testflow/app-go/internal/telemetry"
	"github.com/laposa/testflow/app-go/internal/web"
)

// version is set at build time: -ldflags "-X main.version=…".
var version = "dev"

const usage = `Usage: testflow <command> [flags]

Commands:
  serve [-migrate]  run the web server; -migrate applies pending migrations first
  migrate           apply pending database migrations and exit
  version           print the build version

Configuration is read from environment variables (see spec.md §5).
`

// shutdownTimeout bounds how long in-flight requests may take after
// SIGINT/SIGTERM.
const shutdownTimeout = 15 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "serve":
		fs := flag.NewFlagSet("serve", flag.ContinueOnError)
		fs.SetOutput(stderr)
		migrate := fs.Bool("migrate", false, "apply pending migrations before starting")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		cfg, err := config.Load(getenv)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		logger := newLogger(stdout, cfg.LogLevel, cfg.Environment)
		if err := serve(ctx, cfg, logger, *migrate, nil); err != nil {
			logger.Error("server stopped with error", "err", err)
			return 1
		}
		return 0
	case "migrate":
		cfg, err := config.LoadMigrate(getenv)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		logger := newLogger(stdout, cfg.LogLevel, cfg.Environment)
		if err := migrateOnly(ctx, cfg.Database, logger); err != nil {
			logger.Error("migration failed", "err", err)
			return 1
		}
		return 0
	case "version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "-h", "-help", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

func newLogger(w io.Writer, level slog.Level, environment string) *slog.Logger {
	logger := slog.New(telemetry.NewHandler(w, level, telemetry.Resource{
		ServiceName:    "testflow",
		ServiceVersion: version,
		Environment:    environment,
	}))
	slog.SetDefault(logger)
	return logger
}

func migrateOnly(ctx context.Context, c config.Database, logger *slog.Logger) error {
	pool, err := db.Open(ctx, dbConfig(c))
	if err != nil {
		return err
	}
	defer pool.Close()
	return applyMigrations(ctx, pool, logger)
}

func dbConfig(c config.Database) db.Config {
	return db.Config{URL: c.URL, Host: c.Host, Port: c.Port, Name: c.Name, User: c.User, Password: c.Password}
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	migrations, err := db.Migrations()
	if err != nil {
		return err
	}
	applied, err := db.Migrate(ctx, pool, migrations)
	for _, m := range applied {
		logger.Info("migration applied", "migration", m.Name)
	}
	if err != nil {
		return err
	}
	logger.Info("database schema up to date", "applied", len(applied))
	return nil
}

// checkSchema refuses to run against a database with pending migrations.
// Versions this binary doesn't know (the database was migrated by a newer
// release, e.g. during a rollback) are only a warning, since forward-only
// migrations are meant to stay compatible.
func checkSchema(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	migrations, err := db.Migrations()
	if err != nil {
		return err
	}
	st, err := db.CheckStatus(ctx, pool, migrations)
	if err != nil {
		return err
	}
	if len(st.Unknown) > 0 {
		logger.Warn("database has migrations this binary does not know", "versions", st.Unknown)
	}
	if n := len(st.Pending); n > 0 {
		return fmt.Errorf("%d pending migration(s), first %s: run `testflow migrate` or start with `serve -migrate`", n, st.Pending[0].Name)
	}
	return nil
}

// serve runs the HTTP server until ctx is cancelled, then shuts down
// gracefully. If ready is non-nil it receives the listening address.
func serve(ctx context.Context, cfg *config.Config, logger *slog.Logger, migrate bool, ready chan<- string) error {
	// The GitHub App client parses the private key, so a bad key stops the
	// server here rather than on the first API call.
	gh, err := newGitHubClient(cfg, logger)
	if err != nil {
		return err
	}

	pool, err := db.Open(ctx, dbConfig(cfg.Database))
	if err != nil {
		return err
	}
	defer pool.Close()

	if migrate {
		err = applyMigrations(ctx, pool, logger)
	} else {
		err = checkSchema(ctx, pool, logger)
	}
	if err != nil {
		return err
	}

	st := store.New(pool)
	sessions := &auth.Sessions{
		Store:    st,
		Lifetime: cfg.SessionLifetime,
		Secure:   !cfg.IsDevelopment(),
		Logger:   logger,
	}
	mailer, err := newMailer(cfg.Mail, logger)
	if err != nil {
		return err
	}
	// The syncer also deletes expired web sessions (spec §8.5).
	syncer := runs.NewSyncer(runs.SyncerConfig{GitHub: gh, Store: st, Sessions: sessions, Logger: logger})
	handler, err := web.NewHandler(web.Options{
		Logger:      logger,
		Version:     version,
		Development: cfg.IsDevelopment(),
		StaticDir:   cfg.StaticDir,
		DB:          pool,
		BaseURL:     cfg.BaseURL,
		Sessions:    sessions,
		OAuth: &auth.GitHubOAuth{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.BaseURL.JoinPath("auth", "callback").String(),
		},
		Users:        st,
		Discovery:    discovery.New(discovery.Config{GitHub: gh, Logger: logger}),
		TestSessions: st,
		Activity:     st,
		Environments: cfg.TestSessionEnvironments,
		Runs:         st,
		Dispatcher:   &runs.Dispatcher{GitHub: gh, Store: st, Logger: logger},
		Syncer:       syncer,
		TestFiles:    gh,
		Manual:       st,
		Reviews:      st,
		Mailer:       mailer,
	})
	if err != nil {
		return err
	}
	srv := web.NewServer(cfg.Addr, handler, logger)

	// Background work stops with ctx and finishes before the pool closes.
	var bg sync.WaitGroup
	defer bg.Wait()
	bgCtx, stopBg := context.WithCancel(ctx)
	defer stopBg()
	bg.Go(func() { syncer.Run(bgCtx, cfg.SyncInterval) })

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return err
	}
	logger.Info("server starting", "addr", ln.Addr().String(), "config", cfg)
	if ready != nil {
		ready <- ln.Addr().String()
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	logger.Info("server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errc; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("server stopped")
	return nil
}

func newGitHubClient(cfg *config.Config, logger *slog.Logger) (*github.Client, error) {
	return github.New(github.Config{
		ClientID:       cfg.GitHubClientID,
		InstallationID: cfg.GitHubInstallationID,
		PrivateKey:     cfg.GitHubPrivateKey,
		Logger:         logger,
	})
}

// newMailer returns the mailer MAIL_MAILER selects (spec §8.10).
func newMailer(c config.MailConfig, logger *slog.Logger) (mail.Mailer, error) {
	if c.Mailer != "smtp" {
		return mail.Log{Logger: logger}, nil
	}
	from, err := netmail.ParseAddress(c.From)
	if err != nil {
		return nil, fmt.Errorf("MAIL_FROM: %w", err)
	}
	return &mail.SMTP{Host: c.Host, Port: c.Port, Username: c.Username, Password: c.Password, From: *from,
		AllowPlaintext: c.AllowPlaintext}, nil
}
