# ----- build -------------------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG BUILD_TIME
ARG GIT_COMMIT
ARG GIT_BRANCH

# build dependencies
RUN apk add --no-cache git

WORKDIR /src

# cache dependencies
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
    -X 'ariand/internal/version.BuildTime=${BUILD_TIME:-$(date -u +%Y%m%d-%H%M%S)}' \
    -X 'ariand/internal/version.GitCommit=${GIT_COMMIT:-dev}' \
    -X 'ariand/internal/version.GitBranch=${GIT_BRANCH:-main}'" \
    -o /out/ariand ./cmd/ariand/main.go

# ----- grpc_health_probe -------------------------------------------------------------------------
FROM alpine:3.22.1 AS health-probe

ARG TARGETARCH

RUN apk add --no-cache curl && \
    GRPC_HEALTH_PROBE_VERSION=v0.4.40 && \
    ARCH=${TARGETARCH:-amd64} && \
    curl -fsSL "https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-${ARCH}" \
    -o /grpc_health_probe && \
    chmod +x /grpc_health_probe

# ----- runtime ----------------------------------------------------------------------------------- 
FROM alpine:3.22.1

# metadata
LABEL org.opencontainers.image.title="ariand" \
      org.opencontainers.image.description="main gRPC server" \
      org.opencontainers.image.source="https://github.com/xhos/ariand"

# runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -g 1001 -S app && \
    adduser -u 1001 -S -G app -h /app app

# copy binaries and migrations
COPY --from=builder --chown=app:app /out/ariand /app/ariand
COPY --from=builder --chown=app:app /src/internal/ /app/internal/
COPY --from=health-probe --chown=app:app /grpc_health_probe /usr/local/bin/grpc_health_probe

USER app
WORKDIR /app

EXPOSE 55555

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:55555/grpc.health.v1.Health/Check \
        -H "Content-Type: application/json" \
        -d "{}" || exit 1

ENTRYPOINT ["/app/ariand"]
