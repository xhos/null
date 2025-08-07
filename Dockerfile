FROM golang:tip-alpine AS builder

RUN apk --no-cache add git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG BUILD_TIME
ARG GIT_COMMIT
ARG GIT_BRANCH

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w \
    -X 'ariand/internal/version.BuildTime=${BUILD_TIME}' \
    -X 'ariand/internal/version.GitCommit=${GIT_COMMIT}' \
    -X 'ariand/internal/version.GitBranch=${GIT_BRANCH}'" \
    -o /app/ariand ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add curl ca-certificates

WORKDIR /app

COPY --from=builder /app/ariand /app/ariand
COPY --from=builder /app/docs ./docs

ENTRYPOINT ["/app/ariand"]
