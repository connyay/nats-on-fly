#! /bin/sh
set -e

sed -i "s/{FLY_APP_NAME}/$FLY_APP_NAME/" /etc/nats.conf

ulimit -n 65535

exec nats-server "$@"