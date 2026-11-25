# TestFlow (Go)

The TestFlow web app. Read [../intent.md](../intent.md) for why and
[../spec.md](../spec.md) for how. It replaced the legacy Laravel app in
[../app](../app/), which is still built as a separate image (`laposa/testflow`)
until it is removed. Deploying is described
in [deploy/README.md](deploy/README.md).

## Requirements

- Go 1.27+
- Docker with Compose (for Postgres in `make test-integration` / `make check`,
  and for `make docker`)
- `docker login dhi.io` with a Docker Hub account: the Go build image is a
  [Docker Hardened Image](https://docs.docker.com/dhi/) (free). Only the Go
  image comes from `dhi.io`; Postgres, the Collector and the runtime base
  image don't

## Create .env using 1Password cli

```bash
op inject -i .env.local.tpl -o .env
```

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

If `go` is not on the PATH, the Makefile runs the Go commands in the Docker
Hardened Image `dhi.io/golang:1.27.4-debian13-dev` instead, as your user, with caches in
`~/.cache/testflow-go`. This is slower and doesn't help the editor, so
installing Go is recommended. `GO` and `GOFMT` can also be overridden to point
at any other toolchain.

## Common tasks

All commands run from `app-go/`.

| Command                               | What it does                                                                        |
| ------------------------------------- | ----------------------------------------------------------------------------------- |
| `cp .env.example .env`                | local settings (git-ignored; see spec §5)                                           |
| `make run`                            | run the server on `:8080` with `.env`, applying migrations first                    |
| `make test`                           | unit and handler tests                                                              |
| `make check`                          | `gofmt`, `go vet`, tests, integration tests: what CI runs, also run before every PR |
| `make build`                          | static binary in `bin/testflow`                                                     |
| `make docker`                         | image `laposa/testflow-go:<version>`, local only (CI publishes from `master`)       |
| `make docker-up`                      | Postgres + app + OTel Collector (debug exporter)                                    |
| `make update-htmx HTMX_VERSION=x.y.z` | vendor a new htmx (spec §12)                                                        |

The binary has subcommands:

- `testflow serve [-migrate]`: run the server. It refuses to start while migrations
  are pending, unless `-migrate` applies them first.
- `testflow migrate`: apply pending migrations and exit. Needs only `DATABASE_URL`
  (and `DATABASE_USER`/`DATABASE_PASSWORD` if the credentials aren't in the URL),
  or instead `DATABASE_HOST` with `DATABASE_USER`/`DATABASE_PASSWORD` (optional
  `DATABASE_PORT`, default 5432, and `DATABASE_NAME`, default `testflow`).
- `testflow version`.

With `STATIC_DIR=internal/web/static` (set in `.env.example`), `make run`
serves CSS, JS and icons from the source tree, so edits show on a browser
reload. Go code and views still need a restart.

`make run` needs a Postgres: `docker compose -f deploy/docker-compose.yaml up -d
postgres` starts the one `.env.example` points to.

### Testing email locally with Mailpit

By default (`MAIL_MAILER=log`) emails such as review requests are only written to
the log. To see them as real messages, run [Mailpit](https://mailpit.axllent.org/),
a local SMTP server with a web inbox:

```sh
docker run -d --name mailpit --restart unless-stopped \
  -p 127.0.0.1:1025:1025 -p 127.0.0.1:8025:8025 \
  -e MP_SMTP_AUTH_ACCEPT_ANY=1 -e MP_SMTP_AUTH_ALLOW_INSECURE=1 \
  axllent/mailpit
```

Then point the app at it in `.env` and restart `make run`:

```sh
MAIL_MAILER=smtp
SMTP_HOST=localhost
SMTP_PORT=1025
SMTP_USERNAME=dev
SMTP_PASSWORD=dev
MAIL_FROM=testflow@example.com
SMTP_ALLOW_PLAINTEXT=true
```

Request a review on a session and open <http://localhost:8025> to read the email.

- Mailpit accepts any username and password; the app still needs both set.
- `SMTP_ALLOW_PLAINTEXT=true` lets the app send without STARTTLS, which Mailpit does
  not offer by default. The app refuses to start with it unless
  `ENVIRONMENT=development` and `SMTP_HOST` is `localhost`, `127.0.0.1` or `::1`;
  everywhere else STARTTLS is required.
- The ports are bound to `127.0.0.1` only, so the inbox is not reachable from the
  network. Stop it with `docker rm -f mailpit`.
- Inside the compose app container (`make docker-up`), `localhost` is the container
  itself; use `make run` for this setup.

## Layout

See spec §4. In short: `cmd/testflow` is the entry point, `internal/config`
parses the environment, `internal/telemetry` writes OTel-shaped JSON logs, `internal/db` opens the
Postgres pool and runs the migrations in `internal/db/migrations`,
`internal/store` holds the SQL queries,
`internal/auth` the GitHub OAuth login, web sessions and auth middleware,
`internal/github` the GitHub App REST client, `internal/discovery` the test
discovery behind the test picker, `internal/junit` the JUnit report parser,
`internal/testcase` the `*.manual.yaml` test cases shown in manual runs,
`internal/mail` the SMTP and log mailers,
`internal/runs` the dispatch of automated runs, the start of manual runs and
the background syncer that follows automated runs on GitHub,
`internal/web` holds routes and handlers, `internal/web/views` the Gomponents
components, and `internal/web/static` the embedded CSS, fonts and htmx.
