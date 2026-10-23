#!/usr/bin/env bash
# Rebuild the web app and the Go server, then run it.
# Extra arguments are forwarded to the binary, e.g. ./scripts/run.sh create-admin
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v bun >/dev/null 2>&1; then
  echo "bun is required but was not found: https://bun.sh" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but was not found: https://go.dev/dl/" >&2
  exit 1
fi

echo "==> Building web"
(cd web && bun install && bun run build)

echo "==> Building server"
go build -o bin/situkang ./cmd/situkang

echo "==> Running on http://localhost:${PORT:-8080} (Ctrl+C to stop)"
exec ./bin/situkang "$@"
