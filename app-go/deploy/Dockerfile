# TestFlow image (spec §12). Build context is app-go/:
#   docker build -f deploy/Dockerfile --build-arg VERSION=$(git describe --always) .
# or simply `make docker`.

FROM golang:1.27 AS build
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
