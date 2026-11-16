# TestFlow image (spec §12). Build context is app-go/:
#   docker build -f deploy/Dockerfile --build-arg VERSION=$(git describe --always) .
# or simply `make docker`.
#
# The Go toolchain comes from a Docker Hardened Image (needs `docker login dhi.io`).
# It is pinned by digest; to upgrade, change the tag and digest here and GO_IMAGE
# in the Makefile together (`docker buildx imagetools inspect <image:tag>`).

FROM dhi.io/golang:1.27.4-debian13-dev@sha256:28720f1bd4ab3b1963137f78285af22a050c382ef7e5b901397ec84798a3a359 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/testflow ./cmd/testflow

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/testflow /testflow
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/testflow"]
CMD ["serve"]
