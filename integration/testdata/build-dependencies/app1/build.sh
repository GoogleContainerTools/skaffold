#!/usr/bin/env bash
set -e
# build arg image2 is set by Skaffold to be the image built for app2
docker build -t "$IMAGE" --build-arg image2 .
if [[ "${PUSH_IMAGE}" == "true" ]]; then
    docker push "$IMAGE"
fi
