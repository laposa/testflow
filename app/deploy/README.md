# Deploying TestFlow

How to run the Go TestFlow in production, and how to switch over from the
legacy Laravel app. Configuration is described in full in
[spec.md §5](../../spec.md#5-configuration); this page covers what an operator
needs on top of it.

## What runs

| Component | Notes |
|-----------|-------|
| `laposa/testflow-go` image | One container, one instance (no horizontal scaling). Listens on `:8080` (`ADDR`), runs as a non-root user on `distroless/static`, with no shell. Built from `app-go/` by `.github/workflows/build-image.yaml` on every push to `master`, in the same run that builds the legacy `laposa/testflow` image from `app/`. Tags: `latest` and `master-YYMMDD.<sha>`. Deploy the dated tag so a rollback is a tag change. |
| PostgreSQL | 17 is what CI and the compose stack use. TestFlow needs one database and a user that can create tables (for migrations). |
| SMTP server | For review request emails (`MAIL_MAILER=smtp`). Port 465 (implicit TLS) or a port with STARTTLS. |
| OpenTelemetry Collector | Optional but recommended. Reads the container's JSON log lines and exports them over OTLP (see [Logs](#logs)). |
| TLS-terminating proxy | TestFlow speaks plain HTTP. Put it behind a proxy or load balancer that serves HTTPS on the public `BASE_URL`. |

The image runs `testflow serve` by default. The binary also has
`testflow migrate` and `testflow version`.

## GitHub App

TestFlow signs users in with the GitHub App's OAuth credentials and reads
tests, dispatches workflows and downloads results with the App's installation
token. One App serves one TestFlow instance.

1. In the organisation's settings → **Developer settings → GitHub Apps**, use the
   existing TestFlow App or create one.
2. **Callback URL**: `{BASE_URL}/auth/callback`, e.g.
   `https://testflow.example.com/auth/callback`. Leave "Expire user
   authorization tokens" as it is; TestFlow does not keep user tokens (D14).
3. **Repository permissions**:
   - *Contents*: read (test discovery, manual test cases, raw test files)
   - *Actions*: read and write (list workflows and runs, dispatch, download
     artifacts and logs)
   - *Metadata*: read (mandatory)
4. **Account permissions**: *Email addresses*: read, so reviewers with a private
   email still get review requests.
5. Install the App on the organisation for the repositories whose tests TestFlow
   should list. The number at the end of the installation's settings URL
   (`…/settings/installations/<id>`) is `GITHUB_APP_INSTALLATION_ID`.
6. Note the **Client ID** (`GITHUB_CLIENT_ID`), generate a **client secret**
   (`GITHUB_CLIENT_SECRET`) and a **private key** (`GITHUB_PRIVATE_KEY`, PEM;
   `\n` escapes are accepted when the secret store keeps it on one line).

A tested repository needs `tests/{service}/tests/{suite}/{test}` on `master`
and, for automated tests, an active workflow `run-tests-{service}.yaml` with a
`workflow_dispatch` trigger taking the inputs `environment` and `test_filter`,
which uploads the JUnit report as an artifact named `test-results`
(spec §8.1, §8.4).

## Configuration

All settings are environment variables (spec §5). The binary never reads a
`.env` file or a secret store; inject secrets with the platform (1Password,
Kubernetes secrets, …). A production example:

```sh
ENVIRONMENT=production            # anything but "development": Secure cookies, mailer required
BASE_URL=https://testflow.example.com
LOG_LEVEL=info

DATABASE_URL=postgres://db.example.com:5432/testflow?sslmode=require
DATABASE_USER=testflow
DATABASE_PASSWORD=…               # overrides the URL, no URL-encoding needed

GITHUB_CLIENT_ID=…
GITHUB_CLIENT_SECRET=…
GITHUB_APP_INSTALLATION_ID=…
GITHUB_PRIVATE_KEY=…

TEST_SESSION_ENVIRONMENTS=preprod,beta,prod

MAIL_MAILER=smtp
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=…
SMTP_PASSWORD=…
MAIL_FROM=TestFlow <testflow@example.com>
```

- `ENVIRONMENT` defaults to `development`, which drops the `Secure` flag from the
  session cookie. Always set it in a deployment.
- `BASE_URL` must be the URL users open. It is the OAuth redirect URI, the link
  in review emails and the trusted origin for the CSRF check (§9), so it must
  match the proxy's public scheme and host.
- `STATIC_DIR` and `SMTP_ALLOW_PLAINTEXT` are development-only; startup fails if
  they are set elsewhere.
- Missing or invalid settings stop the process at startup with a message naming
  the variable.

Legacy variables that changed name: `APP_URL` → `BASE_URL`, `APP_ENV` →
`ENVIRONMENT`, `DB_*` → `DATABASE_*`, `GITHUB_APP_INSTALATION_ID` →
`GITHUB_APP_INSTALLATION_ID`, `MAIL_HOST`/`MAIL_PORT`/`MAIL_USERNAME`/`MAIL_PASSWORD` →
`SMTP_*`, `MAIL_FROM_ADDRESS` → `MAIL_FROM`. `SESSION_LIFETIME` takes a Go
duration (`120m`) instead of minutes. `APP_KEY` and the Laravel cache, queue
and session drivers are gone.

## Database migrations

Migrations are embedded in the binary and forward-only (spec §6.11).

- **Recommended:** run `testflow migrate` as a one-off step before rolling out a
  new version (same image, command `migrate`). It needs only the `DATABASE_*`
  variables, not the GitHub or mail secrets.
- **Or** start the server with `testflow serve -migrate`. A Postgres advisory
  lock makes concurrent starts apply each migration once.
- Plain `testflow serve` refuses to start while migrations are pending. After a
  rollback to an older image, it logs a warning about unknown versions and
  starts anyway.

## Health and shutdown

- `GET /healthz` answers `200 {"status":"ok"}`, or `503` when the database does
  not answer within 2 s. It needs no login. The image has no shell or curl, so
  use the platform's HTTP probe (Kubernetes `httpGet`, load balancer health
  check) rather than a Docker `HEALTHCHECK` command.
- On `SIGTERM` the server stops accepting connections and gives requests in
  flight up to 15 s, then exits. The background syncer stops with it.

## Logs

TestFlow writes one JSON object per line to stdout, already in the
OpenTelemetry log data model (spec §11): `timestamp`, `severity_text`,
`severity_number`, `body`, `trace_id`/`span_id`, `resource` (with
`service.version` = the image tag) and `attributes`.

[otel-collector.yaml](otel-collector.yaml) is a Collector config
(`otel/opentelemetry-collector-contrib`) that tails the container runtime's log
files, keeps only TestFlow's lines, maps the fields onto OTel log records and
exports them over OTLP/HTTP:

- `OTLP_ENDPOINT`: the backend's OTLP/HTTP endpoint (default
  `http://localhost:4318`);
- `LOG_FILES`: glob of the container log files (default: Docker's
  `/var/lib/docker/containers/*/*-json.log`; on Kubernetes e.g.
  `/var/log/pods/*testflow*/*/*.log`). The Collector needs read access to them.

Without a Collector the logs are still readable with `docker logs` or
`kubectl logs`.

## Cut-over from the legacy app

The Go app starts with an empty database; legacy sessions are not migrated
(Q7). Sessions in the legacy app stay readable only as long as it runs.

1. Provision Postgres and the secrets above. Reuse the GitHub App, Client ID,
   client secret, private key and installation id of the legacy deployment, or
   create a new App.
2. Deploy `laposa/testflow-go:master-YYMMDD.<sha>` (built by
   `build-image.yaml`) with `ENVIRONMENT`, `BASE_URL` and the rest set, and run
   `testflow migrate`.
3. If `BASE_URL` differs from the legacy `APP_URL`, add
   `{BASE_URL}/auth/callback` to the App's callback URLs.
4. Smoke test: sign in, create a session with an automated and a manual test,
   start both, wait for the automated run to report back, finish the manual run,
   comment, request a review (check the email arrives), approve it, and open both
   reports.
5. Point users (DNS or the proxy) at the new instance and switch the legacy
   deployment off. `build-image.yaml` keeps publishing the legacy image
   (`laposa/testflow`, from `app/`) from the same commits, so it can come back
   if needed; it uses its own database, untouched by the Go app. Dropping the
   legacy build is a separate change.

Old bookmarks of `/sessions/create` redirect to `/sessions/new`. Links to legacy
session ids do not carry over, since the new database numbers sessions afresh.
Removing `app/` from the repository is a separate decision.
