# TestFlow (Go) build, test and run targets (spec §12). Run from app-go/.
#
# If `go` is not on the PATH, the Go commands run in the $(GO_IMAGE) container
# instead (as your user, with module/build caches in $(GO_CACHE)). Override GO
# and GOFMT to use another toolchain.

# Docker Hardened Image for Go (needs `docker login dhi.io`); keep in sync with
# deploy/Dockerfile.
GO_IMAGE ?= dhi.io/golang:1.27.4-debian13-dev@sha256:28720f1bd4ab3b1963137f78285af22a050c382ef7e5b901397ec84798a3a359
GO_CACHE ?= $(HOME)/.cache/testflow-go

ifeq ($(shell command -v go 2>/dev/null),)
$(shell mkdir -p $(GO_CACHE))
DOCKER_GO = docker run --rm -i --init -u $(shell id -u):$(shell id -g) --network host \
	-v $(CURDIR):/src -w /src -v $(GO_CACHE):/cache \
	-e HOME=/cache -e GOCACHE=/cache/build -e GOMODCACHE=/cache/mod \
	-e CGO_ENABLED -e TEST_DATABASE_URL $(DOCKER_GO_ENV) $(GO_IMAGE)
GO    ?= $(DOCKER_GO) go
GOFMT ?= $(DOCKER_GO) gofmt
# `make run` forwards the variables named in .env into the container. Their
# values come from the shell that sourced .env, so quoting works the same as
# with a local Go (docker --env-file would keep the quotes).
run: DOCKER_GO_ENV = $(shell sed -n 's/^[[:space:]]*\(export[[:space:]]\{1,\}\)\{0,1\}\([A-Za-z_][A-Za-z0-9_]*\)=.*/-e \2/p' .env 2>/dev/null)
else
GO    ?= go
GOFMT ?= gofmt
endif
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
IMAGE   ?= laposa/testflow-go

COMPOSE := docker compose -f deploy/docker-compose.yaml
POSTGRES_PORT ?= 5432
TEST_DATABASE_URL ?= postgres://testflow:testflow@localhost:$(POSTGRES_PORT)/testflow?sslmode=disable

HTMX_DIR := internal/web/static

.PHONY: run test test-integration lint check build docker docker-up docker-down update-htmx clean

## run: start the server locally with settings from .env
run:
	@test -f .env || { echo ".env missing: cp .env.example .env"; exit 1; }
	set -a && . ./.env && set +a && $(GO) run -ldflags "$(LDFLAGS)" ./cmd/testflow serve

## test: unit and handler tests (DB tests are skipped without TEST_DATABASE_URL)
test:
	$(GO) test ./...

## test-integration: all tests against the compose Postgres
test-integration:
	POSTGRES_PORT=$(POSTGRES_PORT) $(COMPOSE) up -d --wait postgres
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" $(GO) test -count=1 ./...

## lint: gofmt (fails on any unformatted file) and go vet
lint:
	@out="$$($(GOFMT) -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	$(GO) vet ./...

## check: everything CI would run, in order (must pass before opening a PR)
check: lint test test-integration

## build: static binary in bin/testflow
build:
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o bin/testflow ./cmd/testflow

## docker: build the image locally as $(IMAGE):$(VERSION) (pushing is manual)
docker:
	docker build -f deploy/Dockerfile --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

## docker-up / docker-down: the full development stack (Postgres, app, collector)
docker-up:
	@test -f .env || { echo ".env missing: cp .env.example .env"; exit 1; }
	VERSION=$(VERSION) $(COMPOSE) up --build

docker-down:
	$(COMPOSE) down

## update-htmx: vendor htmx HTMX_VERSION from the npm registry (manual, see spec §12)
update-htmx:
	@test -n "$(HTMX_VERSION)" || { echo "usage: make update-htmx HTMX_VERSION=x.y.z"; exit 1; }
	curl -fsSL "https://registry.npmjs.org/htmx.org/-/htmx.org-$(HTMX_VERSION).tgz" \
		| tar -xzO package/dist/htmx.min.js > "$(HTMX_DIR)/htmx-$(HTMX_VERSION).min.js.tmp"
	find "$(HTMX_DIR)" -maxdepth 1 -name 'htmx-*.min.js' -delete
	mv "$(HTMX_DIR)/htmx-$(HTMX_VERSION).min.js.tmp" "$(HTMX_DIR)/htmx-$(HTMX_VERSION).min.js"
	@echo
	@echo "Vendored $(HTMX_DIR)/htmx-$(HTMX_VERSION).min.js"
	@echo "Now update HTMXVersion and HTMXIntegrity in internal/web/views/layout.go:"
	@echo "  HTMXVersion   = \"$(HTMX_VERSION)\""
	@echo "  HTMXIntegrity = \"sha384-$$(openssl dgst -sha384 -binary "$(HTMX_DIR)/htmx-$(HTMX_VERSION).min.js" | openssl base64 -A)\""
	@echo "and the version in spec.md §2, then check the htmx screens (spec §10)."

clean:
	rm -rf bin
