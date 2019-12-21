#!/usr/bin/env bash
set -e

pack build $IMAGE

if $PUSH_IMAGE; then
    docker push $IMAGE
fi
