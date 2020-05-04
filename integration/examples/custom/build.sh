#!/usr/bin/env bash
set -e

if ! [ -x "$(command -v ko)" ]; then
    pushd $(mktemp -d)
    go mod init tmp; GOFLAGS= go get github.com/google/ko/cmd/ko@v0.4.0
    popd
fi

output=$(ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo $output | tail -n1)

docker tag $ref $IMAGE
if $PUSH_IMAGE; then
    docker push $IMAGE
fi
