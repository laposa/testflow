# TestFlow: Specification

> **How** TestFlow is built. Read [intent.md](intent.md) first for the *why*.
>
> This is a living document. Each iteration may refine it, and a PR that changes
> behaviour described here updates this file in the same PR. GitHub issues refer to
> sections by their anchor (e.g. "implements §6.3").
>
> **Legacy reference:** `~/LaposaDev/testflow_legacy` at commit `b33dbec` (read-only).
> Paths below starting with `legacy:` are relative to that folder.

---

## 1. Decisions

| ID | Decision | Status |
|----|----------|--------|
| D1 | Go, latest stable (1.27.x at time of writing). `go.mod` sets the minimum. | Accepted |
| D2 | HTTP via `net/http` with the `ServeMux` method and wildcard patterns (`GET /sessions/{id}`). No router library. | Accepted |
| D3 | HTML via `maragu.dev/gomponents`. htmx attributes are written with a small local helper (`internal/web/views/hx.go`: one generic `hx.Attr(name, value)` plus named helpers for the attributes we use), not `gomponents-htmx`. Reason: `gomponents-htmx` (v0.6.1, last released Oct 2024) was built for htmx 2, so its helpers would trail htmx 4 (D4). A local helper of about 30 lines works with any htmx version and adds no module. As of Nov 2026 the repo has had no commits since Oct 2024; htmx 4 support exists only as open issues. **Revisit** once `gomponents-htmx` releases htmx 4 support. Only the helper file would change. | Accepted |
| D4 | htmx **4.0.1**, pinned exactly, vendored as a static file and embedded with `embed`. No CDN, no npm. Upgrades are manual (§12). | Accepted |
| D5 | PostgreSQL via `github.com/jackc/pgx/v5` (`pgxpool`), using pgx directly, not `database/sql`. | Accepted |
| D6 | Migrations: plain `.sql` files, embedded, applied by a small runner of our own. No migration library. | Accepted |
| D7 | Logging: `log/slog` JSON to stdout, using OpenTelemetry log data model field names. An OTel Collector reads stdout and exports over OTLP. No OTel SDK in the binary. | Accepted |
| D8 | YAML via `go.yaml.in/yaml/v3`. | Accepted |
| D9 | No Markdown rendering. | Accepted |
| D10 | One binary with subcommands (`serve`, `migrate`, `validate`). The legacy Node tools become part of it. | Accepted |
| D11 | Tests use the standard `testing` package only (no testify). DB tests run against a real Postgres. | Accepted |
| D12 | Manual and automated per-test results are stored in one table (`run_test_results`) instead of keeping manual results as JUnit XML (the legacy approach). | **Proposed** |
| D13 | Activity entries store plain text and structured `metadata`. HTML is produced only by Gomponents views. (Legacy stored HTML in `body`, an XSS risk.) | **Proposed** |
| D14 | GitHub user OAuth tokens are **not** stored. Legacy saved them but never used them. All GitHub API access uses the GitHub App installation token. | **Proposed** |

## 2. Dependency policy

A third-party module is allowed only if the standard library has no reasonable
equivalent **and** it is listed here with a reason. Adding one means updating this
table in the same PR.

| Module | Why | Transitive deps |
|--------|-----|-----------------|
| `maragu.dev/gomponents` | Type-safe HTML components (D3) | none |
| `github.com/jackc/pgx/v5` | Postgres driver and pool (D5) | `jackc/pgpassfile`, `jackc/pgservicefile`, `jackc/puddle/v2`, `golang.org/x/sync`, `golang.org/x/text` |
| `go.yaml.in/yaml/v3` | Parse and emit `*.manual.yaml` (D8) | none |

Static assets (not Go modules), pinned and upgraded by hand:

| Asset | Version | Source |
|-------|---------|--------|
| `internal/web/static/htmx-4.0.1.min.js` | 4.0.1 | npm package `htmx.org@4.0.1`, `dist/htmx.min.js` |
| DM Sans font files | as in legacy | `legacy:app/public/fonts/` |

Everything else comes from the standard library, including: OAuth2 (`net/http`),
the RS256 JWT for the GitHub App (`crypto/rsa`, `crypto/x509`, `encoding/pem`), CSRF
(`http.CrossOriginProtection`), sessions (`crypto/rand` and cookies), zip
(`archive/zip`), JUnit (`encoding/xml`), mail (`net/smtp`), scheduling (`time.Ticker`),
and embedding (`embed`).

## 3. Architecture

```
            browser (HTML + htmx)
                    │
          ┌─────────▼──────────┐        ┌───────────────┐
          │  testflow serve    │◄──────►│  PostgreSQL   │
          │  net/http + views  │        └───────────────┘
          │  background syncer │        ┌───────────────┐
          └───┬───────────┬────┘───────►│  SMTP server  │
              │           │             └───────────────┘
   GitHub OAuth│           │GitHub App (REST API v3)
              ▼           ▼
           github.com / api.github.com
              
  stdout (slog JSON) ──► OTel Collector ──► OTLP backend
```

- **Single process.** The HTTP server and a background syncer goroutine (§8.5) run in
  the same process and shut down gracefully on SIGINT/SIGTERM.
- **Server-rendered.** Every page is a full HTML document. htmx requests receive HTML
  fragments, produced by the same Gomponents components.
- **State** lives only in Postgres, with one exception: the GitHub installation token
  is cached in memory.

## 4. Repository layout

```
cmd/testflow/            main.go: subcommand dispatch (serve | migrate | validate)
internal/config/         env parsing and validation
internal/telemetry/      slog handler (OTel field names), trace context, HTTP middleware
internal/db/             pgxpool setup, migration runner
internal/db/migrations/  0001_init.sql, ... (embedded)
internal/store/          SQL queries, one file per aggregate (users.go, sessions.go, ...)
internal/github/         App JWT, installation token cache, REST client
internal/auth/           OAuth login flow, web sessions, middleware
internal/testcase/       manual test-case model, YAML parse/emit, validation
internal/junit/          JUnit XML parsing and test-name matching
internal/discovery/      repository → service/suite/test tree
internal/runs/           dispatch automated runs, manual run lifecycle, syncer
internal/activity/       activity log writes and types
internal/mail/           SMTP and log mailers
internal/web/            handlers, routes, htmx helpers
internal/web/views/      Gomponents layouts, components, pages
internal/web/static/     css, fonts, htmx.min.js (embedded)
deploy/                  Dockerfile, docker-compose.yaml, otel-collector.yaml
.github/workflows/       ci.yaml, build-image.yaml
```

Rules:
- `internal/web` depends on domain packages. Domain packages never import `web`.
- Domain packages take interfaces for GitHub and mail so tests can use fakes.
- No globals except the `slog` default logger.

## 5. Configuration

Everything comes from environment variables, parsed once at startup by
`internal/config`. Missing required values stop the process with a clear error.
For local development a `.env` file is read only by `make`/compose, not by the binary.

| Variable | Required | Default | Legacy equivalent |
|----------|----------|---------|-------------------|
| `ADDR` | no | `:8080` | |
| `BASE_URL` | yes | | `APP_URL` |
| `ENVIRONMENT` | no | `development` | `APP_ENV` |
| `LOG_LEVEL` | no | `info` | `LOG_LEVEL` |
| `DATABASE_URL` | yes | | `DB_*` |
| `SESSION_LIFETIME` | no | `120m` | `SESSION_LIFETIME` (minutes) |
| `GITHUB_CLIENT_ID` | yes | | same. Also used as JWT `iss` |
| `GITHUB_CLIENT_SECRET` | yes | | same |
| `GITHUB_APP_INSTALLATION_ID` | yes | | `GITHUB_APP_INSTALATION_ID` (typo fixed) |
| `GITHUB_PRIVATE_KEY` | yes | | same (PEM, `\n` escapes allowed) |
| `TEST_SESSION_ENVIRONMENTS` | yes | | same (comma-separated, e.g. `preprod,beta,prod`) |
| `SYNC_INTERVAL` | no | `30s` | (Livewire `wire:poll.30s`) |
| `MAIL_MAILER` | no | `log` | same (`log` or `smtp`) |
| `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD` | if `smtp` | | `MAIL_*` |
| `MAIL_FROM` | if `smtp` | | `MAIL_FROM_ADDRESS` |

Secrets are injected by the environment (1Password `op://` references in the
legacy template). The binary never reads secret stores directly.

## 6. Data model (PostgreSQL)

Laravel housekeeping tables (`cache*`, `jobs`, `job_batches`, `failed_jobs`,
`password_reset_tokens`) are not carried over. All tables have `created_at` and
`updated_at timestamptz NOT NULL DEFAULT now()` unless noted.

### 6.1 `users`
| column | type | notes |
|---|---|---|
| id | bigserial PK | |
| github_id | bigint UNIQUE NOT NULL | |
| name | text NOT NULL | falls back to email, then login (legacy: `AuthController::callback`) |
| email | text | may be null if private on GitHub |
| login | text NOT NULL | GitHub username (new) |

### 6.2 `web_sessions`
`id text PK` (32 random bytes, base64url), `user_id → users ON DELETE CASCADE`,
`expires_at timestamptz`, `created_at`. Expired rows are deleted by the syncer loop.

### 6.3 `test_sessions`
`id`, `issuer_id → users`, `name text NOT NULL`, `environment text NOT NULL`.

### 6.4 `session_services`
`id`, `session_id → test_sessions ON DELETE CASCADE`,
`type text CHECK (type IN ('automated','manual'))`, `name` (e.g. `web`, `api`,
`mobile`), `path`, `repository_id bigint`, `repository_name` (`owner/repo`),
`workflow_id bigint NULL`, `commit_sha`, `branch text DEFAULT 'master'`.

Display name: `<repo without owner>/<name> (<type>)`
(legacy: `SessionService::displayName`).

### 6.5 `service_suites`
`id`, `service_id → session_services ON DELETE CASCADE`, `name`, `path`.

### 6.6 `suite_tests`
`id`, `suite_id → service_suites ON DELETE CASCADE`, `name` (file name), `path`.

### 6.7 `service_runs`
| column | type | notes |
|---|---|---|
| id | bigserial PK | |
| service_id | → session_services CASCADE | |
| type | `automated` \| `manual` | |
| status | text NOT NULL | see §8.4 |
| passed, failed, skipped | int NULL | |
| duration | double precision NULL | seconds, from JUnit |
| result_log | text NULL | raw JUnit XML (automated only) |
| run_log | text NULL | largest file from the GitHub run log zip |
| github_run_id | bigint NULL | |
| commit_sha | text NULL | |
| started_at, finished_at | timestamptz NULL | |

### 6.8 `run_test_results` (D12)
`run_id → service_runs CASCADE`, `test_id → suite_tests CASCADE`,
`status text CHECK (status IN ('pass','fail','skipped','error','notRun'))`,
`comment text`, `failure_message text`, `duration double precision`,
`PRIMARY KEY (run_id, test_id)`.

### 6.9 `activity_log`
`id`, `session_id → test_sessions CASCADE`, `user_id → users NULL` (null = system,
e.g. the syncer), `type text`, `body text` (plain text), `metadata jsonb`, `created_at`.

Types (legacy: `app/Enums/SessionActivityType.php`): `session_created`,
`run_dispatched`, `run_status_changed`, `manual_run_started`, `manual_run_finished`,
`review_requested`, `review_approved`, `review_rejected`, `comment`.

### 6.10 `review_requests`
`id`, `session_id → test_sessions CASCADE`, `requester_id → users`,
`reviewer_id → users`, `status text CHECK (status IN ('pending','approved','rejected')) DEFAULT 'pending'`,
`completed_at timestamptz NULL`.

### 6.11 Migrations
- Files: `internal/db/migrations/NNNN_description.sql`, applied in order, each in a
  transaction. Applied versions are recorded in `schema_migrations(version, applied_at)`.
- Forward-only. Fix mistakes with a new migration.
- `testflow migrate` applies pending migrations. `testflow serve` refuses to start if
  migrations are pending (unless `-migrate` is passed).

## 7. Routes

All routes except `/login`, `/auth/*`, `/healthz` and `/static/*` require
authentication. Unauthenticated requests redirect to `/login`, or get HTTP 401 with
the `HX-Redirect` header for htmx requests.

| Method & path | Purpose | Legacy |
|---|---|---|
| `GET /healthz` | liveness and DB ping | new |
| `GET /static/{path...}` | embedded assets, long cache | `public/` |
| `GET /login` | login page with "Sign in with GitHub" | `AuthController::login` |
| `POST /auth/redirect` | start OAuth (sets `state` cookie) | `AuthController::redirect` |
| `GET /auth/callback` | finish OAuth, upsert user, create session | `AuthController::callback` |
| `POST /auth/logout` | destroy session (legacy used GET) | `AuthController::logout` |
| `GET /` and `GET /sessions` | session list | `SessionController::index` |
| `GET /sessions/new` | test picker (discovery §8.1) | `SessionController::create` |
| `POST /sessions` | create session | `SessionController::store` |
| `GET /sessions/{id}` | session detail: services, runs, activity, reviews | `SessionController::show` |
| `GET /sessions/{id}/runs` | htmx fragment: run list (polled) | `Livewire/Runs/ListRuns` |
| `POST /sessions/{id}/runs` | start runs for selected services | `ListRuns::createRun` |
| `GET /sessions/{id}/runs/{run}` | run detail | `SessionRunController::show` |
| `GET /sessions/{id}/runs/{run}/manual` | htmx fragment: manual run wizard, current test | `Livewire/Runs/ManualTestRun` |
| `POST /sessions/{id}/runs/{run}/manual/{test}` | save result for one test, return next | `ManualTestRun::updateResults` |
| `POST /sessions/{id}/runs/{run}/finish` | finish manual run (unanswered → skipped) | `ManualTestRun::markRunAsFinished` |
| `GET /sessions/{id}/runs/{run}/tests/{test}` | raw test file at the run's commit, `text/plain` | `SessionRunController::showTestFile` |
| `POST /sessions/{id}/comments` | add comment | `Livewire/Comments/CreateComment` |
| `POST /sessions/{id}/reviews` | request review | `Livewire/Reviews/RequestReview` |
| `POST /sessions/{id}/reviews/{review}/approve` | approve | `ReviewForm::approve` |
| `POST /sessions/{id}/reviews/{review}/reject` | reject | `ReviewForm::reject` |
| `POST /sessions/{id}/reviews/{review}/revoke` | delete pending request | `RevokeReviewForm::delete` |
| `GET /sessions/{id}/report/executive` | summary report | `SessionReportController::executive` |
| `GET /sessions/{id}/report/full` | full report | `SessionReportController::full` |
| `GET /tools/test-case` | manual test-case generator form | `legacy:generate-yaml` |
| `POST /tools/test-case` | returns generated YAML as a download | `legacy:generate-yaml` |

Dropped: `GET /github/callback` (points to a controller that does not exist in legacy).

## 8. Behaviour

### 8.1 Test discovery
Legacy: `app/Actions/Github/FetchTests.php`.

1. List repositories of the installation (`GET /installation/repositories`).
2. For each repository, get the latest commit of `master`
   (`GET /repos/{repo}/commits/master`). Its SHA pins everything that follows.
3. Walk the contents API: `tests/{service}/tests/{suite}/{test}`, at most 3 levels.
   Skip entries whose name starts with `_`. Repositories without `tests/` are skipped.
4. Match each service to the workflow whose path ends with `run-tests-{service}.yaml`
   (`GET /repos/{repo}/actions/workflows`).
5. A test is **manual** if its file name contains `manual` (in practice
   `*.manual.yaml`). Otherwise it is **automated**.
6. Result: tree grouped by repository → service → suite → test, each test carrying
   repository id/name, commit SHA, service/suite/test name and path, workflow id.

Discovery calls GitHub many times. It runs concurrently per repository (bounded),
and the result is cached in memory for a short TTL (default 5 min, with a "refresh"
button). Legacy fetched on every page load.

### 8.2 Creating a session
Legacy: `app/Actions/Session/CreateSession.php`.

- Input: `name` (required), `environment` (one of `TEST_SESSION_ENVIRONMENTS`), at
  least one selected test ("Please select at least one test suite.").
- Selected tests are grouped by type (manual/automated), then by
  repository+service. Each group becomes a `session_services` row (with
  `branch='master'` and the discovery commit SHA), its suites, and their tests.
- Logs activity `session_created`: "{user} created the session."
- Redirect to `/sessions/{id}`.

### 8.3 Session list
Legacy: `SessionController::index`, `Session::lastRuns`.

Ordered by id, newest first. For each session, only the **latest run per service**
is aggregated. If a run has `passed>0 || failed>0 || status=='success'`, its counts
are added to passed/failed/skipped. Otherwise the session shows as *running*.

### 8.4 Automated runs
Legacy: `app/Actions/Github/DispatchSessionRun.php`, `FetchSessionWorkflowRuns.php`.

**Dispatch** (`POST /repos/{repo}/actions/workflows/{workflow_id}/dispatches`, ref
`master`) with inputs `environment` = session environment and `test_filter`:

| service name | `test_filter` |
|---|---|
| `mobile` | unique suite names, joined with `,` |
| `web` | `tests/{suite}/{test}` for each test, joined with `,` |
| `api` | test name up to its first `.`, joined with `\|` |
| other | empty (open question Q3) |

Then insert a `service_runs` row with `status='unknown'` and log `run_dispatched`:
"{user} has executed: {service} with run ID {id}, …".

**Matching a GitHub run to our row:** the dispatch API returns no run id. Legacy
matches the workflow run whose `created_at` is within **±5 s** of our row's
`created_at`. Keep this for parity (see Q1).

**Status** = GitHub `conclusion` if set, otherwise `status`. Values:
`unknown`, `queued`, `in_progress`, `success`, `failure`, `neutral`, `cancelled`,
`skipped`, `timed_out`, `action_required`, `stale`.

**On each sync of a matched run:**
- set `github_run_id`, `commit_sha` (= `head_sha`), and `started_at` = min job `started_at`;
- once GitHub `status == completed`:
  - `result_log`: download the artifact named `test-results` and take the first
    `*.xml` in the zip; parse it (§8.6) to fill passed/failed/skipped/duration and
    `run_test_results`;
  - `run_log`: download the run logs zip and keep the **largest** file;
  - `finished_at` = max job `completed_at`;
- if status changed, log `run_status_changed`: "Run {id} for {service} has changed
  status from {old} to {new}", adding "with {p} passed and {f} failed tests" when
  results exist. For `mobile`, if `run_log` contains a URL matching
  `https://console.mobile.dev/uploads\S*`, store it in `metadata.maestro_url`. The
  view renders it as a "Maestro Cloud Console" link.

### 8.5 Background syncer
Replaces Livewire polling (`wire:poll.30s`) and the `app:sync-github-runs` command.

- A goroutine runs every `SYNC_INTERVAL`. It picks automated runs where
  `run_log IS NULL` and the run is newer than 24 h (open question Q2), groups them by
  (repository, workflow), fetches workflow runs once per group, and applies §8.4.
- The session page polls the run-list fragment (`hx-trigger="every 30s"`) **only
  while** the session has unfinished runs. The fragment reads the DB and never calls
  GitHub.
- A "refresh now" action triggers an immediate sync of that session.
- The same loop deletes expired `web_sessions`.

### 8.6 JUnit parsing
Legacy: `app/Services/JUnitLogParser.php`.

- Root `<testsuites>` with `<testsuite name tests failures skipped time>` and
  `<testcase name classname time status>` children.
- Test-case status: `status` attribute if present, otherwise `pass`; `<failure>` →
  `fail` (keep the failure text); `skipped` attribute > 0 → `skipped`.
  `<system-out>` → comment.
- Totals: tests, failures, skipped summed over suites; passed = tests − failures −
  skipped; duration = root `time` + sum of suite `time`.
- **Matching a test to a test case:** normalise both sides (lower-case, `_` → space,
  cut at the first `.` for our test name). It matches if it equals the test case's
  `name` or `classname`.

### 8.7 Manual runs
Legacy: `app/Actions/Session/CreateManualSessionRun.php`, `Livewire/Runs/*`.

- Starting a manual run first finishes any `in_progress` manual run of the same
  service (unanswered tests → `skipped`). It then creates a run with
  `type='manual'`, `status='in_progress'`, `started_at=now()` and logs
  `manual_run_started`: "Manual run {id} for {service} started by {user}".
- The wizard shows one test at a time, with prev/next and a "page N / total" input.
  It opens at the first test without a result. Each test shows the parsed
  test case (§8.8), loaded from GitHub at the service's commit SHA. If validation
  fails (e.g. name/file mismatch), a warning shows above it.
- For each test the user chooses **pass** / **fail** / **skip** and may add a comment.
  Saving upserts `run_test_results` and recomputes passed/failed/skipped on the run.
- Saving the last test, or "finish", marks the remaining tests `skipped`, sets
  `status='success'` and `finished_at=now()`, and logs `manual_run_finished`:
  "Finished run {id} for {service} with {p} passed and {f} failed tests".

### 8.8 Manual test-case format
Legacy: `legacy:tests-validator/validate-test-cases.js`, `app/Data/ManualTest*.php`.

```yaml
name: login_with_valid_credentials   # must equal the file name without .manual.yaml
suite: authentication
description: User can log in with valid credentials.
steps:
  - action: [ "Open the login page" ]          # list of strings, required, ≥1
    input:                                       # optional map
      username: { description: "Valid user", value: "qa@example.com" }
    result: [ "Login form is shown" ]           # list of strings, required, ≥1
```

The `internal/testcase` package owns this type, parsing, emitting and validation,
shared by the UI (§8.7), the validator CLI (§8.11) and the generator (§8.12).
Validation errors carry file, field path and message.

### 8.9 Activity, comments and reviews
- Activity list on the session page, newest first, with user name and relative time.
- **Comment:** body required, minimum 5 characters. Logs `comment`.
- **Request review:** the reviewer is chosen from users who have no pending request
  on this session. Creates `review_requests` (pending), emails the reviewer (§8.10),
  and logs `review_requested`: "{requester} requested a review from {reviewer}".
- **Approve/Reject** (pending requests only): sets status and `completed_at`, then logs
  `review_approved` / `review_rejected`: "{user} approved/rejected the review".
- **Revoke:** deletes a pending request. No activity entry, as in legacy (Q4).
- Forms are normal POST forms enhanced with `hx-post`. Without JS they redirect back
  (PRG pattern).

### 8.10 Email
Legacy: `app/Notifications/ReviewRequestNotification.php`.

`internal/mail` defines `Mailer` with two implementations: `smtp` (`net/smtp`,
STARTTLS) and `log` (writes the message to slog, the default for development). The
review request email says "{requester} has requested a review from you." and links to
the session. A failed send is logged and does not fail the request.

### 8.11 `testflow validate` (replaces `legacy:tests-validator`)
```
testflow validate [-ext .manual.yaml] DIR [DIR...]
```
Walks each directory recursively, validates every file ending in `-ext` against
§8.8, prints `checking <file>` for each, then a list of errors and a summary
"files checked: N, files with errors: M". Exit code `0` if there are no errors, `1`
otherwise (legacy exited with the error count). Usable as-is in CI of the tested
repositories.

### 8.12 Test-case generator (replaces `legacy:generate-yaml`)
An authenticated page in the main app (`/tools/test-case`). The form has name, suite,
description and a dynamic list of steps. Each step has action and result (one line
per entry) and optional inputs (name, description, value), added and removed with
htmx. Submitting validates (§8.8) and returns the YAML as a download named
`{name}.manual.yaml`. Nothing is written to the server disk.

### 8.13 Reports
Legacy: `resources/views/sessions/report.blade.php`.

- **Executive:** session name, environment, run list with counts, and a "Print"
  button (`window.print()`, print stylesheet).
- **Full:** executive content, then a page break, then for each run: id, service,
  start time, duration (or "still running"), counts, and for each suite and test its
  status, time (1 decimal) and comment.

## 9. Authentication and security

- **OAuth:** GitHub web flow done by hand with `net/http`. `state` is 32 random bytes
  in a short-lived `HttpOnly` cookie and is compared in constant time. Scope:
  `read:user user:email`. Name/email/login are read from `/user` (and
  `/user/emails` if email is private).
- **Sessions:** server-side (`web_sessions`). Cookie `testflow_session`: `HttpOnly`,
  `SameSite=Lax`, `Secure` unless `ENVIRONMENT=development`, `Path=/`. The id is
  rotated on login.
- **CSRF:** `http.CrossOriginProtection` wraps all non-GET routes. Logout is POST.
- **GitHub App:** the JWT (RS256, `iat`=now−60s, `exp`=now+9m, `iss`=client id) is
  exchanged for an installation token (`POST /app/installations/{id}/access_tokens`).
  The token is cached in memory under a mutex and refreshed 5 min before
  `expires_at`. API calls send `Accept: application/vnd.github+json` and
  `X-GitHub-Api-Version`, use a context with a timeout, and follow
  redirects for log and artifact downloads.
- **Authorization:** any authenticated user can see and act on all sessions (legacy
  parity). Restricting login to members of a GitHub org is open question Q5.
- **Output:** Gomponents escapes all text. User and GitHub content is never inserted
  as raw HTML.
- **Hardening:** `http.Server` timeouts set, request body size limits, response
  headers `Content-Security-Policy` (self only), `X-Content-Type-Options`,
  `Referrer-Policy`.

## 10. Frontend

- Gomponents components in `internal/web/views`: `Layout` (header with user and
  logout, footer), `Section`, `Dialog`, `PassedFailed`, `SessionList`, `TestTree`, etc.,
  ported from `legacy:app/resources/views/components/*`.
- CSS: one hand-written stylesheet ported from `legacy:app/resources/css/app.css`
  (no build step), plus a `print` media block. Font: DM Sans (self-hosted).
- htmx 4 (D4): the version is in the file name, which also busts the browser cache on
  upgrade. The layout loads it with a Subresource Integrity (`integrity`) hash.
  Used for the run-list polling, the manual run wizard (in a `<dialog>`),
  comment/review forms, the test tree "select all" interactions and the generator's
  dynamic steps. Tiny vanilla JS is allowed for `<dialog>` open/close. Nothing else.
- Handlers check the `HX-Request` header. htmx requests get the fragment, normal
  requests get the full page from the same components.

## 11. Observability (D7)

- `internal/telemetry` provides a `slog.Handler` that writes one JSON object per
  line to stdout, with fields aligned to the OTel log data model:
  - `timestamp` (RFC 3339 nano), `severity_text`, `severity_number` (OTel mapping:
    DEBUG 5, INFO 9, WARN 13, ERROR 17), `body` (message);
  - `trace_id`, `span_id` (hex) when the context carries them;
  - `resource`: `service.name=testflow`, `service.version` (set at build time with
    `-ldflags`), `deployment.environment`;
  - all other attributes under `attributes`.
- HTTP middleware reads or creates W3C `traceparent` (stdlib `crypto/rand`), stores
  trace/span ids in the context, and logs one line per request: method, route pattern,
  status, duration, bytes, user id. It returns `traceparent` in the response.
- The syncer starts a new trace per tick, and GitHub calls log at DEBUG with method,
  path, status and duration.
- `deploy/otel-collector.yaml`: `filelog` receiver (container stdout) → `json_parser`
  operator mapping the fields above → `batch` → `otlphttp` exporter (endpoint from env).
- Moving to the in-app OTel SDK later only changes `internal/telemetry`. Call sites
  use plain `slog`.

## 12. Build, run, deploy

- `Makefile` targets: `run`, `test`, `test-integration`, `lint` (`go vet`, `gofmt -l`),
  `build`, `docker`, `update-htmx`.
- **Upgrading htmx is a manual, deliberate change.** Run
  `make update-htmx HTMX_VERSION=x.y.z`. It downloads `dist/htmx.min.js` for that
  exact version from the npm registry tarball, saves it as `htmx-x.y.z.min.js`,
  removes the old file and prints the new SRI hash. In the same PR, update the
  version constant and hash in the layout and the §2 table, then run through the
  htmx-driven screens (§10) before merging. Nothing upgrades htmx automatically
  (Dependabot does not track vendored files).
- `deploy/docker-compose.yaml` for development: `postgres:17`, the app, and the OTel
  Collector (with the `debug` exporter).
- `Dockerfile`: multi-stage. `golang:1.27` builds with `CGO_ENABLED=0`, `-trimpath`,
  and version ldflags. The final image is `gcr.io/distroless/static` and runs as a
  non-root user.
- CI (`.github/workflows/ci.yaml`) on every push and PR: `gofmt` check, `go vet`,
  `go test ./...`, and integration tests with a Postgres service container.
- Image build (`build-image.yaml`) on push to `master`: tags `laposa/testflow:latest`
  and `laposa/testflow:master-YYMMDD.<sha>` (same scheme as legacy
  `.github/workflows/build-app.yaml`).

## 13. Testing

- Unit tests next to the code with the standard `testing` package. Table-driven where
  it fits.
- Golden files in `testdata/` for JUnit parsing, YAML parse/emit, and validator
  output.
- GitHub client tested against `httptest.Server` fixtures (real API response
  shapes saved as JSON). Domain packages use a fake `github` interface.
- Store and integration tests need `TEST_DATABASE_URL` and are skipped when it is
  unset. Each test uses its own schema or transaction.
- Handler tests use `httptest` and check status codes plus key HTML fragments.
- Every iteration's issue lists its acceptance tests. An issue is closed only when
  CI is green.

## 14. Iterations

One GitHub issue per row, done roughly in this order. Each issue copies the template
in §15.

| # | Iteration | Spec | Legacy reference |
|---|-----------|------|------------------|
| 1 | **Scaffold**: `go.mod`, `cmd/testflow`, config, telemetry handler and middleware, `/healthz`, Gomponents layout, static embed, Makefile, Dockerfile, compose, CI | §3–5, §10–12 | `app/resources/views/components/layout.blade.php`, `app/public/fonts/`, `.github/workflows/build-app.yaml` |
| 2 | **Database**: pgxpool, migration runner, `0001_init.sql` (all tables in §6), `testflow migrate`, store test harness | §6 | `app/database/migrations/` |
| 3 | **Login**: GitHub OAuth, users, web sessions, auth middleware, CSRF, login page, logout | §7, §9 | `AuthController.php`, `views/auth/login.blade.php` |
| 4 | **GitHub App client**: JWT, token cache, REST client with all endpoints used later | §9 | `Services/GithubAppAuth.php`, `Services/GithubClient.php` |
| 5 | **Test-case package and validator CLI**: model, parse, validate, `testflow validate` | §8.8, §8.11 | `tests-validator/`, `helpers.php::validateManualTest` |
| 6 | **Discovery and session creation**: test tree page, create session, session detail (read-only) | §8.1, §8.2 | `Actions/Github/FetchTests.php`, `Actions/Session/CreateSession.php`, `views/sessions/create.blade.php`, `components/tests/list-interactive.blade.php` |
| 7 | **Session list / dashboard** | §8.3 | `SessionController::index`, `views/sessions/index.blade.php`, `components/sessions/list.blade.php` |
| 8 | **Activity log and comments** | §6.9, §8.9 | `Livewire/Activity`, `Livewire/Comments` |
| 9 | **JUnit package** | §8.6 | `Services/JUnitLogParser.php` |
| 10 | **Automated runs**: dispatch, syncer, run list fragment and polling, run detail, raw test file | §8.4, §8.5 | `Actions/Github/DispatchSessionRun.php`, `FetchSessionWorkflowRuns.php`, `Console/Commands/SyncGithubRuns.php`, `Livewire/Runs/ListRuns.php`, `views/sessions/run.blade.php` |
| 11 | **Manual runs**: wizard | §8.7 | `Actions/Session/CreateManualSessionRun.php`, `Livewire/Runs/ManualTestRun.php`, `ManualTestForm.php` |
| 12 | **Reviews and email** | §8.9, §8.10 | `Livewire/Reviews/*`, `Notifications/ReviewRequestNotification.php` |
| 13 | **Reports** | §8.13 | `SessionReportController.php`, `views/sessions/report.blade.php` |
| 14 | **Test-case generator** | §8.12 | `generate-yaml/` |
| 15 | **Parity check and cut-over**: walk through the legacy UI side by side, close gaps, deployment docs, README | intent.md "Definition of done" | whole legacy app |

Issues 5 and 9 depend on nothing but the scaffold and can be picked up in parallel.

## 15. Issue template

```markdown
## Goal
<one paragraph: what this iteration delivers and why>

## Spec
Implements spec.md §X.Y (and §…). Any change to the spec is part of this PR.

## Legacy reference (read-only)
- legacy:app/...

## Scope
- [ ] ...
Out of scope: ...

## Acceptance criteria
- [ ] <observable behaviour>
- [ ] Tests: <unit / integration / handler tests that prove it>
- [ ] `make lint test` green in CI
- [ ] No new dependency (or §2 updated with justification)
```

## 16. Open questions

| ID | Question | Default until answered |
|----|----------|------------------------|
| Q1 | Replace the ±5 s timestamp matching by passing a correlation id as a workflow input (e.g. `testflow_run_id`, shown in `run-name`)? This needs a change to the workflows in the tested repositories. | Keep ±5 s matching |
| Q2 | How long should the syncer keep trying runs that never match or never finish? | Give up after 24 h and set status `stale` |
| Q3 | How should `test_filter` be built for services other than `mobile`/`web`/`api`? | Empty filter (runs all) |
| Q4 | Should revoking a review be logged in the activity log? | No (legacy parity) |
| Q5 | Restrict login to members of a GitHub organisation? | No (legacy parity) |
| Q6 | Is `master` always the branch to discover from and dispatch to, or should it be the repo's default branch? | `master` (legacy parity) |
| Q7 | Do we need to migrate historic sessions from the legacy database? | No |
