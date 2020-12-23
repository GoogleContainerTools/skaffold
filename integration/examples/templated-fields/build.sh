#!/usr/bin/env bash
set -e

if ! [ -x "$(command -v ko)" ]; then
    pushd $(mktemp -d)
    go mod init tmp; GOFLAGS= go get github.com/google/ko/cmd/ko@v0.4.0
    popd
fi

output=$(ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo $output | tail -n1)

img="${IMAGE_REPO}:${IMAGE_TAG}"
docker tag $ref $img
if $PUSH_IMAGE; then
    docker push $img
fi
