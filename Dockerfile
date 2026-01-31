# ----- build -------------------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG GIT_COMMIT

WORKDIR /src

# cache deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# build with cache mounts
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w \
        -X 'null/internal/version.Version=${VERSION:-dev}' \
        -X 'null/internal/version.GitCommit=${GIT_COMMIT:-dev}'" \
        -o /out/null ./cmd/null/main.go

# ----- runtime ----------------------------------------------------------------------------------- 
FROM alpine:3.22.2

# metadata
LABEL org.opencontainers.image.title="null" \
      org.opencontainers.image.source="https://github.com/xhos/null"

# runtime deps
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 1001 -S app && \
    adduser -u 1001 -S -G app -h /app app

# copy binary and migrations
COPY --from=builder --chown=app:app /out/null /app/null
COPY --from=builder --chown=app:app /src/internal/db/migrations /app/internal/db/migrations

USER app
WORKDIR /app

EXPOSE 55555

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:55555/grpc.health.v1.Health/Check \
        -H "Content-Type: application/json" \
        -d "{}" || exit 1

ENTRYPOINT ["/app/null"]
