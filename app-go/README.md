# TestFlow (Go)

The Go rewrite of TestFlow. Read [../intent.md](../intent.md) for why and
[../spec.md](../spec.md) for how. The legacy Laravel app in [../app](../app/)
stays the deployed version until cut-over (spec §14, iteration 13).

## Requirements

- Go 1.27+
- Docker with Compose (for Postgres in `make test-integration` / `make check`,
  and for `make docker`)

### Installing Go

Install the latest Go 1.27.x (check the current patch release on
[go.dev/dl](https://go.dev/dl/)).

**Linux / WSL** (official tarball):

```sh
GO_VERSION=1.27.4   # latest 1.27.x from go.dev/dl
curl -fsSLO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
rm "go${GO_VERSION}.linux-amd64.tar.gz"

# add Go to your PATH (once), then open a new shell
echo 'export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"' >> ~/.bashrc
```

Use `linux-arm64` instead of `linux-amd64` on ARM machines.

**macOS:** `brew install go`, or the `.pkg` installer from go.dev/dl.

**Windows:** `winget install GoLang.Go`, or the `.msi` installer from go.dev/dl.
Inside WSL, use the Linux steps above.

Check the installation:

```sh
go version   # go version go1.27.x ...
```

The editor's Go extension (VS Code: `golang.go`) also needs `gopls`; the
extension offers to install it on first use.

### Without a local Go

If `go` is not on the PATH, the Makefile runs the Go commands in the
`golang:1.27` Docker image instead, as your user, with caches in
`~/.cache/testflow-go`. This is slower and doesn't help the editor, so
installing Go is recommended. `GO` and `GOFMT` can also be overridden to point
at any other toolchain.

## Common tasks

All commands run from `app-go/`.

| Command | What it does |
|---------|--------------|
| `cp .env.example .env` | local settings (git-ignored; see spec §5) |
| `make run` | run the server on `:8080` with `.env` |
| `make test` | unit and handler tests |
| `make check` | `gofmt`, `go vet`, tests, integration tests: run before every PR (D16) |
| `make build` | static binary in `bin/testflow` |
| `make docker` | image `laposa/testflow-go:<version>` (pushing is manual) |
| `make docker-up` | Postgres + app + OTel Collector (debug exporter) |
| `make update-htmx HTMX_VERSION=x.y.z` | vendor a new htmx (spec §12) |

The binary has subcommands: `testflow serve`, `testflow migrate` (iteration 2)
and `testflow version`.

## Layout

See spec §4. In short: `cmd/testflow` is the entry point, `internal/config`
parses the environment, `internal/telemetry` writes OTel-shaped JSON logs,
`internal/web` holds routes and handlers, `internal/web/views` the Gomponents
components, and `internal/web/static` the embedded CSS, fonts and htmx.
