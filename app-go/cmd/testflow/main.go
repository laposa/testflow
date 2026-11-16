// Command testflow is the TestFlow server and its maintenance commands
// (spec D10): `testflow serve` and `testflow migrate`.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/laposa/testflow/app-go/internal/config"
	"github.com/laposa/testflow/app-go/internal/telemetry"
	"github.com/laposa/testflow/app-go/internal/web"
)

// version is set at build time: -ldflags "-X main.version=…".
var version = "dev"

const usage = `Usage: testflow <command>

Commands:
  serve     run the web server and background syncer
  migrate   apply pending database migrations
  version   print the build version

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
		cfg, err := config.Load(getenv)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		logger := newLogger(stdout, cfg)
		if err := serve(ctx, cfg, logger, nil); err != nil {
			logger.Error("server stopped with error", "err", err)
			return 1
		}
		return 0
	case "migrate":
		fmt.Fprintln(stderr, "testflow migrate: not available yet (spec §14, iteration 2)")
		return 1
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

func newLogger(w io.Writer, cfg *config.Config) *slog.Logger {
	logger := slog.New(telemetry.NewHandler(w, cfg.LogLevel, telemetry.Resource{
		ServiceName:    "testflow",
		ServiceVersion: version,
		Environment:    cfg.Environment,
	}))
	slog.SetDefault(logger)
	return logger
}

// serve runs the HTTP server until ctx is cancelled, then shuts down
// gracefully. If ready is non-nil it receives the listening address.
func serve(ctx context.Context, cfg *config.Config, logger *slog.Logger, ready chan<- string) error {
	handler := web.NewHandler(web.Options{
		Logger:      logger,
		Version:     version,
		Development: cfg.IsDevelopment(),
	})
	srv := web.NewServer(cfg.Addr, handler, logger)

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
