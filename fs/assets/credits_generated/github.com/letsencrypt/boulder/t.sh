#!/usr/bin/env bash
#
# Outer wrapper for invoking test.sh inside docker-compose.
#

if type realpath >/dev/null 2>&1 ; then
  cd "$(realpath -- $(dirname -- "$0"))"
fi

# Use a predictable name for the container so we can grab the logs later
# for use when testing logs analysis tools.
docker rm boulder_tests
exec docker compose run --name boulder_tests boulder ./test.sh "$@"
