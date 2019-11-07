#!/usr/bin/env bash
set -e

pack build --builder=heroku/buildpacks $IMAGE

if $PUSH_IMAGE; then
    docker push $IMAGE
fi
