# TestFlow: Specification

> **How** TestFlow is built. Read [intent.md](intent.md) first for the *why*.
>
> This is a living document. Each iteration may refine it, and a PR that changes
> behaviour described here updates this file in the same PR. GitHub issues refer to
> sections by their anchor (e.g. "implements §6.3").
>
> **Where things live:**
> - **Legacy reference:** [app/](app/), the Laravel web app. Read-only during the rewrite.
>   Legacy paths in this document (with or without a `legacy:` prefix) are
>   relative to `app/`. For example, `app/Services/GithubClient.php` is
>   `app/app/Services/GithubClient.php` in the repo, and short names such as
>   `Livewire/Runs/ListRuns` live under `app/app/`.
> - **New code:** [app-go/](app-go/). Go paths in this document (`cmd/…`,
>   `internal/…`, `deploy/…`, `Makefile`) are relative to `app-go/`.
> - **Out of scope:** [generate-yaml/](generate-yaml/) and
>   [tests-validator/](tests-validator/) stay as Node.js tools and are not rewritten.

---

## 1. Decisions

| ID | Decision | Status |
|----|----------|--------|
| D1 | Go, latest stable (1.27.x at time of writing). `go.mod` sets the minimum. | Accepted |
| D2 | HTTP via `net/http` with the `ServeMux` method and wildcard patterns (`GET /sessions/{id}`). No router library. | Accepted |
| D3 | HTML via `maragu.dev/gomponents`. htmx attributes are written with a small local helper (package `hx` in `internal/web/views/hx/hx.go`: one generic `hx.Attr(name, value)` plus named helpers for the attributes we use), not `gomponents-htmx`. Reason: `gomponents-htmx` (v0.6.1, last released Oct 2024) was built for htmx 2, so its helpers would trail htmx 4 (D4). A local helper of about 30 lines works with any htmx version and adds no module. As of Nov 2026 the repo has had no commits since Oct 2024; htmx 4 support exists only as open issues. **Revisit** once `gomponents-htmx` releases htmx 4 support. Only the helper file would change. | Accepted |
| D4 | htmx **4.0.1**, pinned exactly, vendored as a static file and embedded with `embed`. No CDN, no npm. Upgrades are manual (§12). | Accepted |
| D5 | PostgreSQL via `github.com/jackc/pgx/v5` (`pgxpool`), using pgx directly, not `database/sql`. | Accepted |
| D6 | Migrations: plain `.sql` files, embedded, applied by a small runner of our own. No migration library. | Accepted |
| D7 | Logging: `log/slog` JSON to stdout, using OpenTelemetry log data model field names. An OTel Collector reads stdout and exports over OTLP. No OTel SDK in the binary. | Accepted |
| D8 | YAML via `go.yaml.in/yaml/v3`. | Accepted |
| D9 | No Markdown rendering. | Accepted |
| D10 | One binary with subcommands (`serve`, `migrate`). The Node.js tools `generate-yaml/` and `tests-validator/` are **not** part of the rewrite and stay unchanged. | Accepted |
| D11 | Tests use the standard `testing` package only (no testify). DB tests run against a real Postgres. | Accepted |
| D12 | Manual and automated per-test results are stored in one table (`run_test_results`) instead of keeping manual results as JUnit XML (the legacy approach). | **Proposed** |
| D13 | Activity entries store plain text and structured `metadata`. HTML is produced only by Gomponents views. (Legacy stored HTML in `body`, an XSS risk.) | **Proposed** |
| D14 | GitHub user OAuth tokens are **not** stored. Legacy saved them but never used them. All GitHub API access uses the GitHub App installation token. | **Proposed** |
| D15 | The Go app is a separate Go module in `app-go/` (`module github.com/laposa/testflow/app-go`). Nothing outside `app-go/` changes during the rewrite except the project docs. `app/` keeps building and deploying as today until cut-over (§14, iteration 13). | Accepted |
| D16 | No GitHub workflow changes during the rewrite. `.github/workflows/` stays as it is, and `app-go` is built, tested and checked **manually** with `make` (§12). The workflows are switched to the Go app at cut-over (iteration 13). | Accepted |
| D17 | **No Alpine.js** (or any other client-side framework). Client behaviour uses native HTML first (`<details>/<summary>` for show/hide, `<dialog>` + `showModal()` for modals), then htmx for anything that needs the server, then a few lines of plain JS. Reasons: (1) In legacy, Alpine came bundled with Livewire 3 and did three things, all covered natively or by htmx: show/hide toggles (`app/resources/views/sessions/show.blade.php`, `run.blade.php`), the modal (`components/dialog.blade.php`), and sending events between Livewire components (`livewire/runs/list-runs.blade.php`), which htmx makes unnecessary. (2) Alpine's standard build turns expressions into code at runtime (`new Function`), which needs CSP `'unsafe-eval'` and conflicts with §9. Its CSP build gives up inline expressions, the main reason to use it. (3) The run list is replaced by htmx polling every 30 s, which resets Alpine state unless a morph extension is added: two libraries managing the same DOM. (4) Another vendored asset to pin and upgrade by hand (§12), against the minimal-dependency goal. (5) Logic in `x-data` strings is untyped and not covered by `go test`, while htmx keeps behaviour in handlers the §13 tests cover. **Revisit** if the app gains substantial in-browser state that plain JS can't handle cleanly, such as live search or nested "select all" with counts in the test picker. | Accepted |

## 2. Dependency policy

A third-party module is allowed only if the standard library has no reasonable
equivalent **and** it is listed here with a reason. Adding one means updating this
table in the same PR.

| Module | Why | Transitive deps |
|--------|-----|-----------------|
| `maragu.dev/gomponents` | Type-safe HTML components (D3) | none |
| `github.com/jackc/pgx/v5` | Postgres driver and pool (D5) | `jackc/pgpassfile`, `jackc/pgservicefile`, `jackc/puddle/v2`, `golang.org/x/sync`, `golang.org/x/text` |
| `go.yaml.in/yaml/v3` | Parse `*.manual.yaml` test cases shown in manual runs (D8, §8.8) | none |
| `golang.org/x/text` | Decode JUnit reports that declare a non-UTF-8 encoding (`encoding/htmlindex`, §8.6); `encoding/xml` has no charset support. Already in the build as a dependency of pgx, so it adds no code | none |

Static assets (not Go modules), pinned and upgraded by hand:

| Asset | Version | Source |
|-------|---------|--------|
| `internal/web/static/htmx-4.0.1.min.js` | 4.0.1 | npm package `htmx.org@4.0.1`, `dist/htmx.min.js` |
| DM Sans font files | as in legacy | `app/public/fonts/` |

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

Repository root (only `app-go/` is touched by the rewrite):

```
app/                     legacy Laravel app (read-only reference, still deployed)
app-go/                  the Go rewrite (see below)
generate-yaml/           Node.js tool, out of scope, unchanged
tests-validator/         Node.js tool, out of scope, unchanged
.github/workflows/       build-app.yaml (legacy, unchanged until cut-over, D16)
intent.md, spec.md, README.md
```

Inside `app-go/`:

```
go.mod                   module github.com/laposa/testflow/app-go
Makefile
cmd/testflow/            main.go: subcommand dispatch (serve | migrate)
internal/config/         env parsing and validation
internal/telemetry/      slog handler (OTel field names), trace context, HTTP middleware
internal/db/             pgxpool setup, migration runner
internal/db/migrations/  0001_init.sql, ... (embedded)
internal/db/dbtest/      test harness: isolated schema per test (§13)
internal/store/          SQL queries, one file per aggregate (users.go, test_sessions.go,
                         activity.go: activity log writes and types, ...)
internal/github/         App JWT, installation token cache, REST client
internal/auth/           OAuth login flow, web sessions, middleware
internal/auth/authtest/  in-memory user/session store for tests without Postgres
internal/testcase/       manual test-case model, YAML parsing, file-name check
internal/junit/          JUnit XML parsing and test-name matching
internal/discovery/      repository → service/suite/test tree
internal/runs/           dispatch automated runs, manual run lifecycle, syncer
internal/mail/           SMTP and log mailers
internal/web/            handlers, routes, middleware
internal/web/views/      Gomponents layouts, components, pages
internal/web/views/hx/   htmx attribute helpers (D3)
internal/web/static/     css, fonts, icons, app.js, htmx-x.y.z.min.js (embedded)
deploy/                  Dockerfile, docker-compose.yaml, otel-collector.yaml
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
| `STATIC_DIR` | no | | new. Development only (rejected otherwise): serve `/static/` from this directory instead of the embedded files, so CSS and JS edits show on reload. `.env.example` sets `internal/web/static`; the compose app container clears it |
| `DATABASE_URL` | yes | | `DB_*` |
| `DATABASE_USER`, `DATABASE_PASSWORD` | no | | `DB_USERNAME`, `DB_PASSWORD`. Override the credentials in `DATABASE_URL`, so the password needs no URL-encoding and can come from a separate secret |
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
`id text PK`, `user_id → users ON DELETE CASCADE`, `expires_at timestamptz`,
`created_at`. The cookie carries 32 random bytes (base64url); `id` stores only
their SHA-256 (hex), so the table's contents can't be used to sign in. Expiry
is checked against the app's clock (passed to the queries), not Postgres
`now()`. Expired rows are deleted by the syncer loop.

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
  migrations are pending (unless `-migrate` is passed). `migrate` needs only
  `DATABASE_URL` (plus optional `DATABASE_USER`, `DATABASE_PASSWORD`, `LOG_LEVEL`,
  `ENVIRONMENT`), so it can run as a deploy
  step without the app secrets.
- Runs are serialised with a Postgres advisory lock, so several instances starting
  with `-migrate` at once apply each migration exactly once.
- If the database has versions the binary doesn't know (migrated by a newer release,
  e.g. after a rollback), `serve` logs a warning and starts anyway.
- `updated_at` is maintained by a `BEFORE UPDATE` trigger (`set_updated_at()`), so
  queries don't set it. `activity_log` and `web_sessions` have no `updated_at`.
- `review_requests` has a partial unique index: at most one pending request per
  (session, reviewer).

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
| `GET /sessions/new` | test picker (discovery §8.1). Without fresh cached tests, a loading page whose htmx request (same URL) gets the form fragment. `?refresh` discovers again, then redirects back (htmx: the fragment); if that fails, the page is rendered directly (no redirect, which would start another discovery) | `SessionController::create` |
| `POST /sessions` | create session | `SessionController::store` |
| `GET /sessions/{id}` | session detail: services, runs, activity, reviews. The query may carry the run list's state (§8.5) | `SessionController::show` |
| `GET /sessions/{id}/runs` | htmx fragment: run list (polled); the session page without htmx | `Livewire/Runs/ListRuns` |
| `POST /sessions/{id}/runs` | start runs for selected services | `ListRuns::createRun` |
| `POST /sessions/{id}/runs/sync` | "refresh now": sync the session's runs with GitHub (§8.5) | `ListRuns::refreshSessionWorkflowRuns` |
| `GET /sessions/{id}/runs/{run}` | run detail | `SessionRunController::show` |
| `GET /sessions/{id}/runs/{run}/manual` | htmx fragment: manual run wizard at `?test=N` or the first open test; the wizard page without htmx (§8.7) | `Livewire/Runs/ManualTestRun` |
| `POST /sessions/{id}/runs/{run}/manual/{test}` | save result for one test, return next (the last one finishes the run) | `ManualTestRun::updateResults` |
| `POST /sessions/{id}/runs/{run}/finish` | finish manual run (unanswered → skipped) | `ManualTestRun::markRunAsFinished` |
| `GET /sessions/{id}/runs/{run}/tests/{test}` | raw test file at the run's commit, `text/plain` | `SessionRunController::showTestFile` |
| `POST /sessions/{id}/comments` | add comment | `Livewire/Comments/CreateComment` |
| `POST /sessions/{id}/reviews` | request review | `Livewire/Reviews/RequestReview` |
| `POST /sessions/{id}/reviews/{review}/approve` | approve | `ReviewForm::approve` |
| `POST /sessions/{id}/reviews/{review}/reject` | reject | `ReviewForm::reject` |
| `POST /sessions/{id}/reviews/{review}/revoke` | delete pending request | `RevokeReviewForm::delete` |
| `GET /sessions/{id}/report/executive` | summary report | `SessionReportController::executive` |
| `GET /sessions/{id}/report/full` | full report | `SessionReportController::full` |

Dropped: `GET /github/callback` (points to a controller that does not exist in legacy).

## 8. Behaviour

### 8.1 Test discovery
Legacy: `app/Actions/Github/FetchTests.php`.

1. List repositories of the installation (`GET /installation/repositories`).
2. For each repository, get the latest commit of `master`
   (`GET /repos/{repo}/commits/master`). Its SHA pins everything that follows.
3. Walk the contents API: `tests/{service}/tests/{suite}/{test}`, at most 3 levels.
   Skip entries whose name starts with `_`. Services and suites are directories
   (files such as `tests/README.md` are ignored); every entry of a suite is a test.
   Repositories without `master`, without commits, or without `tests/` are skipped,
   as are services without `tests/` and empty suites.
4. Match each service to the workflow whose file name is `run-tests-{service}.yaml`
   (`GET /repos/{repo}/actions/workflows`), fetched only for repositories with tests.
5. A test is **manual** if its file name contains `manual` (in practice
   `*.manual.yaml`, case-sensitive). Otherwise it is **automated**.
6. Result: tree grouped by repository → service → suite → test, each test carrying
   repository id/name, commit SHA, service/suite/test name and path, workflow id.
   Repositories are sorted by name; the rest keeps GitHub's (alphabetical) order.

An automated test whose service has no workflow cannot be selected (an error icon
explains why). Manual tests never use the workflow, so they stay selectable
(legacy required a workflow for both).

Discovery calls GitHub many times. It runs concurrently per repository (at most 4
at once), and the result is cached in memory for a short TTL (default 5 min, with a
"refresh" button). Legacy fetched on every page load. Only one discovery runs at a
time; requests that arrive meanwhile wait for it and share its result. A discovery
is not cancelled when the request that started it goes away; it is bounded by a
45 s timeout, below the server's write timeout.

**Loading** (legacy: the lazy `CreateSession` Livewire component and
`components/loading-overlay`): clicking "New session" opens `/sessions/new`
straight away. Unless a result younger than the TTL is cached (then the form is
rendered directly), the page shows the "New session" heading under the legacy
loading overlay (the spinning check mark over a white veil), and htmx
(`hx-trigger="load"`) requests the same URL, which answers an htmx request with
only the form block (`#new-session`), errors included, under `Vary: HX-Request`.
The Refresh button does the same with `?refresh`, showing the overlay as its
indicator. Without JS, a `<noscript>` "Load the tests" link and the Refresh link
go to `?refresh`, which discovers and redirects back as before.

If a discovery fails (listing the installation's repositories fails, or the
timeout is hit), the picker falls back to the last successful result while its
snapshot is still retained (see below), with a warning above the tree that the
discovery failed and the list may be outdated (HTTP 200); it can still be
submitted. Without such a result it shows a fixed error and a retry link (HTTP
502). A failed discovery is never cached, so the next request tries again.

If one repository fails (other than the skips above), it is left out and named
in a warning above the tree. If only its workflows fail,
its tree is kept without workflow ids and the warning says its automated tests
cannot be selected.

Every result is a **snapshot** with a random id, kept in memory for 1 h after it
was fetched. The picker form carries the snapshot id and the selected tests' keys
(`{repository id}:{path}`), never repository names, paths or SHAs. `POST /sessions`
looks the keys up in that snapshot, so the client cannot inject other
repositories or paths, and the commit SHA is the one the user saw.

### 8.2 Creating a session
Legacy: `app/Actions/Session/CreateSession.php`.

- Input: `name` (required, trimmed, at most 255 characters), `environment` (one
  of `TEST_SESSION_ENVIRONMENTS`), `discovery` (the snapshot id, §8.1) and at least
  one selected test in `tests` ("Please select at least one test suite.").
- The form has `data-submit-once`: app.js disables its submit button once the
  form is sent, so a double click does not create two sessions (legacy could). A
  page restored from the back/forward cache gets the button back.
- Errors re-render the form (HTTP 422) with the messages above it, keeping the
  name, environment and selection; tree levels that hold selected tests start
  open. If the snapshot has expired, the form is re-rendered from the current
  discovery and asks the user to check the selection. Unknown keys or tests that
  cannot be selected are an error, never silently dropped.
- Selected tests are grouped by type (automated first, then manual), then by
  repository+service, in tree order. Each group becomes a `session_services` row
  (with `branch='master'`, the discovery commit SHA and the service's workflow id,
  NULL if none), its suites, and their tests.
- The session, its services, suites, tests and the `session_created` activity
  ("{user} created the session.") are written in one transaction.
- Redirect to `/sessions/{id}`.

The session page shows the name, environment, issuer and creation time, and the
selected tests in a `<details>` ("Selected tests (n)"), listed by service display
name, suite and test. The run list (§8.5) sits above them, and the comment form
and the activity feed (§8.9) below. Manual runs, reviews and report links are
added by their iterations.

### 8.3 Session list
Legacy: `SessionController::index`, `Session::lastRuns`.

Ordered by id, newest first. For each session, only the **latest run per service**
is aggregated. If a run has `passed>0 || failed>0 || status=='success'`, its counts
are added to passed/failed/skipped. Otherwise the session shows as *running*.

- Served on `GET /` and `GET /sessions`: the "All Sessions" heading with a "New
  session" button, then a table with the columns Name (links to the session),
  Status, Environment (capitalised), Created and Issuer. All sessions are
  listed, without paging (legacy parity). Sessions, issuers and latest runs come
  from one query (`store.ListTestSessions`).
- Status is "No runs yet" without runs. While running, it shows the raw status of
  the newest latest run that has no results (legacy showed the first latest
  run's status, which could be a finished one). Otherwise it shows the totals
  with `PassedFailed`: "3 passed, 1 failed and 2 skipped", leaving out zero
  counts, or "None" if all are zero. Legacy printed "3 passed and and 2 skipped"
  when only failed was zero; this is fixed.
- Created is shown as "2 Jan 2006 15:04" in a `<time>` element, as on the session
  page. Legacy's `d M, Y h:m` showed the month where the minutes belong.
- The legacy filter bar (search, date, environment and issuer) is not ported: its
  inputs were never wired up, and making them work would be a new feature.

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

The dispatch goes to the service's `branch` (always `master`, Q6). Right after
each successful dispatch, insert a `service_runs` row with `status='unknown'`
(its `created_at` is what the run is matched by). Then log one `run_dispatched`
for the batch: "{user} has executed: {service} with run ID {id}, …".

- With no service selected, every service is started (legacy "Run All Tests").
  A selected manual service gets a manual run instead (§8.7); it is not part of
  the `run_dispatched` entry.
- A service GitHub refuses (or one without a workflow) does not stop the others.
  It is named above the run list ("Could not start {service}: GitHub answered
  “{message}”."), with HTTP 502 if nothing started. Legacy failed the whole
  request, leaving the services before it started but unlogged.
- The batch is not cancelled when the request goes away: once one workflow
  runs, the rest and the activity entry should follow.
- Without htmx a successful start redirects to the session (PRG); with htmx the
  run list comes back, with the selection cleared as in legacy.

**Matching a GitHub run to our row:** the dispatch API returns no run id. Legacy
matches the workflow run whose `created_at` is within **±5 s** of our row's
`created_at`. Keep this for parity (see Q1), with two fixes: a run that has a
`github_run_id` is found by it, and an unmatched run takes the *closest*
workflow run in range that no other run has (pending ones, and any run of the
last 24 h with that `github_run_id`), so two dispatches of the same workflow a
few seconds apart no longer both take the first.

**Status** = GitHub `conclusion` if set, otherwise `status`. Values:
`unknown`, `queued`, `in_progress`, `success`, `failure`, `neutral`, `cancelled`,
`skipped`, `timed_out`, `action_required`, `stale`. The column accepts only
these, so other GitHub values are mapped: an unknown conclusion (e.g.
`startup_failure`) is `failure`, and `waiting`/`requested`/`pending` are
`queued`.

**On each sync of a matched run:**
- set `github_run_id`, `commit_sha` (= `head_sha`), and `started_at` = min job `started_at`;
- once GitHub `status == completed`:
  - `result_log`: download the artifact named `test-results` (unless expired)
    and take the first `*.xml` in the zip; parse it (§8.6) to fill
    passed/failed/skipped/duration and `run_test_results` (one row per test of
    the service that `Find` matches; a test without a row shows "not found").
    A report that does not parse is kept in `result_log` without counts; the run
    list and the run page then show the run's status instead of counts, and no
    test is "not found". Without
    the artifact, `result_log` and the counts stay NULL;
  - `run_log`: download the run logs zip and keep the **largest** file. Logs
    GitHub no longer has (404/410) are stored as `''`, so the run stops syncing;
  - `finished_at` = max job `completed_at`;
  - if a download fails, nothing of the sync is stored and the next tick tries
    again, so the status change is logged once, with its counts.
- Files from the zips are read up to 64 MiB. A larger JUnit file is skipped (the
  run has no results); a run log keeps its last 10 MiB behind a note. Both are
  stored as valid UTF-8 without NUL bytes, which Postgres `text` rejects.
- The update is guarded by the status the syncer read (`ErrConflict` if it
  changed) and writes the run, its results and the activity entry in one
  transaction.
- if status changed, log `run_status_changed` (no user: the syncer): "Run {id}
  for {service} has changed status from {old} to {new}", adding "with {p} passed
  and {f} failed tests" when results exist, and a final ".". `metadata.run_id`
  holds the run id; the view adds a "Show run" link (legacy linked the id inside
  its HTML body). For `mobile`, if `run_log` contains a URL starting
  `https://console.mobile.dev/uploads` (legacy: `\S*` after it; here it also stops
  at control characters such as the log's ANSI colour codes, quotes and `<>`,
  and drops trailing punctuation), store it in `metadata.maestro_url`. The
  view renders it as a "Maestro Cloud Console" link.

**Run page** (`GET /sessions/{id}/runs/{run}`, legacy `sessions/run.blade.php`):
the run id and service, start time and duration ("ran for 2 minutes 5 seconds",
from `started_at` or `created_at` to `finished_at`), the status, the counts (zeros
shown) when there is a report, and the service's suites and tests with each
test's result icon (title "Execution time 1.2s") and comment, or "not found".
The JUnit report and the run log are in `<details>` blocks. ANSI escape codes
are stripped from the log (legacy coloured it with the `ansi_up` script). A run
of another session is a 404.

**Raw test file** (`GET /sessions/{id}/runs/{run}/tests/{test}`): the file at the
run's `commit_sha`, or the session's discovery commit before GitHub reported
one, as `text/plain`, cached privately for an hour only when it is the run's own
commit (the fallback is `no-cache`: the same URL serves the run's commit later).
The test must belong to the run's service. A file GitHub
does not have is a 404, other GitHub errors a 502. Test names on the run page
link to it; `app.js` opens it in a `<dialog>` instead (legacy: the `test-dialog`
modal), with the test name as its title, the legacy loading spinner until it arrives and an
error line for a 404 or 502. A response redirected to the login page (expired
sign-in) navigates there, and any other non-`text/plain` answer is an error. The file goes in as `textContent`, never as HTML,
so the endpoint stays `text/plain` (plain `fetch` rather than htmx, which would
need an escaped HTML variant). Escape and the close button (a `method="dialog"`
form) close it, as does a click on the backdrop. Opening another file aborts the
previous request. Without JS, or with Ctrl/Cmd/Shift or a middle click, the link
opens the raw file in a new tab.

### 8.5 Background syncer
Replaces Livewire polling (`wire:poll.30s`) and the `app:sync-github-runs` command.

- A goroutine (`runs.Syncer`) runs at startup and then every `SYNC_INTERVAL`, each
  tick in a new trace. It picks automated runs where `run_log IS NULL`, the status
  is not `stale` and the run is newer than 24 h (open question Q2), groups them by
  (repository, workflow, branch), fetches workflow runs once per group (filtered
  by `event=workflow_dispatch`, the branch and `created>=` the oldest run minus a
  minute), and applies §8.4. A failing group or run is logged and retried on the
  next tick.
- **Stale (Q2):** each tick sets runs older than 24 h that are still `unknown`,
  `queued` or `in_progress` to `stale` and logs the status change. Runs that
  completed but whose log never downloaded keep their status.
- Only one sync runs at a time: a "refresh now" waits for a tick in progress.
- The session page polls the run-list fragment (`hx-trigger="every 30s"`) **only
  while** a run is syncing (the conditions above). The fragment reads the DB and
  never calls GitHub. A run that is syncing shows "Queued" in the duration
  column while no runner has picked it up (`queued`, or `unknown` before GitHub
  reported it) and "Running" once it is `in_progress`; one that ended
  without a report, or went stale, shows its status where the counts would be.
- **Run list state.** The selected services (`services`) and the services whose
  older runs are shown (`expand`, hidden fields) are fields of the run list form.
  The poll sends them (`hx-include`), and the older-runs toggle adds
  `toggle={service}`, so a swap keeps them. Without JS the toggle is a GET to
  `/sessions/{id}` with the same fields, and "Run"/"Refresh now" post the form
  (D17). `app.js` keeps the "Run Selected Tests (n)" / "Run All Tests" label in
  step with the checkboxes.
- **Refresh now** (`POST /sessions/{id}/runs/sync`) syncs that session's pending
  runs straight away. Each repository/workflow/branch group holding one of them
  is matched whole, with every session's pending runs, so the session cannot
  take another session's GitHub run. It is bounded by 45 s and not cancelled
  with the request. If it times out or fails, the run list says the runs keep updating in the
  background (HTTP 200). While it runs, the legacy loading overlay (its
  `relative` variant) covers the run list as the request's htmx indicator.
- The same loop deletes expired `web_sessions`.

### 8.6 JUnit parsing
Legacy: `app/Services/JUnitLogParser.php`. Package `internal/junit`:
`Parse(data)` returns a `Report` with `Totals()` and `Find(testName)`.

- Root `<testsuites>` with `<testsuite name tests failures errors skipped time>`
  and `<testcase name classname time status>` children. A bare `<testsuite>`
  root (some reporters) is read as a single suite; legacy found no suites in
  it. Any other root, or XML that does not parse, is an error. A declared
  non-UTF-8 encoding (ISO-8859-1, windows-1252, …) is decoded; an unknown one
  is an error.
- Nested `<testsuite>`s (mocha and others; legacy dropped them) are flattened,
  parent first, up to 64 levels. A parent's attributes usually total its
  children, so a suite with children is counted from its own test cases only
  (time = their sum) and left out when it has none.
- Parsing is lenient, like legacy's PHP casts: a missing or malformed number
  (also negative, `NaN`, `Inf`) is 0. Counts are capped at 2^30 per suite and
  the totals at 2^31 − 1 (Postgres `integer`); a total duration that overflows
  to infinity is 0. A missing
  count attribute on a suite is counted from its test cases.
- **Test-case status**, in rising precedence:
  1. the `status` attribute, normalised (legacy stored it as is, but
     `run_test_results.status` only accepts §6.8 values, and Maestro writes
     `SUCCESS`/`ERROR`): `fail`/`failed`/`failure` → `fail`;
     `error`/`errored`/`broken` → `error`; `skip`/`skipped`/`pending`/`ignored`/
     `disabled`/`canceled`/`cancelled` → `skipped`; missing or anything else →
     `pass`;
  2. an `<error>` element → `error` (legacy ignored it, so it counted as pass);
  3. a `<failure>` element → `fail`;
  4. a `<skipped>` element (mocha, standard JUnit; legacy ignored it) or a
     `skipped` attribute > 0 → `skipped`.

  The failure message is the text of `<failure>` or `<error>`, or its `message`
  attribute when the text is empty. `<system-out>` → comment. Both are trimmed.
- **Totals:** tests, failures + errors (as failed) and skipped summed over the
  suites; passed = tests − failed − skipped, never below 0. Legacy ignored the
  `errors` attribute, so errors counted as passed. Duration = sum of the suite
  times. (Legacy meant to add the root `time` too, but read it from the wrong
  variable, so it never did; adding it would count the time twice, because
  reporters such as mocha put the overall total on the root.)
- **Matching a test to a test case:** normalise both sides (lower-case, `_` → space,
  cut at the first `.` for our test name). It matches if it equals the test case's
  `name` or `classname`. The first match wins. An empty test name (e.g.
  `.gitkeep`) matches nothing, and an empty `name`/`classname` is never matched.
- Golden tests: `internal/junit/testdata/*.xml` (Cypress/mocha, Maestro, Newman,
  legacy manual, bare suite, nested suites) with `*.golden.json`; `go test ./internal/junit
  -update` rewrites them after a change has been checked by hand.

### 8.7 Manual runs
Legacy: `app/Actions/Session/CreateManualSessionRun.php`, `Livewire/Runs/*`.

- Manual services are started with "Run" like automated ones (§8.4). Starting a
  manual run first finishes any `in_progress` manual run of the same service
  (unanswered tests → `skipped`, logged as below). It then creates a run with
  `type='manual'`, `status='in_progress'`, `started_at=now()`, `commit_sha` = the
  service's commit, and logs `manual_run_started`: "Manual run {id} for {service}
  started by {user}". All of it is one transaction, and the service row is locked
  so two starts at once cannot both leave a run open.
- **Run list.** The latest run of a manual service, while `in_progress`, shows
  "In progress" as its duration and, where the counts would be, the legacy
  buttons "Complete manual tests" and "Mark run as finished" (with a confirm).
  Manual runs are never synced, so they do not make the list poll.
- **Wizard** (`GET /sessions/{id}/runs/{run}/manual`, legacy `ManualTestRun` in a
  modal). "Complete manual tests" loads it with htmx into a `<dialog>` on the
  session page, which `app.js` opens (spinner until it arrives); without JS the
  link opens the same wizard as a page, which the run page also links to. It
  shows one test at a time, with ‹ › links and a "N / total" number field (a GET
  form, `?test=N`, clamped to the range; htmx sends it on change and on Enter, so
  Enter never leaves the session page). It opens at the first test without a
  result (the first test if all have one). Each test shows the parsed test case
  (§8.8), loaded from GitHub at the run's commit through a small in-memory cache
  (a file at a commit never changes; legacy cached each case for an hour). The
  cache holds at most 16 MiB, starting over when full, and skips files over
  1 MiB; "Show file contents" (§8.4) uses it too:
  "suite / name", the description, the steps table (actions, inputs with a copy
  button on each value, expected results), and warnings such as a name/file
  mismatch above it. A file GitHub does not have, or a GitHub error, shows a
  message instead; the result can still be recorded. Every link and form
  replaces `#manual-run-content` with htmx and works without JS.
- For each test the user chooses **pass** / **fail** (radio buttons, required) or
  **Skip** (a submit button, which wins over a chosen result) and may add a
  comment (at most 10 000 characters). The previous result and comment are
  preselected. Per-step pass/fail marks are aids only, never sent: `app.js` picks
  Pass when every step is marked passed and Fail when one is marked failed
  (legacy). Saving (`POST …/manual/{test}`) upserts `run_test_results` and
  recomputes passed/failed/skipped on the run, then shows the next test (htmx) or
  redirects to it. A missing choice or a long comment re-shows the test with the
  message (HTTP 422); a run no longer in progress shows that it is finished
  (HTTP 409). A wizard of an automated run is a 404.
- Saving the last test (its button says "Submit and finish"), or "Mark run as
  finished" (`POST /sessions/{id}/runs/{run}/finish`, which answers with the run
  list), marks the remaining tests `skipped`, sets `status='success'` and
  `finished_at=now()`, and logs `manual_run_finished`: "Finished run {id} for
  {service} with {p} passed and {f} failed tests". Both entries carry
  `metadata.run_id` (a "Show run" link). After the last test the wizard says the
  run is finished and sends `HX-Trigger: manual-run-finished`; `app.js` then closes
  the dialog. Without JS it redirects to the session.
- Closing the dialog sends a `runs-refresh` event to `<body>`, which the run list
  listens for (`hx-trigger="runs-refresh from:body"`, besides the 30 s poll while
  runs sync), so the counts update.
- The run page of a manual run says it is "still in progress" until finished and
  never marks tests "not found".

### 8.8 Manual test-case format
Legacy: `app/Data/ManualTest*.php`, `app/Actions/Github/FetchManualTestContent.php`,
`app/helpers.php::validateManualTest`. The schema itself is defined by the Node.js
`tests-validator/validate-test-cases.js` (repo root, out of scope), which remains
the authoritative validator used in the tested repositories' CI.

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

The `internal/testcase` package owns this type and its parsing, for display in the
manual run wizard (§8.7). `input` keeps the file's order; a plain `name: value`
instead of `{description, value}` is shown as "name: value" without a copy button
(legacy), and a single string is accepted where a list of strings is expected.
Golden tests: `internal/testcase/testdata/*.manual.yaml` with `*.golden.json`. Like legacy, the app only checks that `name` matches the
file name and shows a warning if not. A file that fails to parse shows its raw
content with the parse error instead of blocking the run. Full schema validation
is left to `tests-validator`.

### 8.9 Activity, comments and reviews
- Activity list on the session page, newest first (legacy was oldest first),
  with the comment form above it. Times are relative ("5 minutes ago", months
  of 30 days, years of 365), with the exact time as the tooltip of the `<time>`
  element; legacy showed the raw timestamp. A comment shows "{user} commented"
  above its text. Other entries show the legacy icon of their type (inline SVG
  constants) and their body.
- Entries are plain text (D13). Comments are not rendered as Markdown (D9,
  legacy did); CSS keeps their line breaks.
- **Comment** (`POST /sessions/{id}/comments`): the body is trimmed and must
  have at least 5 characters (counted as characters, not bytes): "The comment
  must be at least 5 characters." Logs `comment` with the user. An error
  re-renders with HTTP 422, keeping the text. On success an htmx request gets
  the activity block back with an empty form, and a normal request is
  redirected to `/sessions/{id}#activity`. `hx-sync="this:drop"` ignores a
  second submit while one is in flight. A missing session is a 404.
  htmx 4 swaps error responses too, so for an htmx request a 404 or 500 sends
  `HX-Retarget: #comment-errors` and `HX-Reswap: innerHTML` with a short
  message ("This session does not exist any more." / "Something went wrong.
  Please reload the page."), instead of the plain error page replacing the
  activity block; the typed comment stays in the form. Responses that depend
  on `HX-Request` send `Vary: HX-Request`. Legacy's
  "Comment created" flash message was never displayed and is not ported.
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

### 8.11 Reports
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
  `/user/emails` if email is private: the primary, verified address; a failure
  there is not fatal). The redirect URI is `BASE_URL/auth/callback`.
  - The state cookie `testflow_oauth_state` (`Path=/auth/callback`, 10 min,
    `SameSite=Lax`) is single-use: the callback always clears it.
  - A failed callback redirects to `/login?error=<code>` (`denied`, `state` or
    `github`). The page maps the code to a fixed message and never echoes the URL.
  - On success the user is upserted by `github_id` (an email GitHub could not
    return this time keeps the stored one), a new session starts, and
    the browser goes to `/`. `GET /login` sends signed-in users to `/`.
- **Sessions:** server-side (`web_sessions`). Cookie `testflow_session`: `HttpOnly`,
  `SameSite=Lax`, `Secure` unless `ENVIRONMENT=development`, `Path=/`. The id is
  rotated on login (the old session row is deleted).
  - Sliding expiry, like Laravel's `SESSION_LIFETIME`: a session expires
    `SESSION_LIFETIME` after the last request. To avoid a write per request,
    `expires_at` and the cookie are renewed only once less than half the
    lifetime is left.
  - An unknown or expired cookie is cleared. Logout deletes the row and clears
    the cookie.
  - Expired sessions are deleted by the syncer loop every `SYNC_INTERVAL` (§8.5).
- **CSRF:** `http.CrossOriginProtection` wraps all non-GET routes. Logout is POST.
  The `BASE_URL` origin is added as a trusted origin, so the `Origin` fallback
  check still passes behind a proxy that rewrites `Host`. Rejected requests get
  403 and a WARN log line.
- **GitHub App:** the JWT (RS256, `iat`=now−60s, `exp`=now+9m, `iss`=client id) is
  exchanged for an installation token (`POST /app/installations/{id}/access_tokens`).
  The token is cached in memory under a mutex and refreshed 5 min before
  `expires_at`. API calls send `Accept: application/vnd.github+json` and
  `X-GitHub-Api-Version`, use a context with a timeout, and follow
  redirects for log and artifact downloads.
  - The key (PKCS#1 as GitHub issues it, or PKCS#8) is parsed when `serve`
    starts. A key that is not a usable RSA key stops startup.
  - Timeouts: 30 s per API call, 5 min per log or artifact download. Downloads
    and raw files are read into memory up to 100 MiB; anything larger is an
    error.
  - Redirects are followed (at most 10). The `Authorization` header is dropped
    when the redirect leaves the API host, so signed storage URLs never get
    the token.
  - A 401 drops the cached token and retries the call once with a new one
    (revoked token, clock skew).
  - List endpoints are fetched with `per_page=100`, following the `Link`
    header's `rel="next"` (at most 50 pages).
  - The latest commit is read with `Accept: application/vnd.github.sha` (bare
    SHA, no diff), raw files with `application/vnd.github.raw+json`. Workflow
    runs can be filtered by `event`, `branch` and `created>=`, so the syncer
    does not page through a workflow's whole history as legacy did.
  - Repository names must be `owner/repo`, and file paths and refs may not
    contain empty, `.` or `..` segments. Segments are URL-escaped.
  - A 404 matches `github.ErrNotFound` (e.g. a repository without `tests/`).
    Other non-2xx responses are a `*github.APIError` with GitHub's `message`.
- **Authorization:** any authenticated user can see and act on all sessions (legacy
  parity). Restricting login to members of a GitHub org is open question Q5.
- **Output:** Gomponents escapes all text. User and GitHub content is never inserted
  as raw HTML.
- **Hardening:** `http.Server` timeouts set, request body size limits, response
  headers `Content-Security-Policy` (self only, except `form-action` also allows
  `https://github.com`, because browsers check it against the redirect after
  `POST /auth/redirect`), `X-Content-Type-Options`,
  `Referrer-Policy`.

## 10. Frontend

- Gomponents components in `internal/web/views`: `Layout` (header with user and
  logout, footer), `Section`, `Dialog`, `PassedFailed`, `SessionList`, `TestTree`, etc.,
  ported from `app/resources/views/components/*`.
- CSS: one hand-written stylesheet ported from `app/resources/css/app.css`
  (no build step), plus a `print` media block. Font: DM Sans (self-hosted).
- htmx 4 (D4): the version is in the file name, which also busts the browser cache on
  upgrade. The layout loads it with a Subresource Integrity (`integrity`) hash, and
  a test checks the hash against the embedded file. htmx 4 injects its request
  indicator CSS at runtime by default. The layout turns this off
  (`<meta name="htmx-config" content='{"includeIndicatorCSS":false}'>`) and
  `app.css` carries those rules instead, so the CSP stays self-only (§9).
  Used for the run-list polling, the manual run wizard (in a `<dialog>`),
  comment/review forms, and expanding older runs per service (`hx-get` loads them).
- No Alpine.js or other client framework (D17). Prefer, in this order:
  1. native HTML: `<details>/<summary>` for show/hide sections (selected tests,
     JUnit results, run log, and each repository/service/suite level of the test
     tree), `<dialog>` + `showModal()` for the manual run wizard
     (Escape, focus trap and backdrop come from the browser);
  2. htmx for anything that needs the server;
  3. a small plain-JS file (`internal/web/static/app.js`, no inline scripts, so it
     fits the CSP in §9) for what's left: opening/closing the dialogs (the manual
     run wizard, §8.7, and the run page's test file, §8.4), closing them on a click
     outside, "select all" checkboxes in the test tree, disabling the submit button
     of a `data-submit-once` form once it is sent (§8.2), the run list's button
     label (§8.5), and in the wizard the step marks that pick Pass/Fail and the
     copy buttons of input values (§8.7). A group
     checkbox (in the level's `<summary>`) checks or clears every test below it
     and shows checked, indeterminate or clear for its tests; it needs no server.
- Handlers check the `HX-Request` header. htmx requests get the fragment, normal
  requests get the full page from the same components.

## 11. Observability (D7)

- `internal/telemetry` provides a `slog.Handler` that writes one JSON object per
  line to stdout, with fields aligned to the OTel log data model:
  - `timestamp` (RFC 3339 nano), `severity_text`, `severity_number` (OTel mapping:
    DEBUG 5, INFO 9, WARN 13, ERROR 17), `body` (message);
  - `trace_id`, `span_id`, `trace_flags` (hex) when the context carries them;
  - `resource`: `service.name=testflow`, `service.version` (set at build time with
    `-ldflags`), `deployment.environment`;
  - all other attributes under `attributes`.
- HTTP middleware reads or creates W3C `traceparent` (stdlib `crypto/rand`), stores
  trace/span ids in the context, and logs one line per request: method, route pattern,
  status, duration, bytes, user id. A panicking handler is logged as status 500 at
  ERROR, then the panic continues to net/http. It returns `traceparent` in the response.
- The syncer starts a new trace per tick, and GitHub calls log at DEBUG with method,
  path, status and duration.
- `deploy/otel-collector.yaml`: `file_log` receiver (container log files, `container`
  operator; non-testflow lines are filtered out) → `json_parser` and `trace_parser`
  operators mapping the fields above → `batch` → `otlphttp` exporter (endpoint from
  `OTLP_ENDPOINT`).
- Moving to the in-app OTel SDK later only changes `internal/telemetry`. Call sites
  use plain `slog`.

## 12. Build, run, deploy

- All Go tooling runs from `app-go/`. `app-go/Makefile` targets: `run`, `test`,
  `test-integration`, `lint` (`go vet`, `gofmt -l`), `check`, `build`, `docker`,
  `docker-up`/`docker-down` (compose stack), `update-htmx`. `GO` and `GOFMT` can
  be overridden. Without a local `go`, the Makefile runs the toolchain in the same
  hardened image as the Dockerfile build stage (`GO_IMAGE`).
- **Manual checks and builds during the rewrite (D16).** No workflow runs for
  `app-go`. `make check` runs everything a CI job would, in order: `gofmt -l`
  (fails on output), `go vet ./...`, `go test ./...`, and `test-integration`
  against the compose Postgres. It must pass locally before a PR is opened. The
  PR description says that it did.
- **Upgrading htmx is a manual, deliberate change.** Run
  `make update-htmx HTMX_VERSION=x.y.z`. It downloads `dist/htmx.min.js` for that
  exact version from the npm registry tarball, saves it as `htmx-x.y.z.min.js`,
  removes the old file and prints the new SRI hash. In the same PR, update the
  version constant and hash in the layout and the §2 table, then run through the
  htmx-driven screens (§10) before merging, including an htmx request with an
  expired session: it must land on `/login` (401 + `HX-Redirect`, §7; htmx 4.0.1
  follows `HX-Redirect` on any status). Nothing upgrades htmx automatically
  (Dependabot does not track vendored files).
- `deploy/docker-compose.yaml` for development: `postgres:17-alpine`, the app, and the OTel
  Collector (`otel/opentelemetry-collector-contrib`, pinned, with the `debug` exporter
  selected by extra `--config` overrides on top of `deploy/otel-collector.yaml`).
- `deploy/Dockerfile`: multi-stage, build context `app-go/`. The Docker Hardened Image
  `dhi.io/golang:1.27.x-debian13-dev`, pinned by digest (needs `docker login dhi.io`), builds
  with `CGO_ENABLED=0`, `-trimpath`, and version ldflags. The final image is
  `gcr.io/distroless/static` and runs as a non-root user.
- `make build` produces `bin/testflow`. `make docker` builds the image locally
  as `laposa/testflow-go:<version>`. Pushing it anywhere is a manual decision. The
  separate name keeps it from clashing with `laposa/testflow`, which the legacy
  `build-app.yaml` keeps publishing from `app/`.
- **At cut-over (iteration 13)** the GitHub workflows are replaced:
  - CI on every push and PR (`working-directory: app-go`) runs the same steps as
    `make check`, with a Postgres service container;
  - the image build on push to `master` publishes `laposa/testflow:latest` and
    `laposa/testflow:master-YYMMDD.<sha>` from `app-go/` (same tag scheme as the
    legacy `build-app.yaml`, which is retired).

## 13. Testing

- Unit tests next to the code with the standard `testing` package. Table-driven where
  it fits.
- Golden files in `testdata/` for JUnit parsing and test-case YAML parsing.
- GitHub client tested against `httptest.Server` fixtures (real API response
  shapes saved as JSON). Domain packages use a fake `github` interface.
- Store and integration tests need `TEST_DATABASE_URL` and are skipped when it is
  unset. Each test uses its own schema or transaction: `dbtest.New(t)` returns a pool
  on a new, migrated schema (`test_<random>`, selected through `search_path`) that is
  dropped when the test ends; `dbtest.NewSchemaURL(t)` gives an empty schema's URL for
  code that opens its own pool.
- Handler tests use `httptest` and check status codes plus key HTML fragments.
- Every iteration's issue lists its acceptance tests. An issue is closed only when
  `make check` passes (run locally until cut-over, D16).

## 14. Iterations

One GitHub issue per row, done roughly in this order. Each issue copies the template
in §15.

Legacy paths are relative to `app/` (see the note at the top).

| # | Iteration | Spec | Legacy reference |
|---|-----------|------|------------------|
| 1 | **Scaffold** in `app-go/`: `go.mod`, `cmd/testflow`, config, telemetry handler and middleware, `/healthz`, Gomponents layout, static embed, Makefile (incl. `check`), Dockerfile, compose. No workflows (D16) | §3–5, §10–12 | `resources/views/components/layout.blade.php`, `public/fonts/` |
| 2 | **Database**: pgxpool, migration runner, `0001_init.sql` (all tables in §6), `testflow migrate`, store test harness | §6 | `database/migrations/` |
| 3 | **Login**: GitHub OAuth, users, web sessions, auth middleware, CSRF, login page, logout | §7, §9 | `Http/Controllers/AuthController.php`, `resources/views/auth/login.blade.php` |
| 4 | **GitHub App client**: JWT, token cache, REST client with all endpoints used later | §9 | `Services/GithubAppAuth.php`, `Services/GithubClient.php` |
| 5 | **Discovery and session creation**: test tree page, create session, session detail (read-only) | §8.1, §8.2 | `Actions/Github/FetchTests.php`, `Actions/Session/CreateSession.php`, `resources/views/sessions/create.blade.php`, `resources/views/components/tests/list-interactive.blade.php` |
| 6 | **Session list / dashboard** | §8.3 | `SessionController::index`, `resources/views/sessions/index.blade.php`, `resources/views/components/sessions/list.blade.php` |
| 7 | **Activity log and comments** | §6.9, §8.9 | `Livewire/Activity`, `Livewire/Comments` |
| 8 | **JUnit package** | §8.6 | `Services/JUnitLogParser.php` |
| 9 | **Automated runs**: dispatch, syncer, run list fragment and polling, run detail, raw test file | §8.4, §8.5 | `Actions/Github/DispatchSessionRun.php`, `Actions/Github/FetchSessionWorkflowRuns.php`, `Console/Commands/SyncGithubRuns.php`, `Livewire/Runs/ListRuns.php`, `resources/views/sessions/run.blade.php` |
| 10 | **Manual runs**: test-case package (§8.8) and wizard | §8.7, §8.8 | `Actions/Session/CreateManualSessionRun.php`, `Actions/Github/FetchManualTestContent.php`, `Data/ManualTest*.php`, `Livewire/Runs/ManualTestRun.php`, `Livewire/Runs/ManualTestForm.php` |
| 11 | **Reviews and email** | §8.9, §8.10 | `Livewire/Reviews/*`, `Notifications/ReviewRequestNotification.php` |
| 12 | **Reports** | §8.11 | `Http/Controllers/SessionReportController.php`, `resources/views/sessions/report.blade.php` |
| 13 | **Parity check and cut-over**: walk through the legacy UI side by side, close gaps, deployment docs, README, replace the GitHub workflows with Go CI and image build (retire `build-app.yaml`) | intent.md "Definition of done", §12 | the whole of `app/`, repo-root `.github/workflows/build-app.yaml` |

Iteration 8 (JUnit) depends only on the scaffold and can be picked up in parallel
with 2–7.

## 15. Issue template

```markdown
## Goal
<one paragraph: what this iteration delivers and why>

## Spec
Implements spec.md §X.Y (and §…). Any change to the spec is part of this PR.

## Legacy reference (read-only, do not modify)
- app/app/...        (repo-relative path)

## Where the code goes
All new code goes in app-go/. Nothing outside app-go/ changes (no workflow changes, D16).

## Scope
- [ ] ...
Out of scope: ...

## Acceptance criteria
- [ ] <observable behaviour>
- [ ] Tests: <unit / integration / handler tests that prove it>
- [ ] `make check` passes locally (stated in the PR)
- [ ] No new dependency (or §2 updated with justification)
```

## 16. Open questions

| ID | Question | Default until answered |
|----|----------|------------------------|
| Q1 | Replace the ±5 s timestamp matching by passing a correlation id as a workflow input (e.g. `testflow_run_id`, shown in `run-name`)? This needs a change to the workflows in the tested repositories. | Keep ±5 s matching |
| Q2 | How long should the syncer keep trying runs that never match or never finish? | Give up after 24 h and set status `stale` (implemented, §8.5) |
| Q3 | How should `test_filter` be built for services other than `mobile`/`web`/`api`? | Empty filter (runs all) |
| Q4 | Should revoking a review be logged in the activity log? | No (legacy parity) |
| Q5 | Restrict login to members of a GitHub organisation? | No (legacy parity) |
| Q6 | Is `master` always the branch to discover from and dispatch to, or should it be the repo's default branch? | `master` (legacy parity) |
| Q7 | Do we need to migrate historic sessions from the legacy database? | No |
