#! /bin/sh
set -e

exec /nats-surveyor \
        -s "$NATS_ADDR" \
        -user "$NATS_USER" \
        -password "$NATS_PASSWORD" \
        "$@"
