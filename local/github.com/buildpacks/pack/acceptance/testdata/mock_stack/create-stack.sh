#!/usr/bin/env bash

dir="$(cd $(dirname $0) && pwd)"
os=$(docker info --format '{{json .}}' | jq -r .OSType)

docker build --tag pack-test/build "$dir"/$os/build
docker build --tag pack-test/run "$dir"/$os/run
