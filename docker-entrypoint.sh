#!/bin/sh
set -eu

# Bind mounts are usually owned by the host user. Fix ownership of the data
# directory, then drop to the unprivileged app user.
if [ "$(id -u)" = "0" ]; then
    chown -R situkang:situkang /data 2>/dev/null || true
    exec su-exec situkang /app/situkang "$@"
fi

exec /app/situkang "$@"
