# TestFlow

Optimised workflow for QA teams allowing automated and manual tests execution with audit trail.

## Features

- Manage all automated and manual tests in the same branch as code.
- Run automated tests under the same UI as manual.
- Utilise GitHub actions - using the same tests during custom testing sessions and when executed from CI/CD.
- Allows to audit test session - every step saved in a log, compiled reports for executive overview and full details.
- Clutter free UI.

## Technologies Used

- Go (`net/http`, [Gomponents](https://www.gomponents.com/), [htmx](https://htmx.org/))
- PostgreSQL
- GitHub App and GitHub Actions
- OpenTelemetry (structured logs via the Collector)
- Mocha, Cypress and Maestro in the tested repositories

## Repository

| Path | What it is |
|------|------------|
| [app-go/](app-go/) | The TestFlow web app (Go). Development: [app-go/README.md](app-go/README.md). Deployment: [app-go/deploy/README.md](app-go/deploy/README.md). |
| [app/](app/) | The legacy Laravel app, still built as `laposa/testflow` until it is removed. |
| [generate-yaml/](generate-yaml/), [tests-validator/](tests-validator/) | Node.js tools for manual test cases. |
| [intent.md](intent.md), [spec.md](spec.md) | Why TestFlow exists and how it is built. |

## Installation

TestFlow ships as the container image `laposa/testflow-go`, built from
`app-go/` on every push to `master` (the same workflow run also builds the
legacy `laposa/testflow` image from `app/`). It needs PostgreSQL, a GitHub App installed on the
organisation and an SMTP server. See
[app-go/deploy/README.md](app-go/deploy/README.md).

For local development:

```sh
cd app-go
cp .env.example .env   # fill in the GitHub App settings
docker compose -f deploy/docker-compose.yaml up -d postgres
make run               # http://localhost:8080
make check             # what CI runs
```

CI (`.github/workflows/ci.yaml`) runs `gofmt`, `go vet`, the unit and
integration tests on every pull request and push.
