# syntax=docker/dockerfile:1

# 1. Build the frontend with Bun.
FROM oven/bun:1-alpine AS web
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# 2. Build the Go binary with the built frontend embedded.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/situkang ./cmd/situkang

# 3. Minimal runtime image.
FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 situkang \
    && adduser -S -u 10001 -G situkang situkang \
    && mkdir -p /data \
    && chown situkang:situkang /data

ENV TZ=Asia/Jakarta \
    PORT=8080 \
    DB_PATH=/data/situkang.db

WORKDIR /app
COPY --from=build /out/situkang /app/situkang

USER situkang
EXPOSE 8080
VOLUME ["/data"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/api/health > /dev/null || exit 1

ENTRYPOINT ["/app/situkang"]
