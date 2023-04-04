#!/usr/bin/env bash

set -feuo pipefail

ARGS="-p 4218 \
    --tls \
    --cert /test/redis-tls/redis/cert.pem \
    --key /test/redis-tls/redis/key.pem \
    --cacert /test/redis-tls/minica.pem \
    --user replication-user \
    --pass 435e9c4225f08813ef3af7c725f0d30d263b9cd3"

exec docker compose exec bredis_1 redis-cli $ARGS "${@}"
