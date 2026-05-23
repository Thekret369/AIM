# syntax=docker/dockerfile:1

FROM golang:1.25.1-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/aim ./cmd/server
RUN mkdir -p /out/data /out/logs

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /out/aim /app/aim
COPY --from=builder --chown=nonroot:nonroot /src/config /app/config
COPY --from=builder --chown=nonroot:nonroot /src/web /app/web
COPY --from=builder --chown=nonroot:nonroot /out/data /app/data
COPY --from=builder --chown=nonroot:nonroot /out/logs /app/logs

ENV AIM_SERVER_HOST=0.0.0.0 \
    AIM_SERVER_PORT=8080 \
    AIM_DATABASE_DRIVER=sqlite \
    AIM_DATABASE_DSN=/app/data/aim.db \
    AIM_LOG_LEVEL=info \
    AIM_LOG_FILE=/app/logs/aim.log

VOLUME ["/app/data", "/app/logs"]
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/aim"]
