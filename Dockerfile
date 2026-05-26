# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25.1
ARG ALPINE_VERSION=3.22

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -buildid=" -o /out/lanline ./cmd/server
RUN mkdir -p /out/data/uploads /out/logs

FROM alpine:${ALPINE_VERSION} AS runtime

WORKDIR /app

RUN addgroup -S -g 10001 lanline \
    && adduser -S -D -H -u 10001 -G lanline lanline

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder --chown=lanline:lanline /out/lanline /app/lanline
COPY --from=builder --chown=lanline:lanline /src/config /app/config
COPY --from=builder --chown=lanline:lanline /src/web /app/web
COPY --from=builder --chown=lanline:lanline /out/data /app/data
COPY --from=builder --chown=lanline:lanline /out/logs /app/logs

ENV LANLINE_SERVER_HOST=0.0.0.0 \
    LANLINE_SERVER_PORT=8080 \
    LANLINE_DATABASE_DRIVER=sqlite \
    LANLINE_DATABASE_DSN=/app/data/lanline.db \
    LANLINE_LOG_LEVEL=info \
    LANLINE_LOG_FILE=/app/logs/lanline.log

VOLUME ["/app/data", "/app/logs"]
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -T 3 -O /dev/null "http://127.0.0.1:${LANLINE_SERVER_PORT:-8080}/login" || exit 1

USER 10001:10001
ENTRYPOINT ["/app/lanline"]
