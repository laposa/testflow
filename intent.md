# TestFlow: Intent

> This document says **why** TestFlow exists and **why** we are rewriting it.
> It should rarely change. The *how* lives in [spec.md](spec.md); individual
> tasks live in GitHub issues.

## Problem

QA teams run two kinds of tests, automated (Cypress, Mocha, Maestro, run by
GitHub Actions) and manual (step-by-step instructions a person follows). They
usually live in different tools. Results are scattered, manual runs leave no
reliable trail, and release sign-off has no single source of truth.

## Purpose

TestFlow is one place where a QA team can:

- **Keep all tests next to the code**: automated tests and manual test cases
  (`*.manual.yaml`) live in the same repository and branch as the product.
- **Run both kinds from one UI**: pick tests, start a test session against an
  environment, dispatch automated runs to GitHub Actions and walk through manual
  runs step by step.
- **Reuse CI**: a test session uses the same GitHub Actions workflows that run in
  CI/CD.
- **Audit everything**: every action in a session is logged; reviews are requested
  and approved or rejected inside the tool; reports exist for executives (summary)
  and for engineers (full detail).
- **Stay out of the way**: a clutter-free UI.

## Why rewrite

The current web app (PHP / Laravel / Livewire, in [app](app/)) works, but:

- it carries a large dependency tree (Laravel, Livewire, Spatie packages, npm
  toolchains) that has to be patched constantly (see the Dependabot history);
- runtime, build and deployment are heavier than the problem requires.

We are rewriting TestFlow in **Go** to get a single, small, statically compiled
binary with an almost empty dependency list. It should be easy to read, test and
operate for years.

## Goals

1. **Feature parity** with the legacy web app in [app](app/).
2. **Minimal dependencies.** Prefer the Go standard library. Add a third-party module only
   when the standard library has no reasonable equivalent, and record why in `spec.md`.
3. **One binary** that serves the web UI and runs its own maintenance commands
   (database migrations).
4. **Observable**: structured logs in the OpenTelemetry data model, collected by an
   OpenTelemetry Collector.
5. **Incremental delivery**: the rewrite lands in small, reviewable GitHub issues, one
   iteration at a time. Each iteration leaves the app buildable and tested.

## Non-goals

- New features beyond legacy parity. They can come after parity, as separate issues.
- Rewriting the Node.js tools [generate-yaml](generate-yaml/) and
  [tests-validator](tests-validator/). They stay as they are, outside the Go app.
- A JavaScript SPA or a frontend build toolchain. Interactivity comes from htmx.
- Supporting Git hosts other than GitHub.
- Migrating data from the legacy database. TestFlow starts with an empty database.
  *(Revisit if historic sessions turn out to be needed.)*
- Horizontal scaling. A single instance is the target deployment.

## Constraints

- **Language:** Go (latest stable).
- **Backend:** `net/http` and the rest of the standard library.
- **Frontend:** server-rendered HTML with [Gomponents](https://www.gomponents.com/) plus
  [htmx](https://htmx.org/) served as a static file.
- **Database:** PostgreSQL through `jackc/pgx/v5`.
- **Logging:** `log/slog` JSON in the OpenTelemetry log data model, sent on by an
  OpenTelemetry Collector. No OTel SDK inside the app.
- **Legacy code is read-only.** [app](app/) is the reference for behaviour and is not
  modified during the rewrite.
- **New code lives in [app-go](app-go/).** The rewrite does not add or change anything
  outside `app-go/`, except the project docs (`README.md`, `intent.md`, `spec.md`).
- **GitHub workflows stay unchanged until the rewrite is complete.** Until then,
  `app-go` is built and checked manually (`make` targets in `app-go/`).

## Definition of done (for the rewrite)

- Every iteration in `spec.md` §14 is implemented and its issue is closed.
- A QA engineer can do a full cycle (log in, create a session, run automated and
  manual tests, comment, request and receive a review, print a report) without
  touching the legacy app.
- After the rewrite, the GitHub workflows are switched to build the Go binary and
  container image and to run all tests on every push.
- The legacy deployment can be switched off. Removing `app/` from the repository is a
  separate decision after cut-over.
